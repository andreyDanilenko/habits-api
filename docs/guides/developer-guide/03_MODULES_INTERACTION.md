# Гайд разработчика: Взаимодействие модулей

**Цель:** Понять, как связаны пользователи, workspace, лицензии, контрагенты, валюты и другие модули.

---

## 1. Центральная сущность: Workspace

**Workspace** — это «организация» или «рабочее пространство». Все данные в системе изолированы по `workspace_id`.

```
User A ──┬── Workspace 1 (компания X) ── CRM, Habits, Currencies, Counterparties
         │
         └── Workspace 2 (компания Y) ── CRM, Habits, Currencies, Counterparties
```

Один пользователь может состоять в нескольких workspace. В каждом workspace — свои данные, свои роли, свои модули.

---

## 2. Пользователи и Workspace

### 2.1. Глобальная роль (`users.role`)

| Роль | Описание |
|------|----------|
| `ADMIN` | Суперадмин приложения. Доступ ко всем workspace. |
| `USER` | Обычный пользователь. Доступ только к workspace, где он участник. |

### 2.2. Системная роль в workspace (`user_workspaces.role`)

| Роль | Описание |
|------|----------|
| `OWNER` | Владелец workspace. Полный доступ, управление модулями и лицензиями. |
| `ADMIN` | Администратор. Управление участниками, ролями, приглашениями. |
| `MEMBER` | Участник. Работа с данными по своим правам. |
| `GUEST` | Гость. Read-only доступ. |

### 2.3. Цепочка проверки доступа

```
Запрос: GET /api/v1/workspaces/{workspaceId}/crm/deals

1. GinAuthMiddleware     → user_id из JWT
2. WorkspacePathMiddleware → WorkspaceService.HasAccess(user_id, workspace_id)
   • Если user.role == ADMIN → доступ ко всем workspace
   • Иначе: проверка user_workspaces (есть ли запись)
3. ModuleLicenseMiddleware → модуль CRM включён в workspace? Есть лицензия?
4. PermissionMiddleware   → Casbin: есть ли право crm:deal:read?
```

---

## 3. Модули и лицензии

### 3.1. Таблицы

| Таблица | Назначение |
|---------|------------|
| `modules` | Справочник: habits, crm, projects, tasks, notes, journal. Поле `is_core` — бесплатный модуль. |
| `workspace_modules` | Модуль включён в workspace (status: active, trial, disabled) |
| `user_module_licenses` | У пользователя есть право включить модуль (scope: all_workspaces / single_workspace) |

### 3.2. Core vs не-core модули

| Тип | Примеры | Лицензия |
|-----|---------|----------|
| **Core** | habits, tasks | Бесплатно, включаются автоматически при создании workspace |
| **Не-core** | crm, projects (если не core) | Требуется запись в `user_module_licenses` |

### 3.3. Цепочка включения модуля

```
1. Владелец workspace нажимает «Включить CRM»
2. POST /workspaces/:id/modules { "moduleCode": "crm" }
3. WorkspaceService.EnableModule():
   • Модуль core? → включаем без проверки
   • Иначе: LicenseRepository.HasLicense(user_id, module_id, workspace_id)?
   • INSERT/UPDATE workspace_modules SET status = 'active'
```

### 3.4. Scope лицензии

| Scope | Описание |
|-------|----------|
| `all_workspaces` | Модуль можно включить в любом workspace пользователя |
| `single_workspace` | Модуль только в указанном workspace |

---

## 4. Currencies (Валюты) и Counterparties (Контрагенты)

### 4.1. Общие черты

- Оба — **shared-справочники** в рамках workspace
- API: `/api/v1/workspaces/:workspaceId/currencies`, `/api/v1/workspaces/:workspaceId/counterparties`
- Один handler: `internal/handler/master/`
- Один service: `internal/service/master/`
- Один repository: `internal/repository/master/`

### 4.2. Currencies

- Хранятся в `currencies` (workspace_id, code, name, symbol, ...)
- Используются в сделках, документах, отчётах

### 4.3. Counterparties

- Хранятся в `counterparties` (workspace_id, name, type: client/supplier/both, ...)
- Связь с CRM: контакт/компания может ссылаться на контрагента
- Используются в документах, сделках

