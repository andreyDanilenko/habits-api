# Архитектура Backend

## Обзор архитектурного подхода

Проект использует **Layered Architecture (Многослойная архитектура)** с элементами **Clean Architecture** и **Dependency Injection (DI)** паттерна.

### Основные принципы:

1. **Разделение ответственности** - каждый слой отвечает за свою область
2. **Инверсия зависимостей** - зависимости идут от внешних слоев к внутренним
3. **Тестируемость** - каждый слой можно тестировать изолированно
4. **Масштабируемость** - легко добавлять новые модули и функциональность

## Структура слоев

```
┌─────────────────────────────────────┐
│         Handler Layer              │  ← HTTP обработка, валидация запросов
│    (internal/handler/*)            │
└──────────────┬──────────────────────┘
               │ использует
┌──────────────▼──────────────────────┐
│         Service Layer               │  ← Бизнес-логика, оркестрация
│    (internal/service/*)             │
└──────────────┬──────────────────────┘
               │ использует
┌──────────────▼──────────────────────┐
│       Repository Layer              │  ← Работа с БД, данные
│   (internal/repository/*)           │
└──────────────┬──────────────────────┘
               │ использует
┌──────────────▼──────────────────────┐
│         Database                    │  ← PostgreSQL
└─────────────────────────────────────┘
```

### Описание слоев:

#### 1. Handler Layer (`internal/handler/`)
- **Ответственность**: Обработка HTTP запросов и ответов
- **Функции**:
  - Парсинг и валидация входных данных
  - Вызов сервисов
  - Формирование HTTP ответов
  - Обработка ошибок
- **Не должен**: Содержать бизнес-логику, работать напрямую с БД

**Пример структуры:**
```go
type Handler struct {
    service *service.Service
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
    r.GET("/:id", h.Get)
    r.POST("", h.Create)
}
```

#### 2. Service Layer (`internal/service/`)
- **Ответственность**: Бизнес-логика приложения
- **Функции**:
  - Оркестрация операций
  - Валидация бизнес-правил
  - Координация между репозиториями
  - Трансформация данных
- **Не должен**: Знать о HTTP, работать напрямую с БД

**Пример структуры:**
```go
type Service struct {
    repo *repository.Repository
}

func (s *Service) CreateUser(ctx context.Context, user *model.User) error {
    // Бизнес-логика создания пользователя
    if err := s.validateUser(user); err != nil {
        return err
    }
    return s.repo.Create(ctx, user)
}
```

#### 3. Repository Layer (`internal/repository/`)
- **Ответственность**: Работа с данными
- **Функции**:
  - SQL запросы к БД
  - Маппинг данных из БД в модели
  - Абстракция от конкретной БД
- **Не должен**: Содержать бизнес-логику

**Пример структуры:**
```go
type Repository struct {
    db *sql.DB
}

func (r *Repository) Create(ctx context.Context, user *model.User) error {
    query := `INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id`
    return r.db.QueryRowContext(ctx, query, user.Email, user.Password).Scan(&user.ID)
}
```

## Dependency Injection (DI) Container

### Назначение

DI контейнер (`internal/di/container.go`) централизует создание и управление зависимостями приложения.

### Преимущества:

1. **Единая точка инициализации** - все зависимости создаются в одном месте
2. **Управление жизненным циклом** - контролируем создание и порядок инициализации
3. **Тестируемость** - легко подменять зависимости моками
4. **Читаемость** - видно все зависимости приложения

### Как это работает:

```go
func NewContainer(db *sql.DB) *Container {
    // 1. Создаем репозитории (нижний слой)
    authRepo := authRepo.NewRepository(db)
    
    // 2. Создаем сервисы (средний слой) с зависимостями от репозиториев
    authSvc := authService.NewService(authRepo)
    
    // 3. Создаем handlers (верхний слой) с зависимостями от сервисов
    authHdlr := authHandler.NewHandler(authSvc)
    
    return &Container{
        AuthHandler: authHdlr,
        // ...
    }
}
```

