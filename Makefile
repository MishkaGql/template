include .env
export

# DATABASE_URL берется из .env
MIGRATION_DIR=migrations
SPEC_DIR=contracts/openapi

all: migrate generate run

migrate:
	goose -dir $(MIGRATION_DIR) postgres "$(DATABASE_URL)" up
	
down-migrate:
	goose -dir $(MIGRATION_DIR) postgres "$(DATABASE_URL)" down
	
generate:
	go tool oapi-codegen --config $(SPEC_DIR)/oapi-codegen.yaml	$(SPEC_DIR)/trip-service.openapi.yaml
	
run:
	go run ./cmd/service

.PHONY: all generate down-migrate run spec