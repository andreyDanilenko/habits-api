# Декомпозиция под микросервисы

## 1. Разбиение по доменам

Текущая монолитная схема разбивается на **сервисы** по bounded context:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         API Gateway / BFF                                    │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        │                           │                           │
        ▼                           ▼                           ▼
┌───────────────┐         ┌───────────────┐         ┌───────────────┐
│ Auth Service  │         │Workspace Svc  │         │Permission Svc │
│ (Identity)    │         │(Tenancy)      │         │(RBAC)         │
└───────────────┘         └───────────────┘         └───────────────┘
        │                           │                           │
        └───────────────────────────┼───────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        │           │               │               │            │
        ▼           ▼               ▼               ▼            ▼
┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐
│  Habits   │ │   CRM     │ │  Notes    │ │ Projects  │ │  Master   │
│  Service  │ │  Service  │ │  Service  │ │  Service  │ │  Service  │
└───────────┘ └───────────┘ └───────────┘ └───────────┘ └───────────┘
        │           │               │               │            │
        └───────────┴───────────────┴───────────────┴────────────┘
                                    │
                                    ▼
                          ┌───────────────┐
                          │ Logger Service│
                          │ (Observability)│
                          └───────────────┘
```

---

## 2. Таблицы по сервисам

| Сервис | Таблицы | Зависимости |
|--------|---------|-------------|
| **auth** | `users`, `registration_tokens` | — |
| **workspace** | `workspaces`, `user_workspaces`, `user_preferences`, `invitations`, `modules`, `workspace_modules`, `user_module_licenses` | users (ID) |
| **permission** | `permission_catalog`, `workspace_roles`, `user_role_assignments`, `user_permissions`, `role_inheritance` | users, workspaces |
| **habits** | `habits`, `habit_completions`, `habit_history`, `habit_versions`, `journal_entries` | users, workspaces |
| **notes** | `notes` | users, workspaces |
| **crm** | `crm_contacts`, `crm_contact_phones`, `crm_contact_emails`, `crm_companies`, `crm_company_contacts`, `crm_pipelines`, `crm_stages`, `crm_deals`, `crm_activities`, `crm_activity_files`, `crm_activity_reminders` | users, workspaces |
| **projects** | `projects`, `project_entities` | workspaces |
| **master** | `currencies`, `counterparties` | workspaces |
| **activities** | `activities` | users, workspaces |
| **logger** | `request_logs` | — |

---

## 3. Варианты разбиения БД

### Вариант A: Schema-per-Service (одна БД, разные схемы)

Одна PostgreSQL, каждый сервис — своя схема. Проще миграции, общие транзакции невозможны.

```
postgres://host/db
├── auth
│   ├── users
│   └── registration_tokens
├── workspace
│   ├── workspaces
│   ├── user_workspaces
│   └── ...
├── habits
│   ├── habits
│   └── habit_completions
├── crm
│   ├── crm_contacts
│   └── ...
└── ...
```

**Плюсы:** один инстанс, проще бэкапы, FK между схемами через `schema.table`.  
**Минусы:** не изолированы по масштабированию, один отказ — все сервисы.

---

### Вариант B: Database-per-Service (отдельная БД на сервис)

Каждый сервис — своя БД. Связи только по ID (UUID), без FK между БД.

```
auth_db          → users, registration_tokens
workspace_db     → workspaces, user_workspaces, modules, ...
permission_db    → permission_catalog, workspace_roles, ...
habits_db        → habits, habit_completions, journal_entries
crm_db           → crm_*
notes_db         → notes
projects_db      → projects, project_entities
master_db        → currencies, counterparties
activities_db    → activities
logger_db        → request_logs
```

**Плюсы:** изоляция, независимое масштабирование, можно разные СУБД.  
**Минусы:** eventual consistency, дублирование user_id/workspace_id, нужны API/события для проверки.

---

### Вариант C: Гибрид (Core + Domain DBs)

**Core DB** (auth + workspace + permission) — общая для всех.  
**Domain DBs** — habits, crm, notes, projects, master, logger.

```
core_db (auth + workspace + permission)
├── users, registration_tokens
├── workspaces, user_workspaces, modules, ...
└── permission_catalog, workspace_roles, ...

habits_db
crm_db
notes_db
projects_db
master_db
logger_db
```

**Плюсы:** меньше дублирования, Core — единый источник правды по users/workspaces.  
**Минусы:** Core — единая точка отказа, но её проще реплицировать.

---

## 4. Структура миграций по сервисам

```
backend/
├── migrations/
│   ├── auth/
│   │   └── 001_init.sql
│   ├── workspace/
│   │   └── 001_init.sql
│   ├── permission/
│   │   └── 001_init.sql
│   ├── habits/
│   │   └── 001_init.sql
│   ├── crm/
│   │   └── 001_init.sql
│   ├── notes/
│   │   └── 001_init.sql
│   ├── projects/
│   │   └── 001_init.sql
│   ├── master/
│   │   └── 001_init.sql
│   ├── activities/
│   │   └── 001_init.sql
│   └── logger/
│       └── 001_init.sql
```

Каждый `001_init.sql` — только таблицы своего домена. Для варианта B — без FK на другие БД.

---

## 5. DI Container по сервисам

### Монолит (как сейчас)

```go
// internal/di/container.go — всё в одном
func NewContainer(db *sql.DB, gormDB *gorm.DB, cfg *config.Config) (*Container, error) {
    userRepo := userRepo.NewRepository(db)
    workspaceRepo := workspaceRepo.NewRepository(db)
    crmRepo := crmRepo.NewRepository(db)
    habitsRepo := habitsRepo.NewRepository(db)
    // ... 15+ репозиториев, 10+ сервисов
}
```

### Микросервисы (пример: Habits Service)

```go
// services/habits/internal/di/container.go
package di

