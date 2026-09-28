include .env.example
-include .env
export

.PHONY: run generate migrate migrate-down

run:
	go run ./cmd/trip-service

generate:
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml

migrate:
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DATABASE_URL)" go tool goose -dir migrations up

migrate-down:
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DATABASE_URL)" go tool goose -dir migrations down