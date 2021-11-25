-- Table: public.reorgevents

-- DROP TABLE IF EXISTS public.reorgevents;

CREATE TABLE IF NOT EXISTS public.reorgevents
(
    block_number integer NOT NULL,
    block_hash character varying(66) COLLATE pg_catalog."default" NOT NULL,
    node_id character varying(40) COLLATE pg_catalog."default" NOT NULL,
    CONSTRAINT fk_block_hash FOREIGN KEY (block_hash)
        REFERENCES public.blocks (block_hash) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.reorgevents
    OWNER to postgres;