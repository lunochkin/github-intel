BIN_DIR ?= bin
GO      ?= go
CMDS    := api ingestor

LDFLAGS ?=

BUILD_TARGETS := $(addprefix $(BIN_DIR)/,$(CMDS))

.PHONY: all help build install fmt vet test test-race tidy clean run-api run-ingestor run-migrate-clickhouse lint

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

run-api: ## Run cmd/api (pass LISTEN_ADDR=:8080 to override)
	$(GO) run ./cmd/api

run-ingestor: ## Run cmd/ingestor
	$(GO) run ./cmd/ingestor $(INGESTOR_ARGS)

run-migrate-clickhouse: ## Run cmd/migrate-clickhouse (default: up; use MIGRATE_ARGS=… to override)
	$(GO) run ./cmd/migrate-clickhouse $(MIGRATE_ARGS)

lint: ## golangci-lint run (requires golangci-lint on PATH)
	golangci-lint run ./...
