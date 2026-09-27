include .env.example
-include .env
export

.PHONY: run generate
run: 
	go run ./cmd/trip-service

generate:
	go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

	go tool oapi-codegen \
	-generate types,chi-server \
	-package api \
	-o internal/generated/api.gen.go \
	contracts/openapi/trip-service.openapi.yaml
