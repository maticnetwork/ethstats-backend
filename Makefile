SHELL := /bin/bash

postgresql-test:
	docker run --net=host \
		-e POSTGRES_PASSWORD=password \
		-v $(PWD)/ethstats-data:/var/lib/postgresql/data \
		postgres

postgresql-test-admin:
	docker run --net=host \
		-e PGADMIN_DEFAULT_EMAIL=postgres@gmail.com \
		-e PGADMIN_DEFAULT_PASSWORD=postgres \
		dpage/pgadmin4
