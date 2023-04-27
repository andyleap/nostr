package client

import (
	"context"
	"sync"

	"github.com/andyleap/nostr/common"
	"github.com/andyleap/nostr/proto"
	"github.com/andyleap/nostr/proto/comm"
	"nhooyr.io/websocket"
)

type Client struct {
	conn *websocket.Conn

	subs map[string]chan *proto.Event
	mu   sync.Mutex
}

func Dial(ctx context.Context, url string) (*Client, error) {
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return nil, err
	}

	c := &Client{
		conn: conn,
		subs: make(map[string]chan *proto.Event),
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
			ch, ok := c.subs[resp.ID]
			c.mu.Unlock()
			if ok {
				select {
				case ch <- resp.Event:
				default:
					close(ch)
					c.mu.Lock()
					delete(c.subs, resp.ID)
					c.mu.Unlock()
				}
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

func (c *Client) Subscribe(ctx context.Context, filters ...*comm.Filter) (chan *proto.Event, error) {
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
	c.subs[req.ID] = ch
	err = c.conn.Write(ctx, websocket.MessageText, b)
	if err != nil {
		return nil, err
	}
	return ch, nil
}
