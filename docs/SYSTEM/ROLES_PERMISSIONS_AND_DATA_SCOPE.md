# Роли, права (Casbin) и видимость данных (data scope)

**Навигация:** [README.md](./README.md) · полная модель RBAC и фронт: [PERMISSIONS/README_PERMISSIONS_ROLES.md](./PERMISSIONS/README_PERMISSIONS_ROLES.md).

Документ описывает **текущую** реализацию на бэкенде: таблицы, поток запроса, Casbin, middleware, слой **object scope**, масштабирование и границы ответственности.  
**Data scope — не отдельный микросервис**, а логика внутри монолита (`PermissionService` + доменные сервисы, например CRM).

---

## 1. Два слоя доступа

| Слой | Вопрос | Где решается |
|------|--------|----------------|
| **Авторизация действия (RBAC + домен)** | Имеет ли пользователь право вызвать операцию в этом workspace? (`crm:deal` + `read`) | `PermissionMiddleware` → Casbin (+ опционально `user_permissions`) |
| **Видимость строк (data scope)** | Какие **записи** возвращать в списке / виден ли конкретный ресурс? (`all` / `owner` / `department` / `none`) | `PermissionService.GetEffectiveDataScope` + репозиторий/сервис модуля (например фильтр в `DealList`) |

Оба слоя опираются на **одни и те же роли** (`user_role_assignments`), но **разные хранилища настроек**:

- Права на действия: **`casbin_rule`** (и каталог `permission_catalog`).
- Ограничение списков: **`role_object_scopes`** (ключ объекта в формате как у Casbin `obj`, например `crm:deal`).

---

## 2. Таблицы и связи

### 2.1. Основные таблицы

| Таблица | Назначение |
|---------|------------|
| `workspaces` | Workspace; поле **`owner_id`** — владелец (используется для гарантии полного data scope). |
| `workspace_roles` | Роли в workspace: системные (`OWNER`, `ADMIN`, `MEMBER`, `GUEST`) и кастомные. |
| `user_role_assignments` | Связь пользователь ↔ роль ↔ workspace (источник истины для ролей в рантайме). |
| `user_workspaces` | Членство в workspace + поле `role` (имя для UI/синхронизации). |
| `permission_catalog` | Справочник прав: `(module_code, entity_type, action)` и человекочитаемые имена. |
| `user_permissions` | Индивидуальные права пользователю в workspace (обход/дополнение к ролям). |
| `role_inheritance` | Наследование ролей child ← parent (в Casbin как `g(role:child, role:parent, workspace)`). Продукт может не использовать UI; таблица и API остаются. |
| `casbin_rule` | Политики Casbin: **`p(sub, dom, obj, act)`** и **`g(user, role, workspace)`** (и при необходимости `g` между ролями). |
| **`role_object_scopes`** | Per-role настройка видимости: `(role_id, object_key, data_scope)` где `object_key` например `crm:deal`. |
| **`users.department_id`** | Опциональный UUID для режима **`department`** (общий отдел у владельцев записей). Справочника отделов в БД может не быть — поле задаётся явно. |

### 2.2. Схема связей (упрощённо)

```
workspaces ─┬─ workspace_roles
            ├─ user_workspaces
            └─ user_role_assignments

users ──────┬─ user_workspaces
            ├─ user_role_assignments
            ├─ user_permissions
            └─ department_id (опционально)

workspace_roles ─┬─ user_role_assignments
                 ├─ role_object_scopes
                 └─ role_inheritance

casbin_rule ◄── синхронизируется из ролей/назначений (GORM adapter)
```

---

## 3. Встраивание в приложение: цепочка HTTP

Регистрация в **`internal/di/container.go`** → `RegisterRoutes`.

Типичный порядок для защищённых маршрутов `/api/v1/...`:

1. **`GinAuthMiddleware`** — JWT, в контекст кладутся `user_id`, глобальная роль (`USER` / `ADMIN`).
2. **`WorkspacePathMiddleware`** — для путей с `:workspaceId`: проверка членства / доступа к workspace (`AccessChecker`).
3. **`ModuleLicenseMiddleware`** — модуль включён и доступен для workspace.
4. **`PermissionMiddleware`** — маппинг `(method, path) → (obj, act)` и **`Enforce(user, workspace, obj, act)`** в Casbin; при отказе — проверка **`user_permissions`**; иначе **403** (с особыми правилами, например `workspace:module:read` + GET).

Доменные хендлеры (CRM и т.д.) вызываются **после** этой цепочки: к моменту входа в хендлер **действие** уже разрешено на уровне Casbin (если маршрут замаплен).

Файлы:

- Маппинг эндпоинтов: `internal/authz/endpoint_registry.go` (`MapEndpointToPermission`).
- Middleware: `internal/middleware/workspace_module_permission.go`.

---

## 4. Casbin

- **Инициализация:** `internal/authz/casbin.go` — модель с доменом `dom = workspace_id`, матчер через `g(r.sub, p.sub, r.dom)` и совпадение `obj`, `act`.
- **Субъект запроса:** `user:<uuid>`.
- **Политики `p`:** обычно на **`role:<NAME>`**, не на пользователя напрямую.
- **Связка пользователь–роль:** `g(user:…, role:…, workspace_id)` подгружается из БД при старте/синхронизации (`SyncGroupingPoliciesFromAssignments` и операции `AssignRole` / `RemoveRole`).

Глобальный **`ADMIN`** (поле `users.role`) в `PermissionMiddleware` **пропускается без Casbin** для любого workspace.

---

## 5. PermissionService (пакет `internal/service/permission`)

Отвечает за:

- CRUD ролей, назначения ролей, индивидуальные права;
- синхронизацию с Casbin (`AddPolicy`, `AddGroupingPolicy`, `SavePolicy`);
- **`GetEffectivePermissions`** — список строк `module:entity:action` для UI (`/me/permissions`);
- **`GetEffectiveDataScope(user, workspace, objectKey)`** — агрегированный data scope по ролям пользователя;
- **`ListRoleObjectScopes` / `SetRoleObjectScopes`** — API для UI (`GET/PUT .../roles/:id/object-scopes`).

### 5.1. Семантика прав для UI (override кастомной роли)

Если у пользователя есть **хотя бы одна кастомная** роль в workspace, при расчёте **`GetEffectivePermissions`** системные роли **не смешиваются** с кастомными (берутся права только кастомных + индивидуальные).

Та же идея применяется к **`GetEffectiveDataScope`**: при наличии кастомной роли **системные роли не участвуют** в расчёте scope. Между учтёнными ролями выбирается **наиболее широкий** scope: `all` > `department` > `owner` > `none`.

Если для роли **нет** строки в `role_object_scopes` для данного `object_key`, для этой роли подразумевается **`all`** (обратная совместимость).

### 5.2. Защита владельца и роли OWNER (data scope)

- Пользователь с **`workspaces.owner_id`** всегда получает **`GetEffectiveDataScope = all`** для любого `object_key` — нельзя «закрыть» данные владельцу через роли/scopes.
- Для системной роли **`OWNER`** запрещено менять object-scopes через API (**`ErrProtectedRoleObjectScopes`** / 403): полный доступ зашит политикой продукта.

Реализация: `internal/service/permission/scope.go`.

---

## 6. Data scope в домене (пример: CRM / сделки)

**Отдельного сервиса нет.** После прохождения middleware хендлер вызывает сервис CRM, который:

1. При необходимости вызывает **`PermissionService.GetEffectiveDataScope(ctx, userID, workspaceID, "crm:deal")`**.
2. Передаёт в репозиторий флаги/условия (`owner_id`, подзапрос по `department_id` и т.д.) — см. `internal/repository/crm/deal.go`, `internal/service/crm/service.go`.

Глобальный **`ADMIN`** в хендлере CRM может обходить ограничение scope отдельным флагом (как и для Casbin — согласованность с полным доступом).

