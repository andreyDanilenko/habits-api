FROM golang:latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@latest && \
    swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

RUN go build -o /main ./cmd/api/main.go && \
    go build -o /migrate_force ./cmd/migrate_force/main.go

EXPOSE 8080

# Автозапуск migrate_force при старте (откат версии 22-25 → 21). Потом вернуть: CMD ["/main"]
CMD ["/bin/sh", "-c", "/migrate_force 2>/dev/null || true && exec /main"]

# FROM golang:latest

# WORKDIR /app

# COPY go.mod go.sum ./
# RUN go mod download

# COPY . .

# RUN go install github.com/swaggo/swag/cmd/swag@latest && \
#     swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

# RUN go build -o /main ./cmd/api/main.go

# EXPOSE 8080

# CMD ["/main"]
