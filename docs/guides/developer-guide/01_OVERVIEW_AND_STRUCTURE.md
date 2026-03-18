# Гайд разработчика: Обзор и структура Backend

**Для кого:** Разработчик, который учится и много кода написал с помощью AI.  
**Цель:** Понять, как строилась структура backend с самого начала.

---

## 1. Точка входа и жизненный цикл приложения

### 1.1. Цепочка запуска

```
cmd/api/main.go
    → config.Load()           # Загрузка конфигурации из env
    → app.New(cfg)            # Создание приложения
    → application.Run()       # Запуск HTTP-сервера
```

### 1.2. Что происходит в `app.New()`

| Шаг | Действие | Файл/Компонент |
|-----|----------|----------------|
| 1 | Инициализация PostgreSQL | `database.InitDB()` |
| 2 | Запуск миграций | `database.RunMigrations()` |
| 3 | Инициализация GORM (для Casbin) | `database.InitGormDB()` |
| 4 | Создание DI-контейнера | `di.NewContainer()` |
| 5 | Регистрация маршрутов | `container.RegisterRoutes()` |
| 6 | Загрузка политик Casbin | `PermissionService.EnsureSystemRolePolicies()` |
| 7 | Синхронизация назначений ролей | `PermissionService.SyncGroupingPoliciesFromAssignments()` |
| 8 | Запуск воркера логов | `worker.LogProcessor.Start()` |
| 9 | Запуск HTTP-сервера | `server.ListenAndServe()` |

---

## 2. Архитектурные слои (Layered Architecture)

Проект использует **многослойную архитектуру** с DI:

```
┌─────────────────────────────────────────────────────────────────┐
│  Handler Layer (internal/handler/*)                              │
│  HTTP-запросы, валидация, формирование ответов                   │
└────────────────────────────┬────────────────────────────────────┘
                             │ вызывает
┌────────────────────────────▼────────────────────────────────────┐
│  Service Layer (internal/service/*)                              │
│  Бизнес-логика, оркестрация, проверки                           │
└────────────────────────────┬────────────────────────────────────┘
                             │ вызывает
┌────────────────────────────▼────────────────────────────────────┐
│  Repository Layer (internal/repository/*)                        │
│  SQL-запросы, работа с БД                                       │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│  PostgreSQL + GORM (Casbin)                                      │
└─────────────────────────────────────────────────────────────────┘
```

**Правило:** Handler не знает о БД, Service не знает о HTTP, Repository не содержит бизнес-логики.

---

## 3. Структура директорий backend

```
backend/
├── cmd/
│   └── api/
│       └── main.go              # Точка входа
├── internal/                     # Приватный код (не импортируется извне)
│   ├── app/                      # Инициализация приложения
│   ├── authz/                    # Casbin, RBAC, endpoint registry
│   ├── config/                   # Конфигурация
│   ├── database/                 # Подключение к БД, миграции
│   ├── di/                       # Dependency Injection контейнер
│   ├── handler/                  # HTTP-обработчики по модулям
│   │   ├── auth/
│   │   ├── workspace/
│   │   ├── crm/
│   │   ├── habits/
│   │   ├── master/               # currencies, counterparties
│   │   └── ...
│   ├── middleware/              # Auth, Workspace, Module, Permission
│   ├── model/                    # Модели данных
│   ├── repository/              # Репозитории по модулям
│   ├── router/                   # Роутер Gin
│   ├── service/                  # Сервисы по модулям
│   └── worker/                  # Фоновые воркеры (логи)
├── migrations/                   # SQL-миграции
│   ├── 000001_*.sql ... 000032_*.sql   # Инкрементальные
│   ├── constraints/              # FK, триггеры
│   └── clean_baseline/           # Миграции для чистой БД
├── pkg/                          # Публичные утилиты
│   ├── auth/token/               # JWT
│   ├── email/                    # SMTP, Noop
│   ├── realtime/                 # Redis Pub/Sub
│   └── ...
└── docs/                         # Документация
```

---

## 4. Как создаются зависимости (DI Container)

В `internal/di/container.go` все зависимости создаются в правильном порядке:

```go
// 1. Репозитории (нижний слой)
workspaceRepository := workspaceRepo.NewRepository(db)
licenseRepository := licenseRepo.NewRepository(db)
userRepository := userRepo.NewRepository(db)
// ...

// 2. Сервисы (зависят от репозиториев)
permSvc := permissionService.NewService(permissionRepository, enforcer, workspaceRepository)
workspaceSvc := workspaceService.NewService(workspaceRepository, userPrefsRepository, licenseRepository, permSvc)

// 3. Хендлеры (зависят от сервисов)
workspaceHandler := workspaceHandler.NewHandler(workspaceSvc, ...)
```

**Важно:** `WorkspaceService` зависит от `PermissionService`, потому что при создании workspace нужно назначить OWNER и создать системные роли.

---

## 5. Поток HTTP-запроса

```
1. HTTP Request
   ↓
2. Router (Gin) → определяет маршрут
   ↓
3. Middleware (цепочка):
   • GinAuthMiddleware      → JWT, user_id в контекст
   • WorkspacePathMiddleware → проверка доступа к workspace
   • ModuleLicenseMiddleware → модуль включён, лицензия есть
   • PermissionMiddleware    → Casbin: право на действие
   ↓
4. Handler → парсинг, валидация
   ↓
5. Service → бизнес-логика
   ↓
6. Repository → SQL
   ↓
7. Response
```

---

## 6. Ключевые принципы, заложенные с начала

| Принцип | Реализация |
|---------|------------|
| **Мультитенантность** | Все данные изолированы по `workspace_id` |
| **Модульность** | Каждый домен (CRM, Habits, Notes) — отдельный handler/service/repository |
| **Core-модули** | habits, tasks, crm — включены по умолчанию при создании workspace |
| **Лицензирование** | Не-core модули требуют лицензию в `user_module_licenses` |
| **RBAC** | Casbin + permission_catalog + workspace_roles |
| **Приватность** | `internal/` — код не экспортируется наружу |

---

## 7. Связанные документы

- [ARCHITECTURE.md](../../guides/ARCHITECTURE.md) — подробнее об архитектуре
- [02_DATABASE_AND_TABLES.md](./02_DATABASE_AND_TABLES.md) — схема БД и связи таблиц
- [03_MODULES_INTERACTION.md](./03_MODULES_INTERACTION.md) — взаимодействие модулей
