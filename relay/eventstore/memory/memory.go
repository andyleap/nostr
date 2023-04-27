package memory

import (
	"sync"

	"github.com/andyleap/nostr/proto"
	"github.com/andyleap/nostr/proto/comm"
	"github.com/andyleap/nostr/relay/eventstore"
)

type MemoryStore struct {
	events  []*proto.Event
	mu      sync.Mutex
	filters []func(e *proto.Event) (eventstore.FilterMethod, *comm.Filter)
	ch      chan *proto.Event
}

func New() *MemoryStore {
	ms := &MemoryStore{
		ch: make(chan *proto.Event, 10),
	}
	go func() {
		for e := range ms.ch {
			ms.add(e)
		}
	}()
	return ms
}

func (ms *MemoryStore) Add(e *proto.Event) error {
	ms.ch <- e
	return nil
}

func (ms *MemoryStore) add(e *proto.Event) {
	for _, filter := range ms.filters {
		method, f := filter(e)
		if method == eventstore.FilterMethodDrop {
			return
		}
		if method == eventstore.FilterMethodSingle {
			ms.Delete(f)
		}
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.events = append(ms.events, e)
	return
}

func (ms *MemoryStore) Get(filters ...*comm.Filter) ([]*proto.Event, error) {
	if len(filters) == 0 {
		return nil, nil
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	events := []*proto.Event{}
	for _, e := range ms.events {
		match := false
		for _, filter := range filters {
			if filter.Match(e) {
				match = true
				break
			}
		}
		if match {
			events = append(events, e)
		}
	}
	maxLimit := filters[0].Limit
	for _, filter := range filters {
		if filter.Limit > maxLimit {
			maxLimit = filter.Limit
		}
	}
	if len(events) > int(maxLimit) {
		events = events[len(events)-int(maxLimit):]
	}
	return events, nil
}

func (ms *MemoryStore) Delete(filter *comm.Filter) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	newEvents := ms.events[:0]
	for _, e := range ms.events {
		if !filter.Match(e) {
			newEvents = append(newEvents, e)
		}
	}
	ms.events = newEvents
	return nil
}

func (ms *MemoryStore) AddFilter(f func(e *proto.Event) (eventstore.FilterMethod, *comm.Filter)) {
	ms.filters = append(ms.filters, f)
}
