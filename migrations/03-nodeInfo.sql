-- Table: public.nodeinfo

-- DROP TABLE IF EXISTS public.nodeinfo;

CREATE TABLE IF NOT EXISTS public.nodeinfo
(
    name character varying(40) COLLATE pg_catalog."default" NOT NULL,
    node character varying(100) COLLATE pg_catalog."default",
    port integer,
    network character varying(100) COLLATE pg_catalog."default",
    protocol character varying(100) COLLATE pg_catalog."default",
    api character varying(100) COLLATE pg_catalog."default",
    os character varying(100) COLLATE pg_catalog."default",
    osver character varying(100) COLLATE pg_catalog."default",
    client character varying(100) COLLATE pg_catalog."default",
    history boolean,
    CONSTRAINT nodeinfo_pkey PRIMARY KEY (name)
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.nodeinfo
    OWNER to postgres;