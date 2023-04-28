package nip09

import (
	"github.com/andyleap/nostr/proto"
	"github.com/andyleap/nostr/proto/comm"
	"github.com/andyleap/nostr/relay"
)

func Attach(r *relay.Relay) {
	ch := make(chan *proto.Event, 100)
	r.EventStream().Subscribe("nip09", ch)
	go func() {
		for e := range ch {
			switch e.Kind {
			case 5:
				for _, t := range e.Tags {
					if t[0] == "e" {
						r.EventStore().Delete(&comm.Filter{
							IDs:     []string{t[1]},
							Authors: []string{e.PubKey},
						})
					}
				}
			}
		}
	}()
	r.AddNip(9)
}
