# Habits API (backend for habits.lifedream.tech)

REST API for the Habits ERP module. **Live:** [habits.lifedream.tech](https://habits.lifedream.tech)

**Deploy (all projects):** [deployment](../deployment/README.md) · **Frontend:** [habits-client](../frontend/README.md)

## Stack

| Layer | Tech |
|------|------|
| Backend | Go, Gin, PostgreSQL |

- **Go** — language
- **Gin** — web framework
- **godotenv** — env loading

## Структура проекта

```
backend/
├── cmd/              # Точки входа для разных сервисов
│   └── api/          # API сервис
│       └── main.go   # Точка входа
├── internal/         # Внутренний код приложения
│   ├── app/          # Инициализация приложения
│   ├── config/       # Конфигурация
│   ├── handler/      # HTTP обработчики
│   ├── router/       # Роутинг
│   ├── service/      # Бизнес-логика
│   ├── repository/   # Доступ к данным
│   └── model/        # Доменные модели
└── pkg/              # Публичные пакеты (переиспользуемые)
    ├── logger/       # Логирование
    └── utils/        # Утилиты
```

## Установка

```bash
go mod download
```

## Запуск

```bash
go run cmd/api/main.go
```

Или соберите и запустите:

```bash
go build -o bin/backend ./cmd/api
./bin/backend
```

Server at `http://localhost:8080`. **Deploy:** full stack is deployed from the [deployment](../deployment/README.md) repo (`habits-api` + `habits` frontend + Nginx).

## Переменные окружения

Создайте файл `.env`:

```env
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=habits_db
```

## Документация

- [Архитектура](./docs/ARCHITECTURE.md) - подробное описание архитектуры проекта
- [Логирование](./docs/LOGGING.md) - система логирования запросов
- [MVP структура](./docs/MVP_STRUCTURE.md) - идеи для развития проекта
- [История привычек и календарь](./docs/HABITS_HISTORY.md) - как работает версияция привычек и исторический календарь