### Регистрация маршрутов:

```go
func (c *Container) RegisterRoutes(r *router.Router) {
    // Группируем маршруты по модулям
    authGroup := r.Group("/auth")
    c.AuthHandler.RegisterRoutes(authGroup)
    
    workspaceGroup := r.Group("/workspaces")
    c.WorkspaceHandler.RegisterRoutes(workspaceGroup)
}
```

## Приватность пакетов в Go

### Директория `internal/`

В Go все пакеты внутри директории `internal/` являются **приватными** для родительского модуля.

#### Правила:

1. **Пакеты в `internal/`** могут импортироваться только:
   - Пакетами того же модуля, которые находятся на том же уровне или выше `internal/`
   - Пакетами в `cmd/`, `main.go` в корне

2. **Внешние модули** не могут импортировать пакеты из `internal/`

#### Пример структуры:

```
backend/
├── main.go                    ← может импортировать internal/*
├── cmd/
│   └── api/
│       └── main.go           ← может импортировать internal/*
└── internal/                  ← приватная директория
    ├── handler/              ← приватный пакет
    ├── service/              ← приватный пакет
    └── repository/           ← приватный пакет
```

#### Зачем это нужно:

- **Инкапсуляция** - скрываем внутреннюю реализацию от внешних потребителей
- **Контроль API** - явно определяем, что можно использовать извне
- **Рефакторинг** - можем менять внутреннюю структуру без влияния на внешние модули

#### Пример использования:

```go
// ✅ Правильно - из main.go
package main
import "backend/internal/app"  // OK

// ❌ Неправильно - из внешнего модуля
package external
import "backend/internal/app"  // Ошибка компиляции!
```

### Публичные пакеты

Если нужно предоставить API для внешнего использования, создавайте пакеты в:
- `pkg/` - публичные библиотеки и утилиты
- Корневой уровень модуля (но не рекомендуется для внутренних пакетов)

## Ролевая модель и авторизация маршрутов

### Подход к реализации

Для реализации ролевой модели и защиты маршрутов используется **Middleware паттерн**.

### Структура middleware:

```
Request → Auth Middleware → Role Middleware → Handler
```

### 1. Создание middleware для аутентификации

Создайте файл `internal/middleware/auth.go`:

```go
package middleware

import (
    "backend/internal/service/auth"
    "github.com/gin-gonic/gin"
    "net/http"
    "strings"
)

func AuthMiddleware(authService *auth.Service) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Извлекаем токен из заголовка
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
            c.Abort()
            return
        }

        // Проверяем формат "Bearer <token>"
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
            c.Abort()
            return
        }

        token := parts[1]

        // Валидируем токен и получаем пользователя
        user, err := authService.ValidateToken(c.Request.Context(), token)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }

        // Сохраняем пользователя в контексте
        c.Set("user", user)
        c.Set("userID", user.ID)
        
        c.Next()
    }
}
```

### 2. Создание middleware для ролей

Создайте файл `internal/middleware/role.go`:

```go
package middleware

import (
    "github.com/gin-gonic/gin"
    "net/http"
)

// RoleMiddleware проверяет, что у пользователя есть необходимая роль
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        user, exists := c.Get("user")
        if !exists {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
            c.Abort()
            return
        }

        // Предполагаем, что у user есть метод GetRole()
        // userObj := user.(*model.User)
        // userRole := userObj.Role

        // Проверяем роль (примерная реализация)
        // if !contains(allowedRoles, userRole) {
        //     c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
        //     c.Abort()
        //     return
        // }

        c.Next()
    }
}

// AdminOnly - middleware только для администраторов
func AdminOnly() gin.HandlerFunc {
    return RoleMiddleware("admin")
}

// WorkspaceOwnerOrAdmin - middleware для владельца workspace или админа
func WorkspaceOwnerOrAdmin() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Логика проверки владельца workspace
        // workspaceID := c.Param("id")
        // userID := c.GetString("userID")
        
        // if !isWorkspaceOwner(workspaceID, userID) && !isAdmin(userID) {
        //     c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
        //     c.Abort()
        //     return
        // }
        
        c.Next()
    }
}
```

