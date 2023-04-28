package relay

import (
	"encoding/json"
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

	rd relayData
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
	rd := relayData{
		Name:          "Nostr Relay",
		Description:   "Relay running https://github.com/andyleap/nostr",
		SupportedNIPs: []int{1, 11},
	}

	return &Relay{
		es:    es,
		store: store,
		rd:    rd,
	}
}

func (r *Relay) AddNip(nip int) {
	r.rd.SupportedNIPs = append(r.rd.SupportedNIPs, nip)
}

func (r *Relay) AddFilter(f func(*proto.Event) bool) {
	r.filters = append(r.filters, f)
}

func (r *Relay) EventStream() *eventstream.EventStream {
	return r.es
}

func (r *Relay) EventStore() eventstore.EventStore {
	return r.store
}

/*
{
  "name": <string identifying relay>,
  "description": <string with detailed information>,
  "pubkey": <administrative contact pubkey>,
  "contact": <administrative alternate contact>,
  "supported_nips": <a list of NIP numbers supported by the relay>,
  "software": <string identifying relay software URL>,
  "version": <string version identifier>
}
*/

type relayData struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	PubKey         string `json:"pubkey,omitempty"`
	Contact        string `json:"contact,omitempty"`
	SupportedNIPs  []int  `json:"supported_nips"`
	Software       string `json:"software,omitempty"`
	SoftwareVerion string `json:"version,omitempty"`
}

func (r *Relay) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	//upgrade to websocket
	upgrade := false
	for _, header := range req.Header["Upgrade"] {
		if header == "websocket" {
			upgrade = true
			break
		}
	}

	if !upgrade {
		if req.Header.Get("Accept") == "application/nostr+json" {
			rw.Header().Set("Content-Type", "application/nostr+json")
			buf, _ := json.Marshal(r.rd)
			rw.Write(buf)
			return
		}
		http.Error(rw, "Not a websocket request", http.StatusBadRequest)
	}

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
