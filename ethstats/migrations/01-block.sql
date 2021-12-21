
CREATE TABLE IF NOT EXISTS blocks
(
    number integer NOT NULL,
    hash TEXT,
    parent_hash TEXT,
    timestamp numeric NOT NULL,
    miner TEXT,
    gas_used integer NOT NULL,
    gas_limit integer NOT NULL,
    difficulty integer NOT NULL,
    total_difficulty integer NOT NULL,
    transactions_root TEXT,
    transactions_count integer NOT NULL,
    uncles_count integer NOT NULL,
    state_root TEXT,
    CONSTRAINT blocks_pkey PRIMARY KEY (hash)
);

CREATE TABLE IF NOT EXISTS block_transactions
(
    block_hash TEXT REFERENCES blocks(hash),
    txn_hash TEXT
);
