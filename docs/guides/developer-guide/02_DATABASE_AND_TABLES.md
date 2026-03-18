# Гайд разработчика: База данных и связи между таблицами

**Цель:** Понять, как устроена схема БД и как таблицы связаны друг с другом.

---

## 1. Система миграций

### 1.1. Два типа миграций

| Тип | Папка | Назначение |
|-----|-------|------------|
| **Инкрементальные** | `migrations/000001_*.sql` … `000032_*.sql` | Разработка: добавляют изменения поверх существующей БД |
| **Clean baseline** | `migrations/clean_baseline/001_*.sql` … `027_*.sql` | Продакшен: создание БД с нуля по сущностям |

**Важно:** При изменении инкрементальных миграций нужно обновлять соответствующий файл в `clean_baseline/`. См. `migrations/README.md`.

### 1.2. Порядок создания сущностей (clean_baseline)

```
001 → request_logs (инфра)
002 → users
003 → workspaces
004 → user_workspaces
005 → user_preferences
006 → modules, workspace_modules, user_module_licenses
007 → habits
008 → habit_completions
...
022 → permissions (permission_catalog, workspace_roles, user_role_assignments, ...)
023 → registration_tokens
024 → invitations
025 → notifications
026 → tasks
027 → task_comments
```

---

## 2. Визуальная схема связей

```
                    ┌─────────────┐
                    │   users     │
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
         ▼                 ▼                 ▼
┌────────────────┐ ┌──────────────┐ ┌─────────────────────┐
│user_workspaces │ │user_preferences│ │user_module_licenses│
└───────┬────────┘ └──────────────┘ └──────────┬─────────┘
        │                                        │
        │         ┌─────────────┐                 │
        └────────►│ workspaces │◄─────────────────┘
                  └──────┬─────┘
                         │
    ┌────────────────────┼────────────────────┐
    │                    │                    │
    ▼                    ▼                    ▼
┌─────────────┐  ┌──────────────┐  ┌─────────────────┐
│workspace_   │  │ invitations  │  │ permission_     │
│modules      │  │              │  │ catalog, roles   │
└──────┬──────┘  └──────────────┘  └─────────────────┘
       │
       ▼
┌─────────────┐
│   modules   │  (справочник: habits, crm, projects, tasks, ...)
└─────────────┘
```

---

## 3. Группы таблиц и их связи

### 3.1. Ядро (Core)

| Таблица | Назначение | FK / связи |
|---------|------------|------------|
| `users` | Пользователи | — |
| `workspaces` | Рабочие пространства | `owner_id` → users |
| `user_workspaces` | Связь user ↔ workspace (роль) | `user_id` → users, `workspace_id` → workspaces |
| `user_preferences` | Текущий выбранный workspace | `user_id` → users, `current_workspace_id` → workspaces |

**Связь:** Один пользователь может быть в нескольких workspace (M:N через `user_workspaces`).

### 3.2. Модули и лицензии

| Таблица | Назначение | FK / связи |
|---------|------------|------------|
| `modules` | Справочник модулей (code, name, is_core) | — |
| `workspace_modules` | Какие модули включены в workspace | `workspace_id` → workspaces, `module_id` → modules |
| `user_module_licenses` | Лицензия пользователя на модуль | `user_id` → users, `module_id` → modules, `workspace_id` (опц.) |

**Важно:**
- `workspace_modules` = «модуль включён в этом workspace»
- `user_module_licenses` = «у пользователя есть право включить модуль»

### 3.3. Права доступа (RBAC)

| Таблица | Назначение | FK / связи |
|---------|------------|------------|
| `permission_catalog` | Каталог прав (module:entity:action) | — |
| `workspace_roles` | Роли workspace (OWNER, ADMIN, MEMBER, GUEST + кастомные) | `workspace_id` → workspaces |
| `user_role_assignments` | Назначение ролей пользователям | user_id, role_id, workspace_id |
| `user_permissions` | Индивидуальные права (минуя роль) | user_id, permission_id, workspace_id |
| `role_inheritance` | Наследование ролей (child ← parent) | child_role_id, parent_role_id |

**Casbin:** Политики хранятся в `casbin_rule` (через GORM-адаптер). Синхронизируются с БД при создании/изменении ролей.

### 3.4. Shared (общие справочники)

| Таблица | Назначение | FK / связи |
|---------|------------|------------|
| `currencies` | Валюты | `workspace_id` → workspaces |
| `counterparties` | Контрагенты (client/supplier/both) | `workspace_id` → workspaces |

**Scope:** Все данные shared привязаны к workspace.

### 3.5. Доменные модули (все с workspace_id)

| Модуль | Таблицы | Связи |
|--------|---------|-------|
| **Habits** | habits, habit_completions, habit_history, habit_versions | habits → users, workspaces; completions → habits, users |
| **Notes** | notes | workspace_id, user_id |
| **Journal** | journal_entries | workspace_id, user_id |
| **CRM** | crm_contacts, crm_companies, crm_pipelines, crm_stages, crm_deals, crm_activities, ... | Все через workspace_id (без FK на users/workspaces) |
| **Projects** | projects, project_entities | projects → workspaces; project_entities → projects + entity |
| **Tasks** | tasks, task_comments | workspace_id, project_id (опц.) |

### 3.6. Аутентификация и приглашения

| Таблица | Назначение | FK / связи |
|---------|------------|------------|
| `registration_tokens` | Токены подтверждения email, invite_token | user_id (опц.) |
| `invitations` | Приглашения в workspace | workspace_id, invited_by |

---

## 4. Ключевые FK (constraints)

Файл `migrations/constraints/01_foreign_keys.up.sql`:

```sql
-- workspaces.owner_id → users
-- user_workspaces.user_id → users
-- user_workspaces.workspace_id → workspaces
-- habits.user_id → users
-- habits.workspace_id → workspaces
-- habit_completions.habit_id → habits
-- habit_completions.user_id → users
```

**Примечание:** CRM-таблицы **не имеют FK** на users/workspaces — только `workspace_id` как UUID. Это задел под выделение CRM в отдельный микросервис.

---

## 5. Триггеры

| Триггер | Таблица | Действие |
|---------|---------|----------|
| `fn_workspace_enable_modules` | workspaces | AFTER INSERT → включение core-модулей и trial для не-core |
| `fn_create_system_roles` | workspaces | AFTER INSERT → создание OWNER, ADMIN, MEMBER, GUEST |
| `update_workspaces_updated_at` | workspaces | Обновление updated_at |

---

## 6. Как читать связи при разработке

1. **Новая сущность в модуле** → всегда `workspace_id` (NOT NULL), при необходимости `user_id`/`owner_id`.
2. **Связь с пользователем** → хранить `user_id` (UUID), FK опционален для микросервисов.
3. **Связь между модулями** → через `project_entities` (entity_type, entity_id) или отдельные link-таблицы.
4. **Проверка доступа** → через `WorkspaceService.HasAccess(userID, workspaceID)` и `PermissionService`.

---

## 7. Связанные документы

- [SCHEMA.md](../../schema/SCHEMA.md) — визуализация схемы
- [SQL_AND_MIGRATIONS_GUIDE.md](../SQL_AND_MIGRATIONS_GUIDE.md) — правила миграций
- [03_MODULES_INTERACTION.md](./03_MODULES_INTERACTION.md) — как модули взаимодействуют
