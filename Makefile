-include .env
export

MIGRATION_DIR = migrations
SPEC_DIR = contracts/openapi

COVERAGE_FILE = coverage.out
COVERAGE_PKGS = ./internal/core/application/... ./internal/core/domain/...

GOLANGCI_LINT = go tool golangci-lint

all: infra migrate generate run

infra:
	tripgoctl cluster start
	tripgoctl environment start
	tripgoctl connect

migrate:
	go tool goose -dir $(MIGRATION_DIR) postgres "$(DATABASE_URL)" up

down-migrate:
	go tool goose -dir $(MIGRATION_DIR) postgres "$(DATABASE_URL)" down

generate: client mocks
	go tool oapi-codegen --config $(SPEC_DIR)/oapi-codegen.yaml $(SPEC_DIR)/trip-service.openapi.yaml

client:
	go tool oapi-codegen --config $(SPEC_DIR)/client-codegen.yaml $(SPEC_DIR)/push-service.openapi.yaml

mocks:
	go generate ./...
	go tool mockgen -source=internal/worker/positions/contract.go -destination=internal/worker/positions/mocks_test.go -package=positions

run:
	go run cmd/service/main.go

test:
	go test -count=1 -race -cover ./...
	@go test -count=1 -coverprofile=$(COVERAGE_FILE) $(COVERAGE_PKGS)
	@go tool cover -func=$(COVERAGE_FILE) | tail -1
	@rm -f $(COVERAGE_FILE)

test-integration:
	go test -count=1 -race -timeout 10m ./internal/integrationtests/...

format:
	$(GOLANGCI_LINT) fmt ./...
lint:
	$(GOLANGCI_LINT) run ./...

.PHONY: all infra migrate down-migrate generate client mocks format run test test-integration lint
