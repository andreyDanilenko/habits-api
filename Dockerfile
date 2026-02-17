FROM golang:latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@latest && \
    swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

RUN go build -o /main ./cmd/api/main.go

EXPOSE 8080

CMD ["/main"]

