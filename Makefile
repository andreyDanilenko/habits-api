SWAG := $(shell go env GOPATH)/bin/swag
docs:
	$(SWAG) init -g cmd/api/main.go -o docs --parseDependency --parseInternal

generate:
	go generate ./cmd/api/...

build: docs
	go build -o bin/api ./cmd/api

# Откат версии миграций до 21 (после замены 022-025 на единую 022)
migrate-force-21:
	go run ./cmd/migrate_force

# Генерация bcrypt-хеша для пароля (для ручного UPDATE в БД)
# Использование: make hash-password PWD='MyPass123!'
hash-password:
	@go run ./scripts/hash-password '$(PWD)'

.PHONY: docs generate build migrate-force-21 hash-password
