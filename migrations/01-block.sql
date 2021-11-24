-- Table: public.blocks

-- DROP TABLE IF EXISTS public.blocks;

CREATE TABLE IF NOT EXISTS public.blocks
(
    block_number integer NOT NULL,
    block_hash character varying(66) COLLATE pg_catalog."default" NOT NULL,
    parent_hash character varying(66) COLLATE pg_catalog."default" NOT NULL,
    time_stamp integer NOT NULL,
    miner character varying(42) COLLATE pg_catalog."default" NOT NULL,
    gas_used integer NOT NULL,
    gas_limit integer NOT NULL,
    difficulty integer NOT NULL,
    total_difficulty integer NOT NULL,
    transactions_root character varying(66) COLLATE pg_catalog."default" NOT NULL,
    transactions_count integer NOT NULL,
    uncles_count integer NOT NULL,
    state_root character varying(66) COLLATE pg_catalog."default" NOT NULL,
    node_id character varying(40) COLLATE pg_catalog."default",
    CONSTRAINT blocks_pkey PRIMARY KEY (block_hash)
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.blocks
    OWNER to postgres;