Важно: **маскирование полей** (скрыть колонки в JSON) **не реализовано**; это следующий слой (метаданные роли + presenter), не middleware.

---

## 7. API и фронт (кратко)

- Права в workspace для текущего пользователя: **`GET /api/v1/me/permissions?workspaceId=...`**.
- Настройка видимости по роли: **`GET/PUT /api/v1/workspaces/:workspaceId/roles/:roleId/object-scopes`**  
  Тело PUT: `{ "objectScopes": { "crm:deal": "owner", ... } }`.
- UI: модалка роли (`RoleFormModal`) — блок «Видимость данных»; для роли OWNER блок отключён с пояснением.

---

## 8. Масштабирование и эксплуатация

- **PostgreSQL** — единое хранилище для приложения и `casbin_rule`.
- **Несколько инстансов API:** у каждого процесса свой in-memory Casbin после `LoadPolicy`. При изменении политик в одном инстансе остальные **не видят** изменения, пока не выполнится **`LoadPolicy()`** (или общий механизм инвалидации). Для production обычно: один инстанс на этапе разработки, при горизонтальном масштабировании — политика перезагрузки/Redis watcher (по необходимости).
- **Объём данных:** число строк в `casbin_rule` растёт с числом workspace × ролей × прав; `role_object_scopes` — мельче (роли × несколько object_key).
- **Data scope:** каждый запрос списка с фильтром — обычный SQL с индексами на `workspace_id`, `owner_id`, при необходимости `users.department_id`.

---

## 9. Чистота архитектуры и компромиссы

**Сильные стороны**

- Чёткое разделение: **middleware = «можно ли действие»**, **сервис/репозиторий = «какой срез данных»**.
- Casbin даёт единый язык для прав по модулям и workspace-домену.
- Каталог `permission_catalog` и маппинг эндпоинтов позволяют расширять API осознанно.

**Компромиссы**

- Маппинг `(method, path) → (obj, act)` **ручной** (`endpoint_registry`) — при добавлении маршрутов нужно не забыть правило.
- Два источника истины для «гибкости»: Casbin + `role_object_scopes` — в UI важно объяснять разницу (права vs видимость).
- Агрегация scope по ролям как **max(ширина)** при нескольких ролях — осознанный выбор; при сложных оргмоделях иногда нужен **min** или явный приоритет роли.
- Наследование ролей в БД/Casbin **есть**, но продукт может его не использовать — документация должна это отражать.

---

## 10. Как расширить новый модуль

1. Добавить права в **`permission_catalog`** (миграция).
2. Зарегистрировать правила в **`endpoint_registry.go`** для префиксов модуля.
3. При необходимости — политики системных ролей в **`addSystemPoliciesForWorkspace`** (или только кастомные роли).
4. Для ограничения списков: добавить строки в **`role_object_scopes`** (ключ `module:entity`), в сервисе/репозитории модуля вызвать **`GetEffectiveDataScope`** и применить `WHERE`.

---

## 11. Указатели на код

| Компонент | Путь |
|-----------|------|
| Casbin init | `internal/authz/casbin.go` |
| Маппинг эндпоинтов | `internal/authz/endpoint_registry.go` |
| Цепочка middleware | `internal/di/container.go` → `RegisterRoutes` |
| Permission middleware | `internal/middleware/workspace_module_permission.go` |
| Permission + scope логика | `internal/service/permission/service.go`, `scope.go` |
| Репозиторий прав/scopes | `internal/repository/permission/repository.go` |
| Миграция scopes | `migrations/000029_role_object_scopes.up.sql` |
| Handlers ролей / object-scopes | `internal/handler/permission/handler.go` |
| Пример внедрения scope в домен | `internal/service/crm/service.go`, `internal/repository/crm/deal.go` |

---

*Документ отражает состояние кодовой базы на момент добавления `role_object_scopes` и CRM-фильтрации сделок; при изменении логики обновляйте этот файл.*
