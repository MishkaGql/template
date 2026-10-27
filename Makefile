include .env
export

# DATABASE_URL берется из .env
MIGRATION_DIR=migrations
SPEC_DIR=contracts/openapi

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
	go tool oapi-codegen --config $(SPEC_DIR)/oapi-codegen.yaml	$(SPEC_DIR)/trip-service.openapi.yaml
	
run:
	go run cmd/trip-service/main.go


.PHONY: migrate run generate all infra test lint down-migrate

swagger:
	docker run -d -p 8081:8080 \
      -v ./${SPEC_DIR}:/app/specs \
      -e SWAGGER_JSON=/app/specs/trip-service.openapi.yaml \
      swaggerapi/swagger-ui

test:
	@echo "Нема в 1 лабе"

lint:
	@echo "Нема в 1 лабе"