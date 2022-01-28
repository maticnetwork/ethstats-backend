
CREATE TABLE IF NOT EXISTS nodeinfo
(
    node_id TEXT NOT NULL PRIMARY KEY,
    node TEXT,
    port integer,
    network TEXT,
    protocol TEXT,
    api TEXT,
    os TEXT,
    osver TEXT,
    client TEXT,
    history boolean,
    extra_data json,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
