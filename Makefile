include .env.example
-include .env
export

.PHONY: run generate migrate migrate-down env-debug
run: 
	go run ./cmd/trip-service

generate:
	go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

	go tool oapi-codegen \
	-generate types,chi-server \
	-package api \
	-o internal/generated/api.gen.go \
	contracts/openapi/trip-service.openapi.yaml

migrate:
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DATABASE_URL)" go tool goose -dir migrations up

migrate-down:
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DATABASE_URL)" go tool goose -dir migrations down

env-debug:
	@echo "DATABASE_URL=$(DATABASE_URL)"