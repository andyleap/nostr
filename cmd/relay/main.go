package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/andyleap/nostr/proto"
	"github.com/andyleap/nostr/relay"
	"github.com/andyleap/nostr/relay/eventstore/postgres"
)

func main() {
	pgHost := os.Getenv("PG_HOST")
	pgUser := os.Getenv("PG_USER")
	pgPass := os.Getenv("PG_PASS")
	pgDB := os.Getenv("PG_DB")

	pgConnString := fmt.Sprintf("host=%s user=%s dbname=%s password=%s sslmode=disable", pgHost, pgUser, pgDB, pgPass)

	store, err := postgres.New(pgConnString)
	if err != nil {
		panic(err)
	}

	pubKeysRaw := os.Getenv("PUB_KEYS")
	pubKeys := strings.Split(pubKeysRaw, ",")

	relay := relay.New(store)

	relay.AddFilter(func(e *proto.Event) bool {
		for _, k := range pubKeys {
			if e.PubKey == k {
				return true
			}
		}
		return false
	})

	http.ListenAndServe(":8080", relay)
}
