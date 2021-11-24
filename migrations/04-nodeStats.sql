-- Table: public.nodestats

-- DROP TABLE IF EXISTS public.nodestats;

CREATE TABLE IF NOT EXISTS public.nodestats
(
    node_id character varying(40) COLLATE pg_catalog."default" NOT NULL,
    active boolean,
    syncing boolean,
    mining boolean,
    hashrate integer,
    peers integer,
    gasprice integer,
    uptime integer,
    CONSTRAINT nodestats_pkey PRIMARY KEY (node_id),
    CONSTRAINT fk_node_id FOREIGN KEY (node_id)
        REFERENCES public.nodeinfo (node_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.nodestats
    OWNER to postgres;