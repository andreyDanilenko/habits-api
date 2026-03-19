FROM golang:latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@latest && \
    swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

RUN go build -o /main ./cmd/api/main.go && \
    go build -o /seed_crm ./cmd/seed_crm/main.go && \
    go build -o /seed_tasks ./cmd/seed_tasks/main.go && \
    go build -o /seed_users_roles ./cmd/seed_users_roles/main.go

EXPOSE 8080

CMD ["/main"]
