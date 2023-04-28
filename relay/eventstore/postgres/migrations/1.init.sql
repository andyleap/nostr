BEGIN;

CREATE TABLE IF NOT EXISTS events (
    id CHAR(64) NOT NULL,
    pubkey CHAR(64) NOT NULL,
    created_at INT NOT NULL,
    kind INT NOT NULL,
    tags JSONB NOT NULL,
    mungedTags JSONB NOT NULL,
    content TEXT NOT NULL,
    sig CHAR(128) NOT NULL,
    PRIMARY KEY (id, pubkey)
);

CREATE INDEX IF NOT EXISTS events_kind ON events (kind);
CREATE INDEX IF NOT EXISTS events_id ON events (id);
CREATE INDEX IF NOT EXISTS events_pubkey ON events (pubkey);

COMMIT;