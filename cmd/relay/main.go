package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/andyleap/nostr/relay"
	"github.com/andyleap/nostr/relay/eventstore/postgres"
)

func main() {
	pgUser := os.Getenv("PG_USER")
	pgPass := os.Getenv("PG_PASS")
	pgDB := os.Getenv("PG_DB")

	pgConnString := fmt.Sprintf("user=%s dbname=%s password=%s sslmode=disable", pgUser, pgDB, pgPass)

	store, err := postgres.New(pgConnString)
	if err != nil {
		panic(err)
	}

	relay := relay.New(store)

	http.ListenAndServe(":8080", relay)
}
