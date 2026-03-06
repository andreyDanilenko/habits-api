# Спецификация бэкенда системы прав: масштабирование, откат, гибкость

**Версия:** 1.0  
**Дата:** Март 2026  
**Связь:** [PERMISSIONS_FRONTEND.v1.md](./PERMISSIONS_FRONTEND.v1.md) — общий план и модель данных.

---

## 1. Цели документа

- **Зафиксировать фактическую реализацию** бэкенда ролей и прав (согласованную с `PERMISSIONS_FRONTEND.v1.md` и `perrmissions.v2.md`).
- **Дать фронтенду стабильный контракт**: какие сущности есть, как они связаны, какие есть эндпоинты и что они возвращают.
- **Описать поведение Casbin и middleware**: от HTTP‑запроса до решения allow/deny.
- **Строгая спецификация** для масштабирования, отката и возможного разбиения на сервисы.

---

## 2. Модель данных и Casbin

### 2.1. Таблицы и Go‑модели

Основные таблицы прав (см. миграции и `perrmissions.v2.md`):

- `permission_catalog` → `model.PermissionCatalogItem`  
  Словарь всех возможных прав в формате `module_code` + `entity_type` + `action`.  
  Примеры: `crm:deal:create`, `habits:habit:complete`, `workspace:role:manage`.  
  Метод `PermissionString()` возвращает строку `module:entity:action` для UI и Casbin.

- `workspace_roles` → `model.WorkspaceRole`  
  Роли в рамках одного workspace: системные (`OWNER`, `ADMIN`, `MEMBER`, `GUEST`) и кастомные.  
  Поля: `id`, `workspaceId`, `name`, `description`, `isSystem`, `createdAt`, `updatedAt`.

- `user_role_assignments` → `model.UserRoleAssignment`  
  Назначение ролей пользователям.  
  Поля: `id`, `userId`, `roleId`, `workspaceId`, `assignedBy`, `assignedAt`.

- `user_permissions` → `model.UserPermission`  
  Индивидуальные права для пользователя (минуя роли).  
  Поля: `id`, `userId`, `workspaceId`, `permissionId`, `grantedBy`, `grantedAt`, `expiresAt`.  
  Для API дополнительно подмешиваются: `moduleCode`, `entityType`, `action`, `permission` (строка `module:entity:action`).

- `role_inheritance` → `model.RoleInheritance`  
  Наследование ролей внутри workspace.  
  Поля: `id`, `workspaceId`, `childRoleId`, `parentRoleId`, `createdAt`.  
  Семантика: `childRole` наследует все права `parentRole` в данном workspace.

- `casbin_rule`  
  Хранилище политик Casbin (через GORM‑адаптер), структура стандартная:  
  `ptype='p'` — политика: `v0=sub`, `v1=dom`, `v2=obj`, `v3=act`;  
  `ptype='g'` — группировка: `v0`, `v1`, `v2` (субъект, родитель, домен).

Репозиторий `permission.Repository` закрывает эти таблицы и даёт сервису операции:

- Каталог: `ListCatalog`, `GetCatalogByID`.
- Роли: `ListRolesByWorkspace`, `GetRoleByID`, `GetRoleByName`, `CreateRole`, `UpdateRole`, `DeleteRole`, `CountAssignmentsByRole`.
- Назначения: `ListUserRoleAssignments`, `CreateUserRoleAssignment`, `DeleteUserRoleAssignment`, `ListAllUserRoleAssignments`.
- Индивидуальные права: `ListUserPermissions`, `CreateUserPermission`, `DeleteUserPermission`.
- Наследование: `ListRoleInheritanceByWorkspace`, `CreateRoleInheritance`, `DeleteRoleInheritance`.

### 2.2. Модель Casbin (фактическая)

Модель задана в `internal/authz/casbin.go`:

```ini
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act
```

- `sub` — субъект запроса:
  - для проверок прав: `"user:<userID>"`,
  - в политиках: `"role:<roleName>"`.
- `dom` — домен: `workspaceId` (UUID).
- `obj` — объект: `"<module>:<entity>"`, например `crm:deal`, `projects:project`, `workspace:role`.
- `act` — действие: `create|read|update|delete|manage|move|export|complete|attach|detach`.

