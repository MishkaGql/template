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
	goose -dir $(MIGRATION_DIR) postgres "$(DATABASE_URL)" up

down-migrate:
	goose -dir $(MIGRATION_DIR) postgres "$(DATABASE_URL)" down

generate:
	go tool oapi-codegen --config $(SPEC_DIR)/oapi-codegen.yaml $(SPEC_DIR)/trip-service.openapi.yaml

mocks:
	go generate ./...

run:
	go run cmd/service/main.go

swagger:
	docker run -d -p 8081:8080 \
		-v ./$(SPEC_DIR):/app/specs \
		-e SWAGGER_JSON=/app/specs/trip-service.openapi.yaml \
		swaggerapi/swagger-ui

test:
	go test -count=1 -race -cover ./...
	@go test -count=1 -coverprofile=$(COVERAGE_FILE) $(COVERAGE_PKGS)
	@go tool cover -func=$(COVERAGE_FILE) | tail -1
	@rm -f $(COVERAGE_FILE)

test-integration:
	go test -count=1 -race -timeout 10m -tags integration ./internal/tests/...

lint:
	$(GOLANGCI_LINT) run ./...

.PHONY: all infra migrate down-migrate generate mocks run swagger test test-integration lint
