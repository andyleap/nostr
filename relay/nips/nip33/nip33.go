package nip33

import (
	"github.com/andyleap/nostr/proto"
	"github.com/andyleap/nostr/proto/comm"
	"github.com/andyleap/nostr/relay"
	"github.com/andyleap/nostr/relay/eventstore"
)

func Attach(r *relay.Relay) {
	r.EventStore().(eventstore.StoreFilterer).AddFilter(func(e *proto.Event) (eventstore.FilterMethod, *comm.Filter) {
		if e.Kind >= 30000 && e.Kind < 40000 {
			d := ""
			for _, t := range e.Tags {
				if t[0] == "d" {
					d = t[1]
					break
				}
			}
			return eventstore.FilterMethodSingle, &comm.Filter{
				Kinds:   []int64{e.Kind},
				Authors: []string{e.PubKey},
				TagFilters: map[string][]string{
					"d": {d},
				},
			}
		}
		return eventstore.FilterMethodNormal, nil
	})
	r.AddNip(33)
}