### 3. Применение middleware в handlers

#### Вариант 1: На уровне группы маршрутов

```go
// В container.go
func (c *Container) RegisterRoutes(r *router.Router) {
    // Публичные маршруты (без аутентификации)
    authGroup := r.Group("/auth")
    c.AuthHandler.RegisterRoutes(authGroup)

    // Защищенные маршруты (требуют аутентификации)
    protected := r.Group("")
    protected.Use(middleware.AuthMiddleware(c.authService))
    {
        workspaceGroup := protected.Group("/workspaces")
        c.WorkspaceHandler.RegisterRoutes(workspaceGroup)
        
        habitsGroup := protected.Group("/habits")
        c.HabitsHandler.RegisterRoutes(habitsGroup)
    }

    // Административные маршруты
    admin := r.Group("/admin")
    admin.Use(middleware.AuthMiddleware(c.authService))
    admin.Use(middleware.AdminOnly())
    {
        // Админские endpoints
    }
}
```

#### Вариант 2: На уровне отдельных маршрутов

```go
// В handler.go
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
    // Публичный маршрут
    r.POST("/register", h.Register)
    r.POST("/login", h.Login)

    // Защищенные маршруты
    r.GET("/me", middleware.AuthMiddleware(h.authService), h.Me)
    r.POST("/logout", middleware.AuthMiddleware(h.authService), h.Logout)
}
```

#### Вариант 3: Комбинированный подход

```go
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
    // Публичные
    r.POST("/login", h.Login)
    r.POST("/register", h.Register)

    // Группа защищенных маршрутов
    protected := r.Group("")
    protected.Use(middleware.AuthMiddleware(h.authService))
    {
        protected.GET("/me", h.Me)
        protected.POST("/refresh", h.Refresh)
        protected.POST("/logout", h.Logout)
    }
}
```

### 4. Получение пользователя в handler

```go
func (h *Handler) GetProfile(c *gin.Context) {
    // Получаем пользователя из контекста
    user, exists := c.Get("user")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
        return
    }

    userObj := user.(*model.User)
    
    // Используем userObj для дальнейшей логики
    profile, err := h.service.GetUserProfile(c.Request.Context(), userObj.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, profile)
}
```

### Рекомендуемая структура ролей:

```go
// internal/model/role.go
const (
    RoleAdmin    = "admin"
    RoleUser     = "user"
    RoleOwner    = "owner"
    RoleMember   = "member"
    RoleViewer   = "viewer"
)
```

### Примеры использования:

```go
// Только для авторизованных пользователей
protected.Use(middleware.AuthMiddleware(authService))

// Только для администраторов
admin.Use(middleware.AuthMiddleware(authService), middleware.AdminOnly())

// Для владельца workspace или админа
workspaceAdmin.Use(middleware.AuthMiddleware(authService), middleware.WorkspaceOwnerOrAdmin())

// Для пользователей с определенными ролями
membersOnly.Use(middleware.AuthMiddleware(authService), middleware.RoleMiddleware("member", "owner"))
```

## Запуск приложения

### Предварительные требования

1. **Go 1.21+** установлен
2. **PostgreSQL** запущен и доступен
3. **Переменные окружения** настроены

### Настройка переменных окружения

Создайте файл `.env` в корне проекта `backend/`:

```env
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=app_db
```

### Установка зависимостей

```bash
cd backend
go mod download
go mod tidy
```

### Запуск приложения

#### Вариант 1: Через cmd/api/main.go (текущий способ)

```bash
cd backend
go run cmd/api/main.go
```

