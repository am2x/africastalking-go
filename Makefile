# Tool version used by CI; override locally if you need to.
GOLANGCI_VERSION ?= v2.11.4
COVERPROFILE     ?= coverage.out

.DEFAULT_GOAL := check
.PHONY: help build test test-race cover cover-html fmt fmt-check vet lint tidy tidy-check check clean

help: ## List the available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk -F':.*?## ' '{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

build: ## Compile every package
	go build ./...

test: ## Run the unit tests
	go test ./...

test-race: ## Run the unit tests with the race detector
	go test -race ./...

cover: ## Run test and report total coverage
	go test -race -covermode=atomic -coverprofile=$(COVERPROFILE) ./...
	@go tool cover -func=$(COVERPROFILE) | tail -1

cover-html: cover ## Open the coverage report in a browser
	go tool cover -html=$(COVERPROFILE)

fmt: ## Apply the configured formatters
	golangci-lint fmt

fmt-check: ## Fail if anything is unformatted
	golangci-lint fmt --diff

vet: ## Run go vet
	go vet ./...

# TOOLBIN := $(CURDIR)/.tools/bin

# $(TOOLBIN)/golangci-lint:
# 	GOBIN=$(TOOLBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)

# lint: $(TOOLBIN)/golangci-lint
# 	$(TOOLBIN)/golangci-lint run

lint: ## Run the linters
	golangci-lint run

tidy: ## Tidy go.mod
	go mod tidy

tidy-check: ## Fail if go.mod or go.sum would change
	go mod tidy -diff

check: fmt-check vet lint tidy-check test-race ## Everything CI runs
	@echo "ok"

clean: ## Remove build and coverage output.
	rm -r $(COVERPROFILE)
	go clean -testcache