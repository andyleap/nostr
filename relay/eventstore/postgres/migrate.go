package postgres

import (
	"context"
	"database/sql"
	"embed"
	"log"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations
var migrations embed.FS

func migrateDB(ctx context.Context, db *sql.DB) error {
	var version int
	r := db.QueryRow("SELECT version FROM migrations")
	err := r.Scan(&version)
	if err != nil {
		_, err = db.ExecContext(ctx, "CREATE TABLE migrations (version int); INSERT INTO migrations (version) VALUES (0);")
		if err != nil {
			return err
		}
		version = 0
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		vi, _ := strconv.Atoi(strings.Split(entries[i].Name(), ".")[0])
		vj, _ := strconv.Atoi(strings.Split(entries[j].Name(), ".")[0])
		return vi < vj
	})
	log.Println("[DB] Current version:", version)
	for _, entry := range entries {
		parts := strings.Split(entry.Name(), ".")
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		log.Println("[DB] Found version:", v)
		if v <= version {
			continue
		}
		log.Println("[DB] Migrating to version:", v)
		migration, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}
		_, err = db.ExecContext(ctx, string(migration))
		if err != nil {
			return err
		}
		_, err = db.ExecContext(ctx, "UPDATE migrations SET version = $1", v)
		if err != nil {
			return err
		}
		version = v
	}
	return nil
}
