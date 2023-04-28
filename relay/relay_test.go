package relay_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andyleap/nostr/client"
	"github.com/andyleap/nostr/common"
	"github.com/andyleap/nostr/proto"
	"github.com/andyleap/nostr/proto/comm"
	"github.com/andyleap/nostr/relay"
	"github.com/andyleap/nostr/relay/eventstore/memory"
	"github.com/andyleap/nostr/relay/nips/nip09"
	"github.com/andyleap/nostr/relay/nips/nip16"
	"github.com/andyleap/nostr/relay/nips/nip33"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

var (
	relayServer *relay.Relay
	relayClient *client.Client
	privKey     *secp256k1.PrivateKey
)

func TestMain(m *testing.M) {
	pkey := common.GeneratePrivateKey()
	privKey = pkey
	withRelayClient(func(r *relay.Relay, c *client.Client) {
		relayServer = r
		relayClient = c
		m.Run()
	})
}

func TestSign(t *testing.T) {
	e := &proto.Event{
		Kind:    1,
		Content: common.RandID(),
	}
	e.Sign(privKey)
	if !e.CheckSig() {
		buf, _ := json.Marshal(e)
		t.Log(string(buf))
		t.Fatal("invalid signature")
	}
}

func TestSend(t *testing.T) {

	e := &proto.Event{
		Kind:    1,
		Content: common.RandID(),
	}
	e.Sign(privKey)
	id := e.ID

	ch := relayServer.EventStream().Subscribe(e.ID, make(chan *proto.Event, 5))
	relayClient.Publish(context.Background(), e)
found:
	for {
		select {
		case e := <-ch:
			t.Log(e, id)
			if e.ID != id {
				continue
			}
			break found
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestSubscribe(t *testing.T) {
	e := &proto.Event{
		Kind:    1,
		Content: common.RandID(),
	}
	e.Sign(privKey)
	id := e.ID
	ch, err := relayClient.Subscribe(context.Background(), &comm.Filter{
		IDs: []string{id},
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 100)
	relayClient.Publish(context.Background(), e)
	select {
	case e := <-ch:
		t.Log(e)
		if e.ID != id {
			t.Fatal("wrong id")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestComplexFilter(t *testing.T) {
	tag := common.RandID()
	e := &proto.Event{
		Kind:    1,
		Content: common.RandID(),
		Tags: [][]string{
			{"q", "foo"},
			{"q", tag},
		},
	}
	e.Sign(privKey)
	id := e.ID
	ch, err := relayClient.Subscribe(context.Background(), &comm.Filter{
		TagFilters: map[string][]string{
			"q": {tag},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 100)
	relayClient.Publish(context.Background(), e)
	select {
	case e := <-ch:
		t.Log(e)
		if e.ID != id {
			t.Fatal("wrong id")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestBackfill(t *testing.T) {
	e := &proto.Event{
		Kind:    1,
		Content: common.RandID(),
	}
	e.Sign(privKey)
	relayClient.Publish(context.Background(), e)

	time.Sleep(time.Millisecond * 100)

	id := e.ID
	ch, err := relayClient.Subscribe(context.Background(), &comm.Filter{
		IDs:   []string{id},
		Limit: 100,
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case e := <-ch:
		t.Log(e)
		if e.ID != id {
			t.Fatal("wrong id")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestDelete(t *testing.T) {
	e := &proto.Event{
		Kind:    1,
		Content: common.RandID(),
	}
	e.Sign(privKey)
	relayClient.Publish(context.Background(), e)
	time.Sleep(time.Millisecond * 10)

	del := &proto.Event{
		Kind:    5,
		Content: common.RandID(),
		Tags: [][]string{
			{"e", e.ID},
		},
	}
	del.Sign(privKey)
	relayClient.Publish(context.Background(), del)

	time.Sleep(time.Millisecond * 100)

	id := e.ID
	ch, err := relayClient.Subscribe(context.Background(), &comm.Filter{
		IDs:   []string{id},
		Limit: 100,
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case e := <-ch:
		t.Log(e)
		if e.ID == id {
			t.Fatal("deleted event still exists")
		}
	case <-time.After(time.Millisecond * 100):
	}
}

func TestReplacable(t *testing.T) {
	e := &proto.Event{
		Kind:    10000,
		Content: common.RandID(),
	}
	e.Sign(privKey)
	relayClient.Publish(context.Background(), e)
	id := e.ID

	e = &proto.Event{
		Kind:    10000,
		Content: common.RandID(),
	}
	e.Sign(privKey)
	relayClient.Publish(context.Background(), e)

	time.Sleep(time.Millisecond * 100)

	ch, err := relayClient.Subscribe(context.Background(), &comm.Filter{
		IDs:   []string{id},
		Limit: 100,
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case e := <-ch:
		t.Log(e)
		if e.ID == id {
			t.Fatal("replaced event still exists")
		}
	case <-time.After(time.Millisecond * 100):
	}
}

func TestEphemeralStored(t *testing.T) {
	e := &proto.Event{
		Kind:    20000,
		Content: common.RandID(),
	}
	e.Sign(privKey)
	relayClient.Publish(context.Background(), e)
	id := e.ID

	time.Sleep(time.Millisecond * 100)

	ch, err := relayClient.Subscribe(context.Background(), &comm.Filter{
		IDs:   []string{id},
		Limit: 100,
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case e := <-ch:
		t.Log(e)
		if e.ID == id {
			t.Fatal("ephemeral event was stored")
		}
	case <-time.After(time.Millisecond * 100):
	}
}

func TestEphemeralTransmitted(t *testing.T) {
	e := &proto.Event{
		Kind:    20000,
		Content: common.RandID(),
	}
	e.Sign(privKey)
	id := e.ID

	ch, err := relayClient.Subscribe(context.Background(), &comm.Filter{
		IDs:   []string{id},
		Limit: 100,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log("subscribed", id)

	time.Sleep(100 * time.Millisecond)

	err = relayClient.Publish(context.Background(), e)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 1000)

	select {
	case e := <-ch:
		t.Log(e)
		if e.ID == id {
			break
		}
	case <-time.After(time.Millisecond * 100):
		t.Fatal("ephemeral event wasn't transmitted")
	}
}

func withRelayClient(f func(*relay.Relay, *client.Client)) {
	ms := memory.New()
	r := relay.New(ms)
	nip09.Attach(r)
	nip16.Attach(r)
	nip33.Attach(r)
	wsServe := httptest.NewServer(r)
	defer wsServe.Close()
	ctx := context.Background()
	c, err := client.Dial(ctx, wsServe.URL)
	if err != nil {
		panic(err)
	}
	f(r, c)
}
