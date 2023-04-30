package client

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/andyleap/nostr/common"
	"github.com/andyleap/nostr/proto"
	"github.com/andyleap/nostr/proto/comm"
	"nhooyr.io/websocket"
)

type Client struct {
	conn *websocket.Conn

	subs map[string]*Subscription
	mu   sync.Mutex
}

func Dial(ctx context.Context, url string) (*Client, error) {
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return nil, err
	}

	c := &Client{
		conn: conn,
		subs: make(map[string]*Subscription),
	}
	go c.process()
	return c, nil
}

func (c *Client) process() {
	ctx := context.Background()
	for {
		mt, buf, err := c.conn.Read(ctx)
		if err != nil {
			c.Close()
		}
		if mt != websocket.MessageText {
			c.Close()
		}
		resp, err := comm.ParseResp(buf)
		if err != nil {
			c.Close()
		}
		switch resp := resp.(type) {
		case *comm.Event:
			c.mu.Lock()
			sub, ok := c.subs[resp.ID]
			if ok {
				select {
				case sub.ch <- resp.Event:
				default:
					go sub.Close()
				}
			} else {
				closesub := &comm.Close{
					ID: resp.ID,
				}
				b, _ := json.Marshal(closesub)
				c.conn.Write(ctx, websocket.MessageText, b)
			}
			c.mu.Unlock()
		case *comm.EndOfStoredEvents:
			c.mu.Lock()
			sub, ok := c.subs[resp.ID]
			c.mu.Unlock()
			if ok {
				close(sub.backfilling)
			} else {
				closesub := &comm.Close{
					ID: resp.ID,
				}
				b, _ := json.Marshal(closesub)
				c.conn.Write(ctx, websocket.MessageText, b)
			}
		}
	}
}

func (c *Client) Close() error {
	return c.conn.Close(websocket.StatusNormalClosure, "")
}

func (c *Client) Publish(ctx context.Context, e *proto.Event) error {
	req := &comm.Publish{
		Event: e,
	}
	b, err := req.MarshalJSON()
	if err != nil {
		return err
	}
	return c.conn.Write(ctx, websocket.MessageText, b)
}

func (c *Client) Subscribe(ctx context.Context, filters ...*comm.Filter) (*Subscription, error) {
	ch := make(chan *proto.Event, 100)
	req := &comm.Subscribe{
		ID:      common.RandID(),
		Filters: filters,
	}
	b, err := req.MarshalJSON()
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	sub := &Subscription{
		c:           c,
		id:          req.ID,
		ch:          ch,
		backfilling: make(chan struct{}),
	}
	c.subs[req.ID] = sub
	err = c.conn.Write(ctx, websocket.MessageText, b)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

type Subscription struct {
	c           *Client
	id          string
	ch          chan *proto.Event
	backfilling chan struct{}
}

func (s *Subscription) Close() error {
	s.c.mu.Lock()
	defer s.c.mu.Unlock()
	comm := &comm.Close{
		ID: s.id,
	}
	b, err := comm.MarshalJSON()
	if err != nil {
		return err
	}
	err = s.c.conn.Write(context.Background(), websocket.MessageText, b)
	if err != nil {
		return err
	}
	delete(s.c.subs, s.id)
	close(s.ch)
	return nil
}

func (s *Subscription) Backfilling() <-chan struct{} {
	return s.backfilling
}

func (s *Subscription) Events() <-chan *proto.Event {
	return s.ch
}
