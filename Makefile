include .env.example
-include .env
export

.PHONY: run generate migrate migrate-down test lint test-integration

run:
	go run ./cmd/service

test:
	go test -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -n 1

test-integration:
	go test -race -count=1 -tags=integration ./internal/repository/...

generate:
	go tool oapi-codegen \
		-generate types,chi-server,spec \
		-include-operation-ids createTrip,getTrip,finishTrip,health,ready \
		-package api \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml
	go generate ./...

migrate:
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DATABASE_URL)" go tool goose -dir migrations up

migrate-down:
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DATABASE_URL)" go tool goose -dir migrations down

lint:
	@golangci-lint run --build-tags integration ./...