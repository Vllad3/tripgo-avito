include .env.example
-include .env
export

.PHONY: run generate migrate migrate-down test

run:
	go run ./cmd/trip-service

test:
	go test -race ./...

generate:
	go tool oapi-codegen \
		-generate types,chi-server,spec \
		-include-operation-ids createTrip,getTrip,finishTrip,health,ready \
		-package api \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml

migrate:
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DATABASE_URL)" go tool goose -dir migrations up

migrate-down:
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DATABASE_URL)" go tool goose -dir migrations down