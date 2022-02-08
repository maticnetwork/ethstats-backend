
CREATE TABLE IF NOT EXISTS headevents (
    node_id TEXT REFERENCES nodeinfo(node_id),
    event_id TEXT UNIQUE,
    typ TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS headentry (
    event_id TEXT REFERENCES headevents(event_id) ON DELETE CASCADE,
    block_number integer NOT NULL,
    block_hash TEXT,
    parent_hash TEXT,
    typ TEXT
);
