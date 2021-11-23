-- Table: public.nodestats

-- DROP TABLE IF EXISTS public.nodestats;

CREATE TABLE IF NOT EXISTS public.nodestats
(
    name character varying(40) COLLATE pg_catalog."default" NOT NULL,
    active boolean,
    syncing boolean,
    mining boolean,
    hashrate integer,
    peers integer,
    gasprice integer,
    uptime integer,
    CONSTRAINT nodestats_pkey PRIMARY KEY (name),
    CONSTRAINT fk_node_name FOREIGN KEY (name)
        REFERENCES public.nodeinfo (name) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.nodestats
    OWNER to postgres;