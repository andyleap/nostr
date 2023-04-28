package relay

import (
	"log"
	"net/http"

	"nhooyr.io/websocket"

	"github.com/andyleap/nostr/common"
	"github.com/andyleap/nostr/proto"
	"github.com/andyleap/nostr/proto/comm"
	"github.com/andyleap/nostr/relay/eventstore"
	"github.com/andyleap/nostr/relay/eventstream"
)

type Relay struct {
	es *eventstream.EventStream

	store eventstore.EventStore

	filters []func(*proto.Event) bool
}

func New(store eventstore.EventStore) *Relay {
	es := eventstream.New()
	go es.Run()
	ch := es.Subscribe("store", make(chan *proto.Event, 100))
	go func() {
		for e := range ch {
			err := store.Add(e)
			if err != nil {
				log.Println("Error storing event", err)
			}
		}
	}()
	return &Relay{
		es:    es,
		store: store,
	}
}

func (r *Relay) AddFilter(f func(*proto.Event) bool) {
	r.filters = append(r.filters, f)
}

func (r *Relay) EventStream() *eventstream.EventStream {
	return r.es
}

func (r *Relay) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	//upgrade to websocket
	conn, err := websocket.Accept(rw, req, nil)
	if err != nil {
		http.Error(rw, "Could not open websocket connection", http.StatusBadRequest)
	}
	ctx := req.Context()
	connID := common.RandID()

	for {
		mt, buf, err := conn.Read(ctx)
		if err != nil {
			conn.Close(websocket.StatusInternalError, "The sky is falling")
			return
		}
		if mt != websocket.MessageText {
			conn.Close(websocket.StatusUnsupportedData, "Can't speak binary")
			return
		}

		req, err := comm.ParseReq(buf)
		if err != nil {
			log.Println("Invalid request", err)
			conn.Close(websocket.StatusUnsupportedData, "Invalid request")
			return
		}

		switch req := req.(type) {
		case *comm.Publish:
			log.Println("Publish", string(buf))
			if !req.Event.CheckSig() {
				log.Println("Invalid signature")
				continue
			}
			deny := false
			for _, f := range r.filters {
				if !f(req.Event) {
					deny = true
					break
				}
			}
			if deny {
				log.Println("Denied by filter")
				continue
			}
			r.es.Publish(req.Event)
		case *comm.Subscribe:
			ch := r.es.Subscribe(connID+"-"+req.ID, nil)
			go func() {
				backfill, err := r.store.Get(req.Filters...)
				if err != nil {
					log.Println("Error getting backfill", err)
					return
				}
				for _, e := range backfill {
					resp := &comm.Event{
						ID:    req.ID,
						Event: e,
					}
					e, _ := resp.MarshalJSON()
					conn.Write(ctx, websocket.MessageText, e)
				}
				for e := range ch {
					good := false
					for _, f := range req.Filters {
						if f.Match(e) {
							good = true
							break
						}
					}
					if !good {
						continue
					}
					resp := &comm.Event{
						ID:    req.ID,
						Event: e,
					}
					e, _ := resp.MarshalJSON()
					conn.Write(ctx, websocket.MessageText, e)
				}
			}()
		case *comm.Close:
			r.es.Unsubscribe(connID + "-" + req.ID)
		}

	}
}
