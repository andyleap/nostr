package eventstream

import (
	"sync"

	"github.com/andyleap/nostr/proto"
)

type EventStream struct {
	ch       chan *proto.Event
	watchers map[string]chan<- *proto.Event
	mu       sync.Mutex
}

func New() *EventStream {
	return &EventStream{
		ch:       make(chan *proto.Event),
		watchers: make(map[string]chan<- *proto.Event),
	}
}

func (es *EventStream) Run() {
	deliver := func(e *proto.Event) {
		es.mu.Lock()
		defer es.mu.Unlock()
		for id, ch := range es.watchers {
			select {
			case ch <- e:
			default:
				close(ch)
				delete(es.watchers, id)
			}
		}
	}
	for e := range es.ch {
		deliver(e)
	}
}

func (es *EventStream) Publish(e *proto.Event) {
	es.ch <- e
}

func (es *EventStream) Subscribe(id string, ch chan *proto.Event) <-chan *proto.Event {
	if ch == nil {
		ch = make(chan *proto.Event, 5)
	}
	es.mu.Lock()
	defer es.mu.Unlock()
	es.watchers[id] = ch
	return ch
}

func (es *EventStream) Unsubscribe(id string) {
	es.mu.Lock()
	defer es.mu.Unlock()
	delete(es.watchers, id)
}
