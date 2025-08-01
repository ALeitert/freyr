PROJ_DIR=$(shell pwd)


run:
	go run cmd/main.go


### Linting ###

check-golangci-lint:
	$(eval GOLANGCI_LINT_VERSION=$(shell curl -s https://api.github.com/repos/golangci/golangci-lint/releases/latest | jq -r '.name'))
	@./bin/golangci-lint --version | grep -qF "$(GOLANGCI_LINT_VERSION:1)" || { \
		echo "golangci-lint not found or version mismatch. Installing..."; \
		$(MAKE) install-golangci-lint; \
	}

install-golangci-lint: ## install golang lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s latest

lint-check: check-golangci-lint
	$(PROJ_DIR)/bin/golangci-lint run --config $(PROJ_DIR)/.golangci.yaml


### Unit Tests ###

ut:
	go test -parallel=1 -race ./...


### SQL ###

DB_DIR=$(PROJ_DIR)/internal/database

gen-sql: gen-clean-sql
	echo $(PROJ_DIR)
	docker run --rm -v $(DB_DIR):/src -w /src/sql sqlc/sqlc generate

gen-clean-sql:
	@cd $(DB_DIR)/querier && rm -f *gen.go


### Build and Deploy ###

codegen:
	make gen-sql
	go mod tidy

codegen-check:
	git restore :/ && git clean -d -f
	make codegen
	git diff --exit-code

build-docker: codegen
	docker build -f ./Dockerfile . -t freyr

deploy-local: build-docker
	cd deploy-local; make deploy