#### Вариант 2: Сборка и запуск

```bash
cd backend
go build -o bin/backend ./cmd/api
./bin/backend
```

#### Вариант 3: С использованием air (hot reload)

```bash
# Установка air
go install github.com/cosmtrek/air@latest

# Запуск с hot reload
air
```

### Проверка работы

После запуска сервер будет доступен по адресу: `http://localhost:8080`

Проверьте health check:
```bash
curl http://localhost:8080/health
```

Ожидаемый ответ:
```json
{
  "status": "ok",
  "service": "backend"
}
```

## Структура точки входа

### Текущая структура:

```
backend/
└── cmd/
    └── api/
        └── main.go      ← Точка входа для API сервиса
```

### Почему cmd/api/main.go?

Это стандартная структура Go-проектов согласно [Go Project Layout](https://github.com/golang-standards/project-layout):

- **`cmd/`** - содержит точки входа приложений
- **`cmd/api/`** - точка входа для API сервиса
- В будущем можно добавить `cmd/worker/`, `cmd/scheduler/` и т.д.

### Эволюция архитектуры:

#### Этап 1: Монолит (текущее состояние)
```
cmd/api/main.go → запускает все сервисы вместе
```

#### Этап 2: Модульный монолит
```
cmd/
├── api/          ← API сервис
├── worker/       ← Фоновые задачи
└── scheduler/    ← Планировщик
```

#### Этап 3: Микросервисы
```
cmd/
├── api/          ← REST API микросервис
├── auth/         ← Аутентификация микросервис
├── habits/       ← Habits микросервис
└── journal/      ← Journal микросервис
```

### Преимущества такого подхода:

1. **Готовность к масштабированию** - легко выделить отдельный сервис
2. **Единая кодовая база** - общий `internal/` код
3. **Постепенная миграция** - можно мигрировать по частям
4. **Тестирование** - каждый сервис можно тестировать отдельно

### Когда использовать cmd/api/main.go:

Когда придет время разделять на микросервисы, структура будет такой:

```go
// cmd/api/main.go
package main

import (
    "backend/internal/app"
    "backend/internal/config"
    "log"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Создаем только API часть приложения
    application, err := app.NewAPI(cfg)
    if err != nil {
        log.Fatalf("Failed to create API application: %v", err)
    }
    
    if err := application.Run(); err != nil {
        log.Fatalf("Failed to run API: %v", err)
    }
}
```

И в `internal/app/app.go` можно будет создать отдельные конструкторы:

```go
// Для монолита
func New(cfg *config.Config) (*App, error) { ... }

// Для API микросервиса
func NewAPI(cfg *config.Config) (*App, error) { ... }

// Для Worker микросервиса
func NewWorker(cfg *config.Config) (*App, error) { ... }
```

## Поток выполнения запроса

```
1. HTTP Request
   ↓
2. Router (Gin) → определяет маршрут
   ↓
3. Middleware (если есть) → аутентификация, авторизация
   ↓
4. Handler → парсинг запроса, валидация
   ↓
5. Service → бизнес-логика
   ↓
6. Repository → SQL запросы к БД
   ↓
7. Database → выполнение запроса
   ↓
8. Repository → маппинг результата
   ↓
9. Service → обработка данных
   ↓
10. Handler → формирование ответа
   ↓
11. HTTP Response
```

## Рекомендации по разработке

1. **Всегда начинайте с Repository** - сначала реализуйте работу с данными
2. **Затем Service** - добавьте бизнес-логику
3. **В конце Handler** - свяжите с HTTP
4. **Тестируйте каждый слой** - пишите unit тесты для каждого слоя
5. **Используйте context** - передавайте `context.Context` через все слои
6. **Обрабатывайте ошибки** - возвращайте понятные ошибки на каждом уровне

## Дополнительные ресурсы

- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [Clean Architecture in Go](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Gin Framework Documentation](https://gin-gonic.com/docs/)
