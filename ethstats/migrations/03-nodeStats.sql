
CREATE TABLE IF NOT EXISTS nodestats
(
    node_id TEXT REFERENCES nodeinfo(node_id),
    active boolean DEFAULT false,
    syncing boolean DEFAULT false,
    mining boolean DEFAULT false,
    hashrate BIGINT DEFAULT 0,
    peers integer DEFAULT 0,
    gasprice BIGINT DEFAULT 0,
    uptime BIGINT DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