import (
    "database/sql"
    habitsRepo "services/habits/internal/repository/habits"
    habitsSvc "services/habits/internal/service/habits"
    habitsHandler "services/habits/internal/handler/habits"
)

type Container struct {
    DB             *sql.DB
    HabitsRepo     *habitsRepo.Repository
    HabitsService  *habitsSvc.Service
    HabitsHandler  *habitsHandler.Handler
    WorkspaceClient workspace.Client  // gRPC/HTTP — проверка workspace_id
}

func NewContainer(db *sql.DB, workspaceClient workspace.Client, cfg *config.Config) (*Container, error) {
    habitsRepo := habitsRepo.NewRepository(db)
    habitsSvc := habitsSvc.NewService(habitsRepo, workspaceClient)
    habitsHdlr := habitsHandler.NewHandler(habitsSvc)
    
    return &Container{
        DB:            db,
        HabitsRepo:    habitsRepo,
        HabitsService: habitsSvc,
        HabitsHandler: habitsHdlr,
    }, nil
}
```

### CRM Service (с зависимостью от Workspace)

```go
// services/crm/internal/di/container.go
type Container struct {
    DB              *sql.DB
    CrmRepo         *crmRepo.Repository
    CrmService      *crmService.Service
    CrmHandler      *crmHandler.Handler
    WorkspaceClient workspace.Client  // проверка доступа к workspace
    UserClient      user.Client      // опционально: имена владельцев
}

func NewContainer(db *sql.DB, wsClient workspace.Client, userClient user.Client, cfg *config.Config) (*Container, error) {
    crmRepo := crmRepo.NewRepository(db)
    crmSvc := crmService.NewService(crmRepo, wsClient, userClient)
    crmHdlr := crmHandler.NewHandler(crmSvc)
    // ...
}
```

---

## 6. Конфигурация БД по сервисам

### Вариант A (Schema-per-Service)

```yaml
# config.yaml
database:
  host: localhost
  port: 5432
  dbname: erp
  user: erp_app
  password: secret
  # Каждый сервис подключается к той же БД, но со своим search_path
  schema: habits  # или auth, workspace, crm, ...
```

### Вариант B (Database-per-Service)

```yaml
# services/habits/config.yaml
database:
  host: localhost
  port: 5432
  dbname: erp_habits   # отдельная БД
  user: habits_app
  password: secret

# services/crm/config.yaml
database:
  dbname: erp_crm

# services/auth/config.yaml
database:
  dbname: erp_auth
```

### Вариант C (Гибрид)

```yaml
# services/auth, workspace, permission
database:
  dbname: erp_core

# services/habits, crm, notes, ...
database:
  dbname: erp_habits  # или erp_crm, erp_notes
```

---

## 7. Связи между сервисами

| От кого | К кому | Что нужно | Как |
|---------|--------|-----------|-----|
| Habits | Workspace | Проверка workspace_id | gRPC/HTTP `ValidateWorkspaceAccess(user_id, workspace_id)` |
| CRM | Workspace | То же | То же |
| CRM | Auth | Имена owner_id | gRPC `GetUsers(ids[])` или кэш |
| Notes | Workspace | То же | То же |
| Projects | Workspace | То же | То же |
| Permission | Auth, Workspace | user_id, workspace_id | Локально в Core DB |
| Все | Auth | Валидация JWT | Общий JWKS или Auth API |

---

## 8. Рекомендация

Для вашего кейса разумно начать с **Варианта C (гибрид)**:

1. **Core DB** — auth, workspace, permission (часто нужны вместе).
2. **Domain DBs** — habits, crm, notes, projects, master, logger.

Плюсы:
- Меньше дублирования users/workspaces.
- Permission и workspace в одной БД — проще транзакции.
- Доменные сервисы изолированы и могут масштабироваться отдельно.
- Миграция из монолита пошаговая: сначала схемы, потом разделение БД.

Следующий шаг — вынести SQL-файлы по папкам `migrations/auth/`, `migrations/workspace/`, `migrations/habits/` и т.д. из текущего `NEW_MIGRATE.sql`.

---

## 9. Пример: Habits Service (Database-per-Service)

При **отдельной БД** у habits нет FK на `users` и `workspaces` — только хранение UUID. Проверка доступа через Workspace API.

```sql
-- migrations/habits/001_init.sql (Database-per-Service)
-- БЕЗ REFERENCES users(id), workspaces(id) — они в другой БД

CREATE TABLE habits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,           -- без FK, проверка через Auth/Workspace API
    workspace_id UUID NOT NULL,     -- без FK
    -- ... остальные поля
);

CREATE TABLE habit_completions (
    habit_id UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    workspace_id UUID,
    -- ...
);

-- habit_versions, journal_entries — аналогично, user_id/workspace_id без FK
```

При **Schema-per-Service** (одна БД) можно оставить FK, если `users` и `workspaces` в схеме `auth`/`workspace`:

```sql
-- migrations/habits/001_init.sql (Schema-per-Service, одна БД)
CREATE TABLE habits (
    user_id UUID NOT NULL REFERENCES auth.users(id),
    workspace_id UUID NOT NULL REFERENCES workspace.workspaces(id),
    -- ...
);
```
