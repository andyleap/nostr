package nip09

import (
	"github.com/andyleap/nostr/proto"
	"github.com/andyleap/nostr/proto/comm"
	"github.com/andyleap/nostr/relay/eventstore"
	"github.com/andyleap/nostr/relay/eventstream"
)

func Attach(es *eventstream.EventStream, store eventstore.EventStore) {
	ch := make(chan *proto.Event, 100)
	es.Subscribe("nip09", ch)
	go func() {
		for e := range ch {
			switch e.Kind {
			case 5:
				for _, t := range e.Tags {
					if t[0] == "e" {
						store.Delete(&comm.Filter{
							IDs:     []string{t[1]},
							Authors: []string{e.PubKey},
						})
					}
				}
			}
		}
	}()
}
