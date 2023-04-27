BEGIN;

CREATE TABLE IF NOT EXISTS events (
    id CHAR(32) NOT NULL,
    pubkey CHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    kind INT NOT NULL,
    tags JSONB NOT NULL,
    mungedTags JSONB NOT NULL,
    content TEXT NOT NULL,
    sig CHAR(64) NOT NULL,
    PRIMARY KEY (id, pubkey),
    INDEX (kind),
    INDEX (id),
    INDEX (pubkey),
);

COMMIT;