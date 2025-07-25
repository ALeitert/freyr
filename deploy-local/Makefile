PROJ_DIR=$(shell dirname $(shell pwd))
CONFIG_FILE=config.yaml


deploy:
	make pg-run-local


### Postgres ###

PG_IMAGE=postgres:alpine
PG_CONTAINER=freyrDB

PG_PORT=$(shell yq -r '.database.port' $(CONFIG_FILE))
PG_USER=$(shell yq -r '.database.user' $(CONFIG_FILE))
PG_PASSWORD=$(shell yq -r '.database.password' $(CONFIG_FILE))
PG_NAME=$(shell yq -r '.database.name' $(CONFIG_FILE))
PG_URL="postgres://$(PG_USER):$(PG_PASSWORD)@localhost:$(PG_PORT)/$(PG_NAME)?sslmode=disable"

PG_IPADD_LOG='listening on IPv4 address'
PG_IPADD_CMD=docker logs $(PG_CONTAINER) | grep -c $(PG_IPADD_LOG)
PG_READY_CMD=docker exec $(PG_CONTAINER) pg_isready -q

MIGR_DIR=$(PROJ_DIR)/internal/database/sql/migrations

pg-run-local:
#
#	Start container.

	@-docker rm -f $(PG_CONTAINER) >/dev/null 2>&1
	docker run -ti -d \
		-e POSTGRES_USER=$(PG_USER) \
		-e POSTGRES_PASSWORD=$(PG_PASSWORD) \
		-e POSTGRES_DB=$(PG_NAME) \
		-p $(PG_PORT):$(PG_PORT) \
		-v ./.postgres_data:/var/lib/postgresql/data \
		--name $(PG_CONTAINER) \
		$(PG_IMAGE)

#
#	Verify Postgres is running.
#	The checks below cover the case that Postgres start without data and the
#	case that some data already exists.

	timeout 5 bash -c "until [ \$$($(PG_IPADD_CMD)) -ge 1 ]; do sleep 1 ; done"
	timeout 5 bash -c "until $(PG_READY_CMD); do sleep 1 ; done"

#
#	Migrate schema.

	cd migrate; go run main.go --con=$(PG_URL) --mig=$(MIGR_DIR)
