
CREATE TABLE IF NOT EXISTS reorgevents (
    node_id TEXT REFERENCES nodeinfo(node_id),
    reorg_id TEXT UNIQUE
);

CREATE TABLE IF NOT EXISTS reorgentry (
    reorg_id TEXT REFERENCES reorgevents(reorg_id),
    block_number integer NOT NULL,
    block_hash TEXT,
    parent_hash TEXT,
    typ TEXT
);
