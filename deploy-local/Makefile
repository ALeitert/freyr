PROJ_DIR=$(shell dirname $(shell pwd))


deploy:
	kind delete cluster

	$(eval TMP_CONFIG := $(shell mktemp))
	yq eval \
		'.nodes[0].extraMounts[0].hostPath = "$(shell pwd)/.postgres_data"' \
		cluster.yaml \
		> $(TMP_CONFIG)

	kind create cluster --config $(TMP_CONFIG)
	rm $(TMP_CONFIG)

	helm install freyr ./charts -f ./charts/values.yaml
	make migrate-pg


### Postgres Migration ###

VALUES_FILE=./charts/values.yaml

PG_PORT=$(shell yq -r '.postgres.externalPort'   $(VALUES_FILE))
PG_USER=$(shell yq -r '.freyr.database.user'     $(VALUES_FILE))
PG_PASS=$(shell yq -r '.freyr.database.password' $(VALUES_FILE))
PG_NAME=$(shell yq -r '.freyr.database.name'     $(VALUES_FILE))

PG_READY_CMD=kubectl get pod -n freyr -l app=postgres \
	-o jsonpath='{.items[0].status.containerStatuses[0].ready}' \
	2> /dev/null | grep -q true

MIGR_DIR=$(PROJ_DIR)/internal/database/sql/migrations

migrate-pg:
	timeout 60 bash -c "until $(PG_READY_CMD); do sleep 1 ; done"

	$(eval PG_HOST := $(shell \
		kubectl get nodes -o yaml | \
		yq eval '.items[0].status.addresses[0].address' \
	))
	$(eval PG_URL := \
		"postgres://$(PG_USER):$(PG_PASS)@$(PG_HOST):$(PG_PORT)/$(PG_NAME)?sslmode=disable" \
	)

	cd migrate; go run main.go --con=$(PG_URL) --mig=$(MIGR_DIR)