### 4.4. Связь с другими модулями

```
currencies ──► crm_deals (валюта сделки)
counterparties ──► crm_contacts, crm_companies (опционально)
```

---

## 5. Приглашения (Invitations)

### 5.1. Поток

```
ADMIN создаёт приглашение (email, роль)
    → INSERT invitations (token, email, workspace_id, system_role)
    → Email с ссылкой /invite/{token}

Пользователь переходит по ссылке
    → GET /public/invitations/{token}
    → userExists? isAuthenticated? email совпадает?

Существующий пользователь:
    → POST /accept → user_workspaces + user_role_assignments
    → inv.status = ACCEPTED

Новый пользователь:
    → Редирект на /register?email=...&inviteToken=...
    → После регистрации: user + user_workspaces + inv.status = ACCEPTED
```

### 5.2. Защита

- Токен привязан к email
- При принятии: `currentUser.Email == invitation.email`
- Email при регистрации по invite — readonly

---

## 6. Права доступа (Permissions)

### 6.1. Формат права

`{module}:{entity}:{action}` — например: `crm:deal:create`, `habits:habit:complete`

### 6.2. Источники прав пользователя

1. **Системные роли** (OWNER, ADMIN, MEMBER, GUEST) — предопределённые наборы в коде
2. **Кастомные роли** — создаются ADMIN, назначаются пользователям
3. **Индивидуальные права** — выдаются в обход ролей (user_permissions)

### 6.3. Casbin

- Политики: `p("role:Admin", workspaceId, "crm:deal", "create")`
- Группировки: `g("user:uuid", "role:Admin", workspaceId)`
- Проверка: `Enforcer.Enforce("user:uuid", workspaceId, "crm:deal", "create")`

---

## 7. Projects — связка сущностей

`projects` и `project_entities` позволяют группировать сущности из разных модулей:

```
Project "Клиент X"
├── project_entities: entity_type=crm_deal, entity_id=uuid1
├── project_entities: entity_type=task, entity_id=uuid2
└── project_entities: entity_type=note, entity_id=uuid3
```

Модули (CRM, Tasks, Notes) **не хранят** project_id — связь только через `project_entities`.

---

## 8. Realtime (Redis)

- События публикуются в каналы: `ws:workspace:{id}`, `ws:user:{id}`
- Используется: CRM (сделки), Habits, Journal, Invitations
- Если Redis недоступен — используется NoopPublisher (события не доставляются)

---

## 9. Сводная схема взаимодействия

```
                    ┌──────────┐
                    │  users   │
                    └────┬─────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
         ▼               ▼               ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────────┐
│user_workspaces│ │user_module_  │ │user_role_        │
│(роль в WS)   │ │licenses      │ │assignments       │
└──────┬───────┘ └──────────────┘ └────────┬─────────┘
         │               │                 │
         └───────────────┼─────────────────┘
                         │
                         ▼
                ┌──────────────┐
                │ workspaces   │
                └──────┬───────┘
                       │
    ┌──────────────────┼──────────────────┐
    │                  │                  │
    ▼                  ▼                  ▼
┌─────────┐    ┌─────────────┐    ┌─────────────┐
│currencies│    │counterparties│    │workspace_   │
│         │    │             │    │modules      │
└─────────┘    └─────────────┘    └──────┬──────┘
                                         │
              ┌──────────────────────────┼──────────────────────────┐
              │                          │                          │
              ▼                          ▼                          ▼
       ┌────────────┐            ┌────────────┐            ┌────────────┐
       │ CRM        │            │ Habits     │            │ Notes      │
       │ (deals,    │            │ (habits,   │            │ Journal    │
       │ contacts)  │            │ completions)│            │ Tasks      │
       └────────────┘            └────────────┘            └────────────┘
```

---

## 10. Связанные документы

- [MODULES_LICENSING_GUIDE.md](../../modules/MODULES_LICENSING_GUIDE.md)
- [INVITE_FLOW.md](../../modules/INVITE/INVITE_FLOW.md)
- [README_PERMISSIONS_ROLES.md](../../SYSTEM/PERMISSIONS/README_PERMISSIONS_ROLES.md)
