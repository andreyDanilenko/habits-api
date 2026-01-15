# Backend API

Go бекенд с архитектурой, подготовленной для масштабирования в микросервисы.

## Технологии

- **Go 1.21+** - язык программирования
- **Gin** - веб-фреймворк
- **godotenv** - загрузка переменных окружения

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

Сервер будет доступен по адресу `http://localhost:8080`

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


