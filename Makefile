SWAG := $(shell go env GOPATH)/bin/swag
docs:
	$(SWAG) init -g cmd/api/main.go -o docs --parseDependency --parseInternal

generate:
	go generate ./cmd/api/...

build: docs
	go build -o bin/api ./cmd/api

.PHONY: docs generate build
