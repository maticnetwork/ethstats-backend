SHELL := /bin/bash
POSTGRES_PASSWORD := postgrespassword
PORT := 5432
ADMIN_PORT := 80
PGADMIN_DEFAULT_EMAIL := postgres@example.com

postgresql-test:
	docker run \
		-p $(PORT):$(PORT) \
		-e POSTGRES_PASSWORD=$(POSTGRES_PASSWORD) \
		postgres

postgresql-test-admin:
	docker run \
		-p $(ADMIN_PORT):$(ADMIN_PORT) \
		-e PGADMIN_DEFAULT_EMAIL=$(PGADMIN_DEFAULT_EMAIL) \
		-e PGADMIN_DEFAULT_PASSWORD=$(POSTGRES_PASSWORD) \
		dpage/pgadmin4
