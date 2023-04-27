package eventstore

import (
	"github.com/andyleap/nostr/proto"
	"github.com/andyleap/nostr/proto/comm"
)

type EventStore interface {
	Add(e *proto.Event) error
	Get(filters ...*comm.Filter) ([]*proto.Event, error)
	Delete(*comm.Filter) error
}

type FilterMethod int

const (
	FilterMethodNormal FilterMethod = iota
	FilterMethodDrop
	FilterMethodSingle
)

type StoreFilterer interface {
	EventStore
	//AddFilter adds a filter that allows greater control over how events are stored
	AddFilter(func(e *proto.Event) (FilterMethod, *comm.Filter))
}