### 2.3. Что хранится в Casbin

- **Политики ролей (`p`)**:
  - Для каждой роли и workspace записываются строки:  
    `p, role:<NAME>, <workspaceId>, <obj>, <act>`.  
  - Системные роли OWNER/ADMIN/MEMBER/GUEST получают базовые наборы прав через `EnsureSystemRolePolicies`/`addSystemPoliciesForWorkspace`.
  - Кастомные роли получают политики при `CreateRole`/`UpdateRole` на основе переданного списка строк `module:entity:action`.

- **Группировки пользователей (`g` для user→role)**:
  - `g, user:<userID>, role:<roleName>, <workspaceId>`  
    создаются при `AssignRole` и при старте через `SyncGroupingPoliciesFromAssignments`.

- **Группировки ролей (`g` для role→role, наследование)**:
  - `g, role:<child>, role:<parent>, <workspaceId>`  
    создаются на основе `role_inheritance`:
    - при вызове `AddRoleInheritance`;
    - при старте в `SyncGroupingPoliciesFromAssignments`, где помимо user→role поднимаются все связи из `role_inheritance`.

---

## 3. Поток запроса и middleware

Полная цепочка для защищённых маршрутов (см. `di.Container.RegisterRoutes`):

```text
[Request]
  → GinAuthMiddleware
  → WorkspacePathMiddleware
  → ModuleLicenseMiddleware
  → PermissionMiddleware    (пока в режиме логирования)
  → Handler (*_handler.go)
```

### 3.1. GinAuthMiddleware

- Извлекает токен из cookie `access_token` или заголовка `Authorization: Bearer ...`.
- Валидирует через `token.Generator.Validate`.
- Кладёт в контекст Gin:
  - `GinUserIDKey = "user_id"` — строка `userID`;
  - `GinRoleKey = "role"` — глобальная роль (`ADMIN`/`USER`) как `model.UserRole`.
- Хелперы:
  - `GetUserIDFromGin(c) (string, bool)`;
  - `GetAuthFromGin(c) (userID string, role model.UserRole, ok bool)` — единая точка для чтения `user_id` и глобальной роли.

### 3.2. WorkspacePathMiddleware

- Опирается на паттерн URL `/api/v1/workspaces/:workspaceId/...`.
- Извлекает `workspaceId` из параметров Gin.
- Через `WorkspaceService.HasAccess(ctx, workspaceID, userID, userRole)` проверяет:
  - членство пользователя в workspace;
  - глобальный ADMIN имеет доступ ко всем workspace.
- Кладёт `workspaceId` в контекст Gin (`GinWorkspaceIDKey = "workspace_id"`).  
  Если доступа нет — возвращает 403.

### 3.3. ModuleLicenseMiddleware

- По пути (`/crm/`, `/habits/`, `/projects/` и т.д.) определяет `moduleCode`.
- Через `WorkspaceService.GetWorkspaceModules` и `CanEnableModuleInWorkspace` проверяет:
  - включён ли модуль в данном workspace;
  - есть ли у пользователя лицензия (для не core‑модулей).
- При отсутствии доступа — 403 «Module not enabled / No license».

### 3.4. PermissionMiddleware

- Работает **в режиме логирования**:
  - по `c.FullPath()` и HTTP‑методу вызывает `mapEndpointToPermission`,
  - получает `(obj, act)` из таблицы `endpointPermissionTable` (workspace‑админка + CRM/Habits/Projects),
  - собирает `sub = "user:"+userID`, `dom = workspaceID`,
  - вызывает `enforcer.Enforce(sub, dom, obj, act)`,
  - логирует результат (allow/deny), но **не блокирует** запрос.
- После включения боевого режима логика будет той же, но при `allowed=false` и отсутствии индивидуального права — возврат `403 Forbidden`.

---

## 4. Сервисы, роли и сценарии

### 4.1. PermissionService (ядро ролей и прав)

Сервис `internal/service/permission.Service` поверх репозитория и Casbin реализует:

- Каталог:
  - `GetCatalog(ctx)` — возвращает `[]PermissionCatalogItem` для UI.

