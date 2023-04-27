package nip16

import (
	"github.com/andyleap/nostr/proto"
	"github.com/andyleap/nostr/proto/comm"
	"github.com/andyleap/nostr/relay/eventstore"
)

func Attach(store eventstore.StoreFilterer) {
	store.AddFilter(func(e *proto.Event) (eventstore.FilterMethod, *comm.Filter) {
		if e.Kind >= 20000 && e.Kind < 30000 {
			return eventstore.FilterMethodDrop, nil
		}
		if e.Kind >= 10000 && e.Kind < 20000 {
			return eventstore.FilterMethodSingle, &comm.Filter{
				Kinds:   []int64{e.Kind},
				Authors: []string{e.PubKey},
			}
		}
		return eventstore.FilterMethodNormal, nil
	})
}
