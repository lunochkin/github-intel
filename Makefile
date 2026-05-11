BIN_DIR ?= bin
GO      ?= go
CMDS    := api ingestor
# Pinned dev watcher: override with GOW=github.com/mitranim/gow@latest etc.
GOW     ?= github.com/mitranim/gow@latest

LDFLAGS ?=

BUILD_TARGETS := $(addprefix $(BIN_DIR)/,$(CMDS))

.PHONY: all help build install fmt vet test test-race tidy clean run-api dev-api run-ingestor run-ingestor-hourly run-ingestor-backfill run-migrate-clickhouse lint web-install web-dev web-build

# Arguments for run-migrate-clickhouse (e.g. make run-migrate-clickhouse MIGRATE_ARGS=version)
MIGRATE_ARGS ?= up
INGESTOR_ARGS ?= data/2015-01-01.json.gz

all: build ## Default: compile cmd binaries into $(BIN_DIR)/

help: ## Print this help
	@grep -hE '^[a-zA-Z0-9_.-]+:.*##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*##"} {printf "%-18s %s\n", $$1, $$2}'

build: $(BUILD_TARGETS) ## Build api and ingestor into $(BIN_DIR)/

$(BIN_DIR)/%:
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $@ ./cmd/$*

install: ## go install both commands (into $(GOBIN) or GOPATH/bin)
	$(GO) install -trimpath -ldflags "$(LDFLAGS)" ./cmd/api ./cmd/ingestor

fmt: ## go fmt all packages
	$(GO) fmt ./...

vet: ## go vet
	$(GO) vet ./...

test: ## go test (short)
	$(GO) test -short ./...

test-race: ## go test with -race
	$(GO) test -race ./...

tidy: ## go mod tidy
	$(GO) mod tidy

clean: ## Remove $(BIN_DIR)/
	rm -rf $(BIN_DIR)

run-api: ## Run cmd/api (pass LISTEN_ADDR=:8800 to override)
	$(GO) run ./cmd/api

dev-api: ## Run cmd/api with file watch (restarts on Go source / module changes)
	$(GO) run $(GOW) -e=go -e=mod -e=sum run ./cmd/api

run-ingestor: ## Run cmd/ingestor (set INGESTOR_ARGS or use run-ingestor-hourly)
	$(GO) run ./cmd/ingestor $(INGESTOR_ARGS)

run-ingestor-hourly: ## Poll GH Archive for current UTC hour file (next published) and ingest once
	$(GO) run ./cmd/ingestor -hourly

# Example: make run-ingestor-backfill BACKFILL_ARGS='-backfill-until=2015-01-02 -backfill-state=./var/backfill.json'
BACKFILL_ARGS ?=
run-ingestor-backfill: ## Sequential GH Archive backfill (see architecure/github-archive-backfill.md)
	$(GO) run ./cmd/ingestor -backfill $(BACKFILL_ARGS)

run-migrate-clickhouse: ## Run cmd/migrate-clickhouse (default: up; use MIGRATE_ARGS=… to override)
	$(GO) run ./cmd/migrate-clickhouse $(MIGRATE_ARGS)

lint: ## golangci-lint run (requires golangci-lint on PATH)
	golangci-lint run ./...

web-install: ## npm install in web/
	cd web && npm install

web-dev: ## Vite dev server (proxies /summary and /healthz to localhost:8800)
	cd web && npm run dev

web-build: ## Production build of web UI into web/dist
	cd web && npm run build