- Роли:
  - `ListRoles(ctx, workspaceID)` — список ролей workspace.
  - `GetRole(ctx, roleID)` — одна роль.
  - `GetRolePermissions(ctx, roleID)` — права роли как строки `module:entity:action` (из Casbin).
  - `CreateRole(ctx, workspaceID, name, description, permissions, createdBy)`:
    - валидация имени и уникальности в рамках workspace;
    - создание записи в `workspace_roles`;
    - добавление политик `p(role:<name>, workspaceId, obj, act)` для всех прав;
    - при ошибке AddPolicy/SavePolicy — **компенсирующий откат**: удаление роли из БД.
  - `UpdateRole(ctx, roleID, name, description, permissions)`:
    - запрет изменения системных ролей;
    - при смене имени: проверка уникальности, обновление в БД;
    - удаление всех политик по старому имени роли в данном workspace;
    - добавление политик по новому имени;
    - `SavePolicy()`.
  - `DeleteRole(ctx, roleID)`:
    - запрет удаления системных ролей;
    - проверка отсутствия назначений (`CountAssignmentsByRole`);
    - удаление всех `p` для `role:<name>` в любом домене;
    - `SavePolicy()` и только затем удаление роли из БД.

- Назначения ролей:
  - `AssignRole(ctx, userID, roleID, workspaceID, assignedBy)`:
    - проверка, что роль принадлежит workspace;
    - создание записи в `user_role_assignments`;
    - добавление `g("user:<userID>", "role:<roleName>", workspaceID)`;
    - при ошибке AddGroupingPolicy/SavePolicy — откат: удаление назначения из БД и (если нужно) удаление grouping policy.
  - `RemoveRole(ctx, userID, roleID, workspaceID)`:
    - удаление записи `user_role_assignments`;
    - удаление `g("user:<userID>", "role:<roleName>", workspaceID)`;
    - `SavePolicy()`.

- Наследование ролей:
  - `AddRoleInheritance(ctx, workspaceID, childRoleID, parentRoleID)`:
    - проверка, что обе роли существуют и принадлежат одному workspace, и что `child != parent`;
    - добавление записи в `role_inheritance`;
    - добавление `g("role:<child>", "role:<parent>", workspaceID)` и `SavePolicy()`;
    - при ошибке — удаление записи из `role_inheritance`.
  - `RemoveRoleInheritance(ctx, workspaceID, childRoleID, parentRoleID)`:
    - удаление записи из `role_inheritance`;
    - удаление `g("role:<child>", "role:<parent>", workspaceID)` и `SavePolicy()`.

- Индивидуальные права:
  - `GrantPermission(ctx, userID, workspaceID, permissionID, grantedBy, expiresAt)`:
    - валидация наличия права в `permission_catalog`;
    - запись в `user_permissions` (для аудита и `/me/permissions`);
    - (пока **не** дублируется в Casbin).
  - `RevokePermission(ctx, userID, workspaceID, permissionID)` — удаление записи из `user_permissions`.
  - `GetUserPermissions(ctx, userID, workspaceID)` — полный список индивидуальных прав (для UI админки).

- Эффективные права:
  - `GetEffectivePermissions(ctx, userID, workspaceID) ([]string, error)`:
    - собирает роли пользователя в workspace (`GetUserRoles`);
    - по каждой роли читает политики `p` из Casbin и формирует строки `module:entity:action`;
    - добавляет индивидуальные права из `user_permissions` (по `PermissionStr`);
    - возвращает множество всех прав (без дублей).

- Системные политики и старт:
  - `EnsureSystemRolePolicies(ctx)` и `SeedSystemPoliciesForWorkspace(workspaceID)` — заливают базовые наборы прав для OWNER/ADMIN/MEMBER/GUEST (см. массивы `ownerAdminPolicies`, `memberBasePolicies`, `guestBasePolicies`).
  - `SyncGroupingPoliciesFromAssignments(ctx)`:
    - очищает все `g` в Casbin;
    - пересоздаёт g(user→role) по всем `user_role_assignments`;
    - пересоздаёт g(role→role) по всем `role_inheritance` во всех workspace;
    - сохраняет политики.

