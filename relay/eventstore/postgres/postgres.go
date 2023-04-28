package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/andyleap/nostr/proto"
	"github.com/andyleap/nostr/proto/comm"
	"github.com/andyleap/nostr/relay/eventstore"
	"github.com/lib/pq"
)

type addReq struct {
	e *proto.Event
	c chan error
}

type PostgresStore struct {
	conn    *sql.DB
	filters []func(e *proto.Event) (eventstore.FilterMethod, *comm.Filter)
	ch      chan addReq
}

func New(connStr string) (*PostgresStore, error) {
	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	err = migrateDB(context.Background(), conn)
	if err != nil {
		return nil, err
	}

	ch := make(chan addReq, 10)
	ps := &PostgresStore{
		conn: conn,
		ch:   ch,
	}
	go func() {
		for ar := range ch {
			ps.add(ar)
		}
	}()

	return ps, nil
}

func (ps *PostgresStore) Add(e *proto.Event) error {
	eCh := make(chan error)
	ps.ch <- addReq{
		e: e,
		c: eCh,
	}
	return <-eCh
}

/*

CREATE TABLE IF NOT EXISTS events (
    id CHAR(32) NOT NULL,
    pubkey CHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    kind INT NOT NULL,
    tags JSONB NOT NULL,
    content TEXT NOT NULL,
    sig CHAR(64) NOT NULL,
    PRIMARY KEY (id, pubkey),
    INDEX (kind),
    INDEX (id),
    INDEX (pubkey),
);

*/

func (ps *PostgresStore) add(ar addReq) {
	defer close(ar.c)
	e := ar.e
	for _, filter := range ps.filters {
		method, f := filter(e)
		if method == eventstore.FilterMethodDrop {
			return
		}
		if method == eventstore.FilterMethodSingle {
			ps.Delete(f)
		}
	}
	tagBuf, _ := json.Marshal(e.Tags)
	mungedTags := map[string][]string{}
	for _, v := range e.Tags {
		if len(v) < 2 {
			continue
		}
		if len(v[1]) != 1 {
			continue
		}
		mungedTags[v[0]] = append(mungedTags[v[0]], v[1])
	}
	mungedTagsBuf, _ := json.Marshal(mungedTags)
	_, err := ps.conn.Exec("INSERT INTO events (id, pubkey, created_at, kind, tags, mungedTags, content, sig) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", e.ID, e.PubKey, e.CreatedAt, e.Kind, tagBuf, mungedTagsBuf, e.Content, e.Sig)
	ar.c <- err
	return
}

/*
	IDs     []string
	Authors []string
	Kinds   []int64
	Since   int64
	Until   int64
	Limit   int64

	TagFilters map[string][]string
*/

func buildWhereClause(filters ...*comm.Filter) (string, []interface{}) {
	query := "WHERE ("
	args := []interface{}{}
	for i, filter := range filters {
		if i != 0 {
			query += ") OR ("
		}
		sep := ""
		if len(filter.IDs) > 0 {
			query += fmt.Sprintf("id = ANY($%d)", len(args)+1)
			args = append(args, pq.StringArray(filter.IDs))
			sep = " AND "
		}
		if len(filter.Authors) > 0 {
			query += sep + fmt.Sprintf("pubkey = ANY($%d)", len(args)+1)
			args = append(args, pq.StringArray(filter.Authors))
			sep = " AND "
		}
		if len(filter.Kinds) > 0 {
			query += sep + fmt.Sprintf("kind = ANY($%d)", len(args)+1)
			args = append(args, pq.Int64Array(filter.Kinds))
			sep = " AND "
		}
		if filter.Since > 0 {
			query += sep + fmt.Sprintf("created_at >= $%d", len(args)+1)
			args = append(args, filter.Since)
			sep = " AND "
		}
		if filter.Until > 0 {
			query += sep + fmt.Sprintf("created_at <= $%d", len(args)+1)
			args = append(args, filter.Until)
			sep = " AND "
		}
		if len(filter.TagFilters) > 0 {
			query += sep + "("
			subsep := ""
			for k, vals := range filter.TagFilters {
				for _, v := range vals {
					query += subsep + fmt.Sprintf("tags->'%s' @> $%d", k, len(args)+1)
					args = append(args, v)
					subsep = " AND "
				}
			}
			query += ")"
		}
	}
	query += ")"
	return query, args
}

func (ps *PostgresStore) Get(filters ...*comm.Filter) ([]*proto.Event, error) {
	if len(filters) == 0 {
		return nil, nil
	}
	query := "SELECT id, pubkey, created_at, kind, tags, content, sig FROM events "

	where, args := buildWhereClause(filters...)
	query += where + " ORDER BY created_at DESC"

	limit := int(filters[0].Limit)
	for _, filter := range filters {
		if int(filter.Limit) > limit {
			limit = int(filter.Limit)
		}
	}
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := ps.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ret := []*proto.Event{}
	for rows.Next() {
		e := &proto.Event{}
		tagraw := []byte{}

		rows.Scan(&e.ID, &e.PubKey, &e.CreatedAt, &e.Kind, &tagraw, &e.Content, &e.Sig)
		json.Unmarshal(tagraw, &e.Tags)
		ret = append(ret, e)
	}
	//reverse ret so that it's in chronological order
	for i := len(ret)/2 - 1; i >= 0; i-- {
		opp := len(ret) - 1 - i
		ret[i], ret[opp] = ret[opp], ret[i]
	}
	return ret, nil
}

func (ps *PostgresStore) Delete(filter *comm.Filter) error {
	query := "DELETE FROM events "

	where, args := buildWhereClause(filter)
	query += where

	_, err := ps.conn.Exec(query, args...)
	return err
}

func (ps *PostgresStore) AddFilter(f func(e *proto.Event) (eventstore.FilterMethod, *comm.Filter)) {
	ps.filters = append(ps.filters, f)
}