### 4.2. HTTP‑контракт (для фронтенда)

Полный список эндпоинтов задокументирован в `DOC_ALL_METHODS_BACK.md`. Ключевые для прав и ролей:

- Каталог прав:
  - `GET /api/v1/workspaces/:workspaceId/permissions/catalog`  
    → сгруппированный по модулям/сущностям каталог + сырые записи `permission_catalog`.

- Роли:
  - `GET /api/v1/workspaces/:workspaceId/roles` — список ролей.
  - `POST /api/v1/workspaces/:workspaceId/roles` — создание роли (`name`, `description`, `permissions[]`).
  - `GET /api/v1/workspaces/:workspaceId/roles/:roleId` — роль.
  - `PUT /api/v1/workspaces/:workspaceId/roles/:roleId` — обновление роли и её прав.
  - `DELETE /api/v1/workspaces/:workspaceId/roles/:roleId` — удаление кастомной роли без назначений.
  - `GET /api/v1/workspaces/:workspaceId/roles/:roleId/permissions` — права роли.
  - `POST /api/v1/workspaces/:workspaceId/roles/:roleId/inherit/:parentRoleId` — добавить наследование.
  - `DELETE /api/v1/workspaces/:workspaceId/roles/:roleId/inherit/:parentRoleId` — удалить наследование.

- Назначения ролей и индивидуальные права:
  - `GET /api/v1/workspaces/:workspaceId/users/:userId/roles` — роли пользователя.
  - `POST /api/v1/workspaces/:workspaceId/users/:userId/roles/:roleId` — назначить роль.
  - `DELETE /api/v1/workspaces/:workspaceId/users/:userId/roles/:roleId` — снять роль.
  - `GET /api/v1/workspaces/:workspaceId/users/:userId/permissions` — индивидуальные права.
  - `POST /api/v1/workspaces/:workspaceId/users/:userId/permissions` — выдать право.
  - `DELETE /api/v1/workspaces/:workspaceId/users/:userId/permissions/:permissionId` — отозвать право.

- Права текущего пользователя:
  - `GET /api/v1/me/permissions?workspaceId=...`  
    → JSON: `{ permissions: string[], roles: string[], systemRole: 'OWNER'|'ADMIN'|'MEMBER'|'GUEST' }`.

Фронтенд может трактовать эти данные ровно как описано в `PERMISSIONS_FRONTEND.v1.md`: `permissions[]` — полный набор строк `module:entity:action`, `roles[]` — имена ролей в workspace, `systemRole` — системная роль статуса участника.

---

## 5. Текущее состояние и ограничения

- **Ролевой функционал на бэкенде завершён:**
  - модель данных и миграции;
  - репозиторий и сервис прав;
  - системные и кастомные роли;
  - назначение ролей и индивидуальных прав;
  - наследование ролей через `role_inheritance` и Casbin `g(role→role)`;
  - эндпоинты для каталога, ролей, назначений, индивидуальных прав и `/me/permissions`.
- **PermissionMiddleware** пока в режиме **логирования**:
  - решения Casbin логируются, но не блокируют доступ;
  - включение боевого режима (403 при отсутствии прав) запланировано как отдельный шаг.
- **Индивидуальные права** пока не синхронизируются в Casbin:
  - они учитываются в `GetEffectivePermissions` и на фронте;
  - при необходимости их можно добавить как отдельные `p` с `sub="user:<id>"`.
- **Кэширование прав** пока отсутствует; его можно включить согласно рекомендациям ниже.

---

## 6. Масштабирование бэкенда

### 2.1. Текущие ограничения (монолит)

| Компонент | Ограничение | Метрика |
|-----------|-------------|---------|
| Проверка прав | Один инстанс Casbin на процесс | Нет горизонтального масштабирования проверки без общих политик |
| БД | Одна PostgreSQL | Рост таблиц `casbin_rule`, `user_role_assignments`, `user_permissions` |
| Запросы прав | GetEffectivePermissions без кэша | Рост времени ответа при большом числе ролей/прав |

### 2.2. Спецификация масштабирования

#### 2.2.1. Горизонтальное масштабирование (несколько инстансов API)

- **Политики Casbin** хранятся в PostgreSQL (`casbin_rule`). Каждый инстанс при старте загружает политики через `LoadPolicy()`.
- **Требование:** после любого изменения политик (CreateRole, UpdateRole, AssignRole, RemoveRole, GrantPermission, RevokePermission) вызывается `SavePolicy()`. При нескольких инстансах другие инстансы **не видят** изменения до перезагрузки политик.
- **Спецификация масштабирования:**
  1. **Вариант A (короткий срок):** TTL-кэш GetEffectivePermissions (in-memory, 1–5 мин) + инвалидация по событию при изменении прав (очередь/бродкаст не обязательны на первом этапе).
  2. **Вариант B (рекомендуемый при >1 инстансе):** после каждого `SavePolicy()` публиковать событие «политики обновлены» (Redis Pub/Sub или очередь). Все инстансы по подписке вызывают `Enforcer.LoadPolicy()` для данного Enforcer.
  3. **Вариант C:** вынести проверку прав в отдельный сервис (см. раздел 4), который один раз держит Casbin в памяти и масштабируется через кэш/реплики.

#### 2.2.2. Масштабирование данных

- **Индексы (обязательны):**  
  `user_role_assignments(user_id, workspace_id)`,  
  `user_permissions(user_id, workspace_id)`,  
  `casbin_rule(ptype, v0, v1, v2, v3)` — согласно PERMISSIONS_FRONTEND.v1.md.
- **Целевое время проверки прав:** < 5 мс (замер по перцентилям p95).
- При росте таблицы `casbin_rule` (> 100k строк) рассмотреть партиционирование по `v1` (domain/workspace_id) или вынос политик в отдельное хранилище с кэшем.

#### 2.2.3. Масштабирование по нагрузке

- **Чтение каталога прав:** `GET /permissions/catalog` — редко меняется; допускается кэш на уровне приложения или HTTP (Cache-Control).
- **Чтение «мои права»:** `GET /me/permissions?workspaceId=...` — частый запрос; кэширование на фронте (как в PERMISSIONS_FRONTEND.v1.md) + на бэке TTL-кэш с инвалидацией при изменении назначений/прав.

---

## 7. Откат (rollback)

### 3.1. Откат релиза приложения

- **Политики** лежат в БД (`casbin_rule`). Откат кода на предыдущую версию **не откатывает** уже записанные в БД политики.
- **Спецификация:**
  1. Миграции БД (таблицы прав, каталог, casbin_rule) должны быть **обратимы**: для каждой миграции «вперёд» описать миграцию «назад» (удаление/откат таблиц или полей). Хранить скрипты отката рядом с миграциями (например, `migrations/000022_permissions_down.sql`).
  2. При откате кода на версию **до** введения системы прав: выполнить скрипты отката миграций вручную; старый код не должен падать при наличии новых таблиц (можно оставить таблицы, но не использовать их).
  3. **Не откатывать** данные `user_role_assignments` и `user_permissions` автоматически при деплое; откат данных — отдельная процедура из бэкапов.

### 3.2. Откат при сбое Casbin

- **Сценарий:** после CreateRole/AssignRole и т.п. запись в БД прошла, а `SavePolicy()` упал или таймаут.
- **Текущее поведение (спецификация):**
  - CreateRole: при сбое AddPolicy/SavePolicy выполняется **компенсирующий откат** — удаление созданной роли из БД.
  - AssignRole: при сбое AddGroupingPolicy/SavePolicy — удаление назначения из БД и, при необходимости, RemoveGroupingPolicy.
  - UpdateRole: при смене имени удаляются политики по старому имени, добавляются по новому; при сбое SavePolicy ошибка возвращается клиенту (состояние БД уже обновлено; повтор вызова допустим для идемпотентности перезаписи политик).
- **Рекомендация:** логировать все сбои SavePolicy и иметь фоновую job «синхронизация Casbin с БД»: перечитать `workspace_roles`, `user_role_assignments`, `user_permissions` и пересобрать политики для затронутого workspace (или глобально `SyncGroupingPoliciesFromAssignments` + перезалив политик ролей).

### 3.3. Восстановление после сбоя БД

- Регулярные бэкапы PostgreSQL включают `casbin_rule`, `workspace_roles`, `user_role_assignments`, `user_permissions`, `permission_catalog`.
- После восстановления из бэкапа: перезапуск приложения; при старте вызываются `EnsureSystemRolePolicies` и `SyncGroupingPoliciesFromAssignments` — политики в Casbin приводятся в соответствие с БД.

---

## 8. Гибкость и разбиение на микросервисы

### 4.1. Принципы

- Изменения вводить **без «большого взрыва»**: сначала монолит с чёткими границами модулей, затем при необходимости вынос в отдельные сервисы.
- **Контракты API** для прав и ролей должны быть стабильными (версионирование `/api/v1/...`), чтобы фронт и возможный будущий Authz-сервис не ломались.

### 4.2. Кандидаты на выделение

| Кандидат | Описание | Зависимости | Сложность выноса |
|----------|----------|-------------|-------------------|
| **Authz-сервис** | Только проверка прав: вход (userID, workspaceID, obj, act), выход (allow/deny). Хранит Casbin и/или кэш. | БД (casbin_rule), при необходимости подписка на события изменений ролей | Средняя: текущий PermissionMiddleware и PermissionService вызывают Enforcer; нужен HTTP/gRPC вызов к Authz-сервису. |
| **Role & Permission Admin API** | CRUD ролей, назначения ролей, индивидуальные права, каталог. | БД (workspace_roles, user_role_assignments, user_permissions, permission_catalog), Casbin или Authz-сервис для записи политик | Средняя: отдельный сервис с тем же API, что и текущие handler'ы permission. |
| **Workspace API** | Workspace CRUD, участники, модули. | БД (workspaces, user_workspaces, workspace_modules и т.д.) | Уже логически выделен; вынос — по необходимости. |
| **Модули (CRM, Habits, Projects и т.д.)** | Каждый модуль — отдельный сервис. | Workspace/Authz для проверки доступа | Высокая: много эндпоинтов и общей логики; выносить по одному модулю. |

### 4.3. Идеи для гибкости без немедленного разбиения

1. **Единая точка проверки прав в коде:** только PermissionMiddleware + Casbin (или в будущем вызов Authz-сервиса). Хендлеры не дублируют проверки по ролям для одних и тех же сущностей.
2. **События домена (опционально):** при создании/обновлении/удалении роли или назначении публиковать событие (RoleCreated, RoleUpdated, RoleAssigned, …). Позволит в будущем подписывать Authz-сервис и инвалидировать кэши без жёсткой связки.
3. **Конфигурируемый маппинг endpoint → право:** таблица `endpointPermissionTable` в middleware уже вынесена в данные; при переходе на микросервисы маппинг может жить в конфиге или в Authz-сервисе.
4. **Единый формат прав:** строка `module:entity:action` и каталог в БД — единый источник истины для бэка и фронта; при разбиении сервисов контракт не меняется.

### 4.4. Ограничения при разбиении

- **Системные роли (OWNER, ADMIN, MEMBER, GUEST)** и их базовые политики задаются в коде (или конфиге) и при старте заливаются в Casbin. При выделении Authz-сервиса эта логика должна жить в нём или в отдельном «наполнителе» политик.
- **Транзакции БД + Casbin:** в монолите при сбое SavePolicy выполняется компенсирующий откат БД. В архитектуре с отдельным Authz-сервисом потребуется либо 2PC/сага, либо идемпотентность и повторные вызовы с перезаписью политик по данным из БД.

---

## 9. Критерии приёмки спецификации

- [ ] Документированы шаги масштабирования API (несколько инстансов) и поведение кэша политик/прав.
- [ ] Описаны процедуры отката релиза и отката при сбое Casbin; есть скрипты отката миграций.
- [ ] Перечислены кандидаты на вынос в сервисы и зависимости между ними.
- [ ] Сохранена совместимость с [PERMISSIONS_FRONTEND.v1.md](./PERMISSIONS_FRONTEND.v1.md) (модель данных, форматы прав, цели внедрения).

---

**Статус:** Утверждено к использованию при масштабировании и планировании откатов.
