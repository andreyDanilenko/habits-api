## Обзор системы прав и ролей

**Цель документа** — дать разработчикам и администраторам workspace одно место, где полностью описано:

- **как устроены права (`permissions`) и роли** на бэкенде и фронтенде;
- **какие есть системные и кастомные роли**, как они связаны с правами;
- **как настраиваются роли и индивидуальные права** через API и UI;
- **как именно производится проверка доступа** (middleware, Casbin, фронтовые хуки и компоненты);
- **какие есть нюансы и ограничения**.

Документ опирается на спецификации:

- `PERMISSIONS_FRONTEND.v1.md` — системный анализ и целевое состояние;
- `SPEC_ROLE_BACK.md` — подробная спецификация реализации бэкенда ролей и прав;
- код бэкенда в `internal/service/permission`, `internal/authz/casbin.go`, `internal/middleware/*`;
- код фронтенда в `src/features/permissions`, `src/features/roles`, `src/features/user-permissions`, `src/features/auth`.

Этот README описывает **фактическую реализованную модель**, с привязкой к методам и компонентам.

---

## Базовые понятия

- **Workspace** — логическая «организация»/аккаунт, внутри которого настраиваются роли и права.
- **Глобальная роль пользователя** (`users.role`) — `ADMIN` или `USER`.
  - `ADMIN` имеет доступ ко всем workspaces.
  - `USER` имеет доступ только к тем workspaces, где он участник.
- **Системная роль workspace** (`OWNER`, `ADMIN`, `MEMBER`, `GUEST`) — статус участника в конкретном workspace.
- **Кастомная роль** — роль, созданная администратором workspace. Хранится в `workspace_roles` с `isSystem=false`.
- **Права (permissions)** — атомарные единицы доступа вида:
  - `moduleCode:entityType:action`, например:
  - `crm:deal:create`, `crm:deal:read`, `habits:habit:complete`, `workspace:role:manage`.
- **Индивидуальные права** — права, выданные конкретному пользователю, в обход ролей (но всё равно из общего каталога).

Формат строки права соответствует методу:

- `backend/internal/model/permission.go` → `PermissionCatalogItem.PermissionString()`  
  Возвращает строку `module:entity:action` и является **единым форматом** для:
  - бэкенда (Casbin, API);
  - фронтенда (тип `PermissionString` и утилиты `usePermissions`, `PermissionGuard`, `PermissionTree`).

---

## Модель данных

### Каталог прав: `permission_catalog`

Модель: `model.PermissionCatalogItem`.

- Поля:
  - `id` — внутренний ID права.
  - `moduleCode` — код модуля (`crm`, `habits`, `projects`, `workspace`).
  - `entityType` — тип сущности (`deal`, `contact`, `habit`, `project`, …).
  - `action` — действие (`create`, `read`, `update`, `delete`, `move`, `manage`, `complete`, …).
  - `name`, `description` — человекочитаемое название и описание для UI.
  - `isSystem` — системное право (не редактируется).
- Гарантия уникальности: `(moduleCode, entityType, action)` уникальны.
- Метод `PermissionString()` возвращает строку `moduleCode:entityType:action`.  
  Эта строка:
  - отображается во фронте;
  - используется при создании ролей;
  - используется для проверки прав в Casbin.

### Роли workspace: `workspace_roles`

Модель: `model.WorkspaceRole`.

- Хранит **все роли** в рамках workspace:
  - системные: `OWNER`, `ADMIN`, `MEMBER`, `GUEST` (`isSystem=true`);
  - кастомные: произвольные роли администратора (`isSystem=false`).
- Ключевые поля:
  - `workspaceId` — к какому workspace относится роль;
  - `name` — имя роли (уникально в паре `(workspaceId, name)`);
  - `description` — описание роли;
  - `isSystem` — флаг системной роли;
  - `createdAt`, `updatedAt`.

**Нельзя**:

- изменять или удалять системные роли (ограничения на уровне сервиса `PermissionService`).

### Назначения ролей: `user_role_assignments`

Модель: `model.UserRoleAssignment`.

- Связывает пользователя, роль и workspace:
  - `userId`
  - `roleId`
  - `workspaceId`
  - `assignedBy`, `assignedAt` — кто и когда назначил.
- Один пользователь может иметь **несколько кастомных ролей** в рамках одного workspace.
- Системная роль (`OWNER`/`ADMIN`/`MEMBER`/`GUEST`) **не задаётся здесь напрямую** — она определяется:
  - по данным `user_workspaces` и логике workspace-сервиса;
  - и возвращается на фронт полем `systemRole` в `/me/permissions`.

### Индивидуальные права: `user_permissions`

Модель: `model.UserPermission`.

- Даёт право конкретному пользователю, минуя роль:
  - `userId`, `workspaceId`;
  - `permissionId` → ссылка на `permission_catalog`;
  - `grantedBy`, `grantedAt`, `expiresAt`.
- В API дополняется полями:
  - `moduleCode`, `entityType`, `action`, `permission` (строка `module:entity:action`).

**Важно:**  
Индивидуальные права:

- учитываются в методе `GetEffectivePermissions` и в UI (например, в `UserPermissionsPanel.vue`);
- пока **не дублируются** в Casbin как отдельные политики (можно добавить позже, если нужно).

### Наследование ролей: `role_inheritance`

Модель: `model.RoleInheritance`.

- Описывает связь `childRole` ← наследует `parentRole`:
  - `workspaceId`
  - `childRoleId`
  - `parentRoleId`
- На уровне Casbin наследование реализовано как **групповые политики**:
  - `g("role:<child>", "role:<parent>", workspaceId)`.
- Сервис `PermissionService`:
  - создаёт/удаляет такие записи;
  - синхронизирует их в Casbin через `AddRoleInheritance`, `RemoveRoleInheritance` и `SyncGroupingPoliciesFromAssignments`.

### Политики Casbin: `casbin_rule`

Используется стандартная структура адаптера Casbin (`ptype`, `v0`..`v5`).

- Политики (`ptype='p'`):
  - `v0=sub`, `v1=dom`, `v2=obj`, `v3=act`.
- Группировки (`ptype='g'`):
  - `v0` — субъект;
  - `v1` — родитель (роль или другая роль);
  - `v2` — домен (workspaceId).

---

## Casbin: модель и интерпретация

Фактическая модель в `internal/authz/casbin.go`:

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

Интерпретация полей:

- `sub` (subject):
  - в запросе (`r.sub`) — `"user:<userID>"`;
  - в политике (`p.sub`) — `"role:<roleName>"`;
  - в группировке (`g`) — `"user:<userID>"` или `"role:<child>"`.
- `dom` (domain) — `workspaceId` (UUID).
- `obj` (object) — строка `"<module>:<entity>"`, например `crm:deal`, `projects:project`, `workspace:member`.
- `act` (action) — строка действия (`create`, `read`, `update`, `delete`, `manage`, `move`, `export`, `complete`, `attach`, `detach` и т.д.).

**Связь с `PermissionString`:**

- `PermissionString = "module:entity:action"`;
- при разборе разрешения `parsePermission` делит строку на:
  - `obj = "module:entity"`,
  - `act = "action"`.

---

## Системные и кастомные роли

### Системные роли workspace

Список фиксирован:

- `OWNER` — владелец workspace;
- `ADMIN` — администратор;
- `MEMBER` — обычный участник;
- `GUEST` — гость (read‑only, где это возможно).

Системные роли:

- создаются автоматически для каждого workspace (см. миграции и логику `EnsureSystemRolePolicies`/`SeedSystemPoliciesForWorkspace`);
- имеют предопределённый набор политик (`ownerAdminPolicies`, `memberBasePolicies`, `guestBasePolicies` в `permission/service.go`);
- **нельзя удалить или редактировать** (любая попытка изменить — приводит к ошибке `ErrRoleSystem`).

Примеры встроенных прав:

- `OWNER` и `ADMIN`:
  - полный доступ к CRM: `crm:deal` (create/read/update/delete/move), `crm:contact`, `crm:company`, `crm:pipeline:manage`, `crm:activity`, `crm:export:deals`;
  - полный доступ к `habits:*`, `projects:*`;
  - управление workspace: `workspace:member:invite`, `workspace:member:remove`, `workspace:role:manage`, `workspace:module:manage`.
- `MEMBER`:
  - права на создание и редактирование **своих рабочих сущностей** (CRM, Habits, Projects);
  - базовый админ‑функционал по участникам/модулям (ограниченнее, чем у ADMIN/OWNER).
- `GUEST`:
  - только чтение ключевых сущностей (`crm:deal:read`, `crm:contact:read`, `crm:company:read`, `habits:habit:read`, `projects:project:read` и т.п.).

Фактическое наполнение можно посмотреть в массивах:

- `ownerAdminPolicies`
- `memberBasePolicies`
- `guestBasePolicies`

в `internal/service/permission/service.go`.

### Кастомные роли

Кастомная роль:

- создаётся в конкретном workspace (`workspaceId`);
- имеет произвольное имя (уникальное в пределах workspace) и описание;
- состоит из **списка прав** вида `module:entity:action`.

Жизненный цикл:

1. **Создание**:
   - UI: модалка `RoleFormModal.vue` + дерево прав `PermissionTree.vue`;
   - фронт вызывает `POST /api/v1/workspaces/:workspaceId/roles`;
   - бэкенд: `PermissionService.CreateRole`.
2. **Редактирование**:
   - UI: та же модалка, но с заполненными данными;
   - фронт:
     - сначала загружает права роли через `roleService.getPermissions(workspaceId, roleId)` (`GET /roles/:roleId/permissions`);
     - затем отправляет `PUT /roles/:roleId`;
   - бэкенд: `PermissionService.UpdateRole`.
3. **Удаление**:
   - возможно **только если у роли нет назначенных пользователей** (`ErrRoleHasUsers`);
   - бэкенд: `PermissionService.DeleteRole`:
     - сначала удаляет все политики роли в Casbin;
     - затем удаляет запись из БД.

Ограничения:

- кастомные роли **нельзя пометить как системные**;
- имена ролей уникальны в рамках одного workspace;
- при смене имени роли:
  - проверяется уникальность;
  - обновляются политики в Casbin (удаляются старые, создаются новые).

---

## Индивидуальные права пользователя

Индивидуальные права позволяют:

- выдать конкретному пользователю дополнительное право (например, временный экспорт сделок, не меняя его роли);
- повысить уровень доступа отдельного сотрудника без создания новой роли.

### Работа на бэкенде

Методы сервиса `PermissionService`:

- `GrantPermission(ctx, userID, workspaceID, permissionID, grantedBy, expiresAt)`:
  - проверяет, что `permissionID` существует в `permission_catalog`;
  - создаёт запись в `user_permissions`;
  - пока **не** создаёт отдельную политику Casbin (источником индивидуальных прав служит БД).
- `RevokePermission(ctx, userID, workspaceID, permissionID)`:
  - удаляет запись в `user_permissions`.
- `GetUserPermissions(ctx, userID, workspaceID)`:
  - возвращает список `UserPermission` с заполненным `PermissionStr` для UI.

В расчёте эффективных прав:

- `GetEffectivePermissions`:
  - берёт все права из ролей пользователя (через Casbin);
  - добавляет все строки `PermissionStr` из `user_permissions`;
  - возвращает уникальный список строк `module:entity:action`.

### Работа на фронтенде

UI‑компонент: `UserPermissionsPanel.vue`.

- Загружает через `useUserPermissions`:
  - доступные права для workspace (каталог);
  - текущие индивидуальные права пользователя;
  - статус загрузки.
- Позволяет:
  - выбрать право из списка и вызвать `grantPermission(permissionId)`;
  - отозвать право через `revokePermission(permissionId)`.
- После операции список автоматически обновляется.

На UI это выглядит как отдельный блок «Индивидуальные права» для конкретного пользователя, с выпадающим списком каталога прав и кнопками «Выдать право»/«Отозвать».

---

## Наследование ролей

Наследование ролей позволяет описать иерархию:

- например, `Senior Manager` наследует `Manager` и добавляет несколько прав сверху;
- изменения в базовой роли автоматически распространяются на всех наследников.

Бэкенд:

- `PermissionService.AddRoleInheritance(workspaceID, childRoleID, parentRoleID)`:
  - валидация:
    - роли существуют;
    - принадлежат одному workspace;
    - `child != parent`;
  - создаёт запись в `role_inheritance`;
  - добавляет группировку `g("role:<child>", "role:<parent>", workspaceId)` в Casbin;
  - сохраняет политики.
- `PermissionService.RemoveRoleInheritance(...)`:
  - удаляет запись в `role_inheritance`;
  - удаляет соответствующую `g`‑политику;
  - сохраняет.
- `PermissionService.SyncGroupingPoliciesFromAssignments()`:
  - очищает все групповые политики;
  - заново заливает:
    - `g("user:<id>", "role:<name>", workspaceId)` из `user_role_assignments`;
    - `g("role:<child>", "role:<parent>", workspaceId)` из `role_inheritance`.

Casbin‑matcher учитывает эти группировки, поэтому:

- права родительской роли автоматически доступны всем дочерним.

---

## Поток проверки доступа (бэкенд)

Маршрутизатор (`di.Container.RegisterRoutes`) собирает цепочку middleware:

1. **`GinAuthMiddleware`** (`internal/middleware/gin_auth.go`):
   - читает токен из cookie `access_token` или заголовка `Authorization`;
   - валидирует;
   - кладёт в контекст:
     - `user_id` (`GinUserIDKey`);
     - глобальную роль (`ADMIN`/`USER`) — `GinRoleKey`.
2. **`WorkspacePathMiddleware`**:
   - извлекает `workspaceId` из URL (`/api/v1/workspaces/:workspaceId/...`);
   - через `WorkspaceService.HasAccess` проверяет:
     - членство пользователя;
     - право глобального ADMIN зайти в любой workspace;
   - помещает `workspaceId` в контекст.
3. **`ModuleLicenseMiddleware`**:
   - определяет `moduleCode` по префиксу пути (`/crm/`, `/habits/`, `/projects/` и др.);
   - проверяет, включён ли модуль в workspace и есть ли лицензия у пользователя (для не core‑модулей).
4. **`PermissionMiddleware`** (`internal/middleware/workspace_module_permission.go`):
   - по `c.FullPath()` и HTTP‑методу находит запись в таблице `endpointPermissionTable`;
   - получает `obj` и `act` (`crm:deal` + `create` и т.п.);
   - собирает параметры для Casbin:
     - `sub = "user:" + userID`;
     - `dom = workspaceId`;
     - `obj`, `act`.
   - вызывает `enforcer.Enforce(sub, dom, obj, act)`.

**Текущее состояние:**

- `PermissionMiddleware` работает **в режиме логирования**:
  - результат `allow/deny` логируется;
  - доступ **не блокируется**, даже при отсутствии прав.

**Боевой режим (план):**

- при `allowed=false` и отсутствии индивидуального права:
  - запрос завершается `403 Forbidden`;
  - хендлер не вызывается.

---

## Сервис прав `PermissionService`: основные методы

Реализация: `backend/internal/service/permission/service.go`.

Ниже — **описание каждого публичного метода**, что он принимает, что возвращает, какую ответственность несёт и где используется.

### Каталог

**Метод:** `GetCatalog(ctx context.Context) ([]model.PermissionCatalogItem, error)`

- **Что делает:** читает все записи из таблицы `permission_catalog` и возвращает их как плоский список.
- **Что возвращает:**
  - `[]PermissionCatalogItem` — полный каталог прав (ID, `moduleCode`, `entityType`, `action`, `name`, `description`, `isSystem`, `createdAt`).
- **Кто использует:**
  - HTTP‑эндпоинт `GET /api/v1/workspaces/:workspaceId/permissions/catalog`;
  - фронтовый хук `usePermissionTree` и компонент `PermissionTree.vue` для построения дерева прав.
- **За что отвечает:**
  - предоставляет **эталонный список всех возможных прав** в системе;
  - служит источником истины для:
    - построения дерева прав в UI;
    - сравнения текущей конфигурации ролей с доступными правами.

### Роли

#### `ListRoles`

**Метод:** `ListRoles(ctx context.Context, workspaceID string) ([]model.WorkspaceRole, error)`

- **Что делает:** отдаёт все роли (системные и кастомные) в рамках одного workspace.
- **Что возвращает:** массив `WorkspaceRole` с полями `id`, `workspaceId`, `name`, `description`, `isSystem`, `createdAt`, `updatedAt`.
- **Кто использует:**
  - `GET /api/v1/workspaces/:workspaceId/roles`;
  - компонент `RolesList.vue` (страница «Роли workspace»).
- **За что отвечает:** даёт **снимок списка ролей**, но **без прав**; права загружаются отдельным методом.

#### `GetRole`

**Метод:** `GetRole(ctx context.Context, roleID string) (*model.WorkspaceRole, error)`

- **Что делает:** находит и возвращает конкретную роль по её `id`.
- **Что возвращает:** объект `WorkspaceRole` или ошибку `ErrRoleNotFound`.
- **Кто использует:** эндпоинт получения одной роли (например, для отображения деталей).
- **За что отвечает:** точка правды о базовой информации роли (имя, описание, системная ли).

#### `GetRolePermissions`

**Метод:** `GetRolePermissions(ctx context.Context, roleID string) ([]string, error)`

- **Что делает:**
  - по `roleID` находит роль и её имя;
  - читает политики Casbin для субъекта `role:<roleName>`;
  - для каждой политики в формате `p("role:<name>", workspaceId, obj, act)` собирает строку `obj + ":" + act` → `module:entity:action`.
- **Что возвращает:**
  - `[]string` — список прав роли в виде `PermissionString` (например, `crm:deal:create`).
- **Кто использует:**
  - `GET /api/v1/workspaces/:workspaceId/roles/:roleId/permissions`;
  - фронтовый сервис `roleService.getPermissions`, который подставляет эти права в `RoleFormModal.vue`.
- **За что отвечает:**
  - связывает **конфигурацию ролей (Casbin)** и **каталог прав**;
  - позволяет UI **сравнить дерево прав** (все возможные права) с тем, что реально назначено роли:
    - все галочки в `PermissionTree` ставятся по списку из `GetRolePermissions`;
    - отсутствующие права видны как неотмеченные чекбоксы.

#### `CreateRole`

**Метод:**  
`CreateRole(ctx context.Context, workspaceID, name, description string, permissions []string, createdBy string) (*model.WorkspaceRole, error)`

- **Вход:**
  - `workspaceID` — workspace, в котором создаётся роль;
  - `name` — имя роли (строка, уникальная внутри workspace);
  - `description` — произвольный текст;
  - `permissions` — список строк в формате `module:entity:action`;
  - `createdBy` — ID пользователя, создавшего роль (для аудита).
- **Что делает:**
  1. Валидирует имя (не пустое, уникально в workspace).
  2. Создаёт запись `WorkspaceRole` в БД с `isSystem=false`.
  3. Для каждого `permission`:
     - разбирает строку через `parsePermission` (получает `obj`, `act`);
     - добавляет политику `p("role:<name>", workspaceId, obj, act)` в Casbin.
  4. Вызывает `SavePolicy()` для сохранения политик в БД (`casbin_rule`).
  5. Если на любом шаге Casbin упал:
     - удаляет созданную роль из БД (компенсирующий откат).
- **Что возвращает:** созданную `WorkspaceRole` (с заполненным `id` и служебными полями).
- **Кто использует:**
  - `POST /api/v1/workspaces/:workspaceId/roles`;
  - модалка `RoleFormModal.vue` (создание новой роли).
- **За что отвечает:** точка создания **кастомной роли** и синхронизации её конфигурации с Casbin.

#### `UpdateRole`

**Метод:**  
`UpdateRole(ctx context.Context, roleID, name, description string, permissions []string) error`

- **Вход:**
  - `roleID` — обновляемая роль;
  - `name` — новое имя (опционально);
  - `description` — новое описание (опционально);
  - `permissions` — новый список прав роли (`module:entity:action`).
- **Что делает:**
  1. Загружает роль по `roleID`.
  2. Если `role.IsSystem == true` → возвращает `ErrRoleSystem` (запрет изменения системных ролей).
  3. Если имя меняется:
     - проверяет уникальность имени в рамках workspace;
     - обновляет имя и описание в БД.
  4. Удаляет **все старые политики** роли в Casbin для этого workspace:
     - `GetFilteredPolicy("role:<oldName>")`, затем `RemovePolicy(...)`.
  5. Добавляет политики для нового набора `permissions`:
     - `AddPolicy("role:<effectiveName>", workspaceId, obj, act)` для каждой строки.
  6. Вызывает `SavePolicy()`.
- **Что возвращает:** только ошибку или `nil`.
- **Кто использует:**
  - `PUT /api/v1/workspaces/:workspaceId/roles/:roleId`;
  - модалка `RoleFormModal.vue` (редактирование роли).
- **За что отвечает:** полное **переписывание состава прав роли** и, при необходимости, её имени.

#### `DeleteRole`

**Метод:** `DeleteRole(ctx context.Context, roleID string) error`

- **Что делает:**
  1. Загружает роль по `roleID`.
  2. Если роль системная → `ErrRoleSystem`.
  3. Считает количество назначений (`CountAssignmentsByRole`):
     - если `> 0` → `ErrRoleHasUsers` (нельзя удалить роль, пока она кому‑то назначена).
  4. Удаляет все Casbin‑политики вида `p("role:<name>", *, *, *)`.
  5. Сохраняет политики.
  6. Удаляет роль из БД.
- **За что отвечает:** безопасное удаление кастомной роли **без ломания** связанных пользователей и политик.

### Назначения ролей пользователям

#### `AssignRole`

**Метод:**  
`AssignRole(ctx context.Context, userID, roleID, workspaceID, assignedBy string) error`

- **Вход:** ID пользователя, ID роли, ID workspace, кто назначил.
- **Что делает:**
  1. Проверяет, что `role.WorkspaceID == workspaceID`.
  2. Создаёт запись `UserRoleAssignment`.
  3. Добавляет в Casbin группировку:
     - `g("user:<userID>", "role:<roleName>", workspaceId)`.
  4. Сохраняет политики.
  5. При любой ошибке Casbin:
     - удаляет запись назначения из БД;
     - при ошибке `SavePolicy()` откатывает grouping policy и назначение.
- **Кто использует:**
  - `POST /api/v1/workspaces/:workspaceId/users/:userId/roles/:roleId`;
  - UI управления участниками и их ролями.
- **За что отвечает:** связывание пользователя и роли **и в БД, и в Casbin**.

#### `RemoveRole`

**Метод:**  
`RemoveRole(ctx context.Context, userID, roleID, workspaceID string) error`

- **Что делает:**
  1. Удаляет запись из `user_role_assignments`.
  2. Удаляет группировку `g("user:<userID>", "role:<roleName>", workspaceId)` в Casbin.
  3. Сохраняет политики.
- **Кто использует:**
  - `DELETE /api/v1/workspaces/:workspaceId/users/:userId/roles/:roleId`.
- **За что отвечает:** корректное отвязывание пользователя от роли.

#### `GetUserRoles` и `GetUserRolesFull`

- `GetUserRoles(ctx, userID, workspaceID) ([]string, error)`:
  - возвращает **только имена** ролей пользователя в workspace;
  - используется для:
    - подсчёта эффективных прав;
    - формирования ответа `/me/permissions`.
- `GetUserRolesFull(ctx, userID, workspaceID) ([]WorkspaceRole, error)`:
  - возвращает полные объекты ролей (для UI страниц со списком ролей пользователя).

### Наследование ролей

#### `AddRoleInheritance` / `RemoveRoleInheritance`

См. раздел «Наследование ролей». Дополнительно:

- **Ответственность `AddRoleInheritance`:**
  - формирует **дерево зависимостей ролей** в рамках workspace:
    - в БД: `role_inheritance`;
    - в Casbin: `g("role:child", "role:parent", workspaceId)`.
- **Ответственность `RemoveRoleInheritance`:**
  - удаляет одну «ветку» в дереве:
    - запись в `role_inheritance`;
    - соответствующую `g`‑политику.

#### `SyncGroupingPoliciesFromAssignments`

**Метод:** `SyncGroupingPoliciesFromAssignments(ctx context.Context) error`

- **Что делает:**
  1. Читает все существующие `g`‑политики в Casbin и удаляет их.
  2. Перечитывает:
     - все `user_role_assignments` → `g("user:<id>", "role:<name>", workspaceId)`;
     - все `role_inheritance` по каждому workspace → `g("role:<child>", "role:<parent>", workspaceId)`.
  3. Сохраняет политики.
- **За что отвечает:**
  - восстанавливает **целостность дерева зависимостей** между пользователями и ролями/ролями и ролями;
  - полезен при рассинхронизации Casbin и БД.

### Индивидуальные права и эффективные права

#### `GrantPermission`, `RevokePermission`, `GetUserPermissions`

Подробно описаны в разделе «Индивидуальные права».

#### `GetEffectivePermissions`

**Метод:**  
`GetEffectivePermissions(ctx context.Context, userID, workspaceID string) ([]string, error)`

- **Что делает:**
  1. Получает все роли пользователя для workspace через `GetUserRoles`.
  2. Для каждой роли:
     - читает Casbin‑политики `p("role:<name>", workspaceId, obj, act)`;
     - собирает строки `obj + ":" + act` (`PermissionString`);
     - кладёт в множество `seen`.
  3. Загружает все индивидуальные права пользователя (`ListUserPermissions`):
     - добавляет `PermissionStr` каждого права в `seen`.
  4. Возвращает все ключи из `seen` как список.
- **Что возвращает:** полный набор строк `module:entity:action` без дублей.
- **Кто использует:**
  - HTTP‑эндпоинт `/api/v1/me/permissions`;
  - косвенно — фронтовый хук `usePermissions`, который через `authStore` опирается на этот ответ.
- **За что отвечает:** даёт **итоговое множество прав пользователя** с учётом:
  - дерева ролей (`user → role → наследование ролей`);
  - состава прав каждой роли;
  - индивидуальных прав.

### Системные политики

#### `EnsureSystemRolePolicies`

**Метод:** `EnsureSystemRolePolicies(ctx context.Context) error`

- **Что делает:**
  1. Получает все `workspaceId`, где есть роли/назначения (через `ListDistinctWorkspaceIDs`).
  2. Для каждого workspace:
     - вызывает `addSystemPoliciesForWorkspace(workspaceId)`, которая:
       - чистит существующие политики `role:OWNER`, `role:ADMIN`, `role:MEMBER`, `role:GUEST` в этом workspace;
       - заново добавляет наборы `ownerAdminPolicies`, `memberBasePolicies`, `guestBasePolicies`.
  3. Сохраняет политики.
- **За что отвечает:**
  - **поддерживает системные роли в корректном состоянии** во всех existing workspaces;
  - используется при старте приложения и/или миграциях.

#### `SeedSystemPoliciesForWorkspace`

**Метод:** `SeedSystemPoliciesForWorkspace(workspaceID string)`

- **Что делает:**
  - вызывает `addSystemPoliciesForWorkspace(workspaceID)` только для одного workspace.
- **Когда используется:**
  - при создании нового workspace;
  - при выборочном восстановлении/инициализации прав в одном workspace.
- **Как «настраивать» системные роли:**
  - **через изменение массивов** `ownerAdminPolicies`, `memberBasePolicies`, `guestBasePolicies` в коде;
  - после изменения наборов:
    - запустить `EnsureSystemRolePolicies` (глобально) или `SeedSystemPoliciesForWorkspace` (точечно);
  - **через API системные роли не меняются**, только перечитываются наборы из кода.

---

## HTTP‑API и связь с фронтендом

Ключевые эндпоинты (см. также `DOC_ALL_METHODS_BACK.md` и `PERMISSIONS_FRONTEND.v1.md`):

- **Каталог прав**:
  - `GET /api/v1/workspaces/:workspaceId/permissions/catalog`
  - возвращает каталог, сгруппированный по модулям/сущностям, для `PermissionTree` и UI‑админки.
- **Роли**:
  - `GET /api/v1/workspaces/:workspaceId/roles` — список ролей;
  - `POST /api/v1/workspaces/:workspaceId/roles` — создать роль;
  - `GET /api/v1/workspaces/:workspaceId/roles/:roleId` — получить роль;
  - `PUT /api/v1/workspaces/:workspaceId/roles/:roleId` — обновить роль;
  - `DELETE /api/v1/workspaces/:workspaceId/roles/:roleId` — удалить роль;
  - `GET /api/v1/workspaces/:workspaceId/roles/:roleId/permissions` — получить права роли (`PermissionString[]`);
  - `POST /api/v1/workspaces/:workspaceId/roles/:roleId/inherit/:parentRoleId` — добавить наследование;
  - `DELETE /api/v1/workspaces/:workspaceId/roles/:roleId/inherit/:parentRoleId` — удалить наследование.
- **Назначения ролей и индивидуальные права**:
  - `GET /api/v1/workspaces/:workspaceId/users/:userId/roles` — роли пользователя;
  - `POST /api/v1/workspaces/:workspaceId/users/:userId/roles/:roleId` — назначить роль;
  - `DELETE /api/v1/workspaces/:workspaceId/users/:userId/roles/:roleId` — снять роль;
  - `GET /api/v1/workspaces/:workspaceId/users/:userId/permissions` — индивидуальные права;
  - `POST /api/v1/workspaces/:workspaceId/users/:userId/permissions` — выдать право;
  - `DELETE /api/v1/workspaces/:workspaceId/users/:userId/permissions/:permissionId` — отозвать право.
- **Права текущего пользователя**:
  - `GET /api/v1/me/permissions?workspaceId=...`
  - возвращает:
    - `permissions: string[]` — полный список `module:entity:action`;
    - `roles: string[]` — имена ролей;
    - `systemRole: 'OWNER' | 'ADMIN' | 'MEMBER' | 'GUEST'`.

Фронтенд использует этот ответ как **единственный источник истины** о правах текущего пользователя.

---

## Как сравнивать дерево прав с ролями

Для анализа и настройки ролей важно уметь сравнивать:

- **дерево прав** (каталог) — все возможные права в системе;
- **состав ролей** — какие из этих прав реально назначены каждой роли и пользователю.

### 1. Получить дерево прав (каталог)

- В бэкенде:
  - `PermissionService.GetCatalog` → `[]PermissionCatalogItem`.
- В API:
  - `GET /api/v1/workspaces/:workspaceId/permissions/catalog`.
- На фронтенде:
  - хук `usePermissionTree` превращает плоский список в структуру:
    - `modules[moduleCode].entities[entityType].actions[action]`.

Это и есть **«дерево зависимостей» прав**, которым оперируют UI‑компоненты.

### 2. Получить права роли

- В бэкенде:
  - `PermissionService.GetRolePermissions(roleID)` → `[]string` (`PermissionString`).
- В API:
  - `GET /api/v1/workspaces/:workspaceId/roles/:roleId/permissions`.
- На фронтенде:
  - `roleService.getPermissions(workspaceId, roleId)`;
  - результат передаётся в `RoleFormModal` как `initialPermissions`.

### 3. Сопоставить роль и каталог

На фронте это делает `PermissionTree`:

- он получает:
  - полное дерево всех возможных прав (из каталога);
  - текущий массив `modelValue: PermissionString[]` — права роли.
- логика:
  - для каждого `action` в дереве:
    - если его `permissionString` содержится в `modelValue` → чекбокс отмечен;
    - если нет → чекбокс пустой.

При сохранении:

- `PermissionTree` эмитит обновлённый массив `PermissionString[]`;
- `RoleFormModal` передаёт этот массив в `useRoleEditor`;
- `useRoleEditor` вызывает:
  - `CreateRole` или `UpdateRole` с новым списком прав;
  - бэкенд пересобирает политики Casbin через `CreateRole`/`UpdateRole`.

### 4. Сравнение прав пользователя с деревом

Для текущего пользователя:

- бэкенд:
  - `GetEffectivePermissions(userID, workspaceID)` → `[]string`;
- фронт:
  - `usePermissions` использует `effectivePermissions.permissions`;
  - UI‑компоненты через `can/canAny/canAll` сравнивают:
    - `PermissionString` из дерева (`Perm.xxx`) с `permissions` пользователя.

Таким образом:

- **дерево** отвечает на вопрос: «Какие права вообще существуют и как они сгруппированы по модулям/сущностям?»;
- **методы ролей и эффективных прав** отвечают: «Какие из этих прав назначены конкретной роли/конкретному пользователю?»;
- сравнение делается либо:
  - на фронте (через `PermissionTree`, `usePermissions`, `PermissionGuard`);
  - либо на бэке (через `GetRolePermissions`, `GetEffectivePermissions`) для диагностики и отчётов.

---

## Работа с правами на фронтенде

### Хук `usePermissions`

Файл: `frontend/src/features/permissions/model/use-permissions.ts`.

- Берёт данные из `authStore.effectivePermissions` (который наполняется ответом `/me/permissions`).
- Возвращает:
  - `permissions: PermissionString[]` — список прав;
  - `roles: string[]` — имена ролей;
  - `systemRole` — системная роль в workspace;
  - методы:
    - `can(permission: PermissionString): boolean`;
    - `canAny(permissions: PermissionString[]): boolean`;
    - `canAll(permissions: PermissionString[]): boolean`.

**Рекомендация:**  
Во всех компонентах использовать именно эти методы, а не напрямую лезть в стор.

### Компонент `PermissionGuard`

Файл: `frontend/src/features/permissions/ui/PermissionGuard.vue`.

API:

- `permission: PermissionString | PermissionString[]` — одно или несколько прав;
- `requireAll?: boolean` — если `true`, то требуется **выполнение всех** прав из массива, иначе достаточно любого (логика `canAny`/`canAll`);
- `fallback?: Component` — компонент, который нужно отрендерить при отсутствии прав (например, заглушка или `AccessDenied`).

Логика:

- Если `permission` — массив и `requireAll=true` → используется `canAll`;
- Если `permission` — массив и `requireAll=false|undefined` → `canAny`;
- Если `permission` — строка → `can`.

Использование:

```vue
<PermissionGuard :permission="Perm.crm.deal.create">
  <Button @click="onCreateDeal">Создать сделку</Button>
</PermissionGuard>

<PermissionGuard
  :permission="[Perm.crm.deal.update, Perm.crm.deal.move]"
  :require-all="false"
  :fallback="ReadOnlyBanner"
>
  <DealEditForm />
</PermissionGuard>
```

### Дерево прав `PermissionTree`

Файл: `frontend/src/features/permissions/ui/PermissionTree.vue`.

- Основано на хуке `usePermissionTree`, который:
  - загружает каталог прав workspace;
  - строит иерархию `modules → entities → actions`.
- Принимает `modelValue: PermissionString[]` и эмитит `update:modelValue` с новым набором прав.
- UI:
  - карточки модулей;
  - внутри — сущности;
  - внутри сущностей — чекбоксы для каждого действия с человекочитаемыми названиями.

Используется в модалке создания/редактирования ролей (`RoleFormModal.vue`) и, потенциально, в других местах админки.

### Админка ролей

Основные компоненты:

- `RolesList.vue`:
  - отображает:
    - заголовок «Роли workspace»;
    - списки системных и кастомных ролей;
  - использует:
    - `useRolesPage` для загрузки ролей;
    - `RoleCard` для отображения;
    - `RoleFormModal` для создания/редактирования.
- `RoleFormModal.vue`:
  - обёртка над формой создания/редактирования роли:
    - поля «Название», «Описание»;
    - компонент `PermissionTree` для выбора прав.
  - использует хук `useRoleEditor` для бизнес‑логики сохранения:
    - создаёт или обновляет роль через `roleService`;
    - подставляет `initialPermissions` при открытии модалки.

### Индивидуальные права: `UserPermissionsPanel`

Файл: `frontend/src/features/user-permissions/ui/UserPermissionsPanel.vue`.

- Принимает `userId`.
- С помощью `useUserPermissions(userId)`:
  - загружает индивидуальные права пользователя и доступные права из каталога;
  - предоставляет методы `grantPermission`, `revokePermission`.
- Рендерит:
  - селект с доступными правами (`availablePermissions`);
  - список уже выданных прав (`userPermissions`) c возможной датой истечения (`expiresAt`);
  - кнопки «Выдать право» и «Отозвать».

---

## Типовые сценарии настройки доступа

### 1. Дать пользователю полный доступ к CRM, но запретить управление workspace

1. Создать кастомную роль «CRM Manager»:
   - в `RolesList` нажать «Создать роль»;
   - в `PermissionTree`:
     - отметить все права `crm:*:*`;
     - **не отмечать** права `workspace:*:*`.
2. Назначить эту роль нужным пользователям через страницу управления участниками/ролями.
3. Проверить, что:
   - пользователь может делать всё в CRM;
   - не видит/не может использовать действия управления участниками и модулями workspace.

### 2. Создать read‑only роль для CRM

1. Создать кастомную роль «CRM Read Only»:
   - отметить только права `crm:*:read` (сделки, контакты, компании, активности);
   - не ставить `create/update/delete/move`.
2. Назначить эту роль гостям, которые должны только смотреть данные.
3. UI с использованием `PermissionGuard` и `usePermissions` автоматически скроет:
   - кнопки создания/редактирования/удаления;
   - действия перетаскивания на канбан‑доске сделок.

### 3. Выдать временный доступ к экспорту сделок одному пользователю

1. Открыть карточку пользователя в настройках.
2. В блоке «Индивидуальные права» (`UserPermissionsPanel`):
   - выбрать право `crm:export:deals` из селекта;
   - при наличии поддержки `expiresAt` (на уровне API) — указать дату окончания;
   - нажать «Выдать право».
3. Пользователь сможет экспортировать сделки, даже если его роли не содержат это право.
4. После истечения срока (или отзыва вручную) — право исчезнет.

### 4. Построить иерархию ролей через наследование

1. Допустим, есть роль `Manager` и нужно создать `Senior Manager`:
   - создать кастомную роль `Senior Manager` с дополнительными правами;
   - через API:
     - вызвать `POST /roles/{seniorId}/inherit/{managerId}`;
   - или через UI (когда появится обёртка).
2. В Casbin появится группировка:
   - `g("role:Senior Manager", "role:Manager", workspaceId)`.
3. Все пользователи с ролью `Senior Manager` унаследуют права `Manager` + свои.

---

## Нюансы и подводные камни

- **Системные роли неизменяемы**:
  - нельзя удалить или изменить `OWNER`, `ADMIN`, `MEMBER`, `GUEST`;
  - их базовые права задаются в коде и поддерживаются методами `EnsureSystemRolePolicies`/`SeedSystemPoliciesForWorkspace`.
- **Индивидуальные права не в Casbin**:
  - сейчас они учитываются только в `GetEffectivePermissions` и, соответственно, во фронтовой логике;
  - при включении строгого PermissionMiddleware нужно либо:
    - дублировать индивидуальные права в Casbin;
    - либо вызывать дополнительную проверку в middleware против `user_permissions`.
- **Кэширование прав**:
  - пока не реализовано на бэкенде (каждый вызов `GetEffectivePermissions` ходит в БД и Casbin);
  - на фронте права кэшируются в `authStore` и/или локальном хранилище;
  - при высокой нагрузке рекомендуется добавить кэш с TTL и инвалидацией при изменении ролей/прав.
- **Режим логирования PermissionMiddleware**:
  - сейчас защита на уровне HTTP ПОКА не включена;
  - UI уже может скрывать недоступные действия на основе `/me/permissions`;
  - включение боевого режима должно сопровождаться регрессионным тестированием.
- **Расширение каталога прав**:
  - при добавлении новых модулей/сущностей **обязательно**:
    - добавить записи в `permission_catalog`;
    - обновить спецификации и (при необходимости) генерацию типов на фронте;
    - настроить маппинг эндпоинтов на права в `endpointPermissionTable`;
    - при необходимости расширить массивы базовых политик системных ролей.
- **Согласованность строк прав**:
  - строка `module:entity:action` должна быть **строго одинаковой**:
    - в `permission_catalog`;
    - в коде Casbin;
    - в типах фронтенда (`PermissionString`);
    - в утилитах `Perm`/`usePermissions`.

---

## Как использовать этот документ

- **Бэкенд‑разработчики**:
  - ориентируются на разделы о модели данных, Casbin и `PermissionService` при доработке API и middleware;
  - используют типовые сценарии как чек‑листы при настройке прав.
- **Фронтенд‑разработчики**:
  - используют разделы про `usePermissions`, `PermissionGuard`, `PermissionTree` и UI‑компоненты для правильной интеграции прав в новую функциональность;
  - избегают «магических строк» и полагаются на `PermissionString`/`Perm`.
- **Администраторы workspace / продукт**:
  - читают разделы о системных и кастомных ролях, индивидуальных правах и сценариях;
  - формулируют правила доступа в терминах `module:entity:action`, которые затем реализуются через UI и/или API.

Документ должен оставаться **актуальным источником истины** о системе прав и ролей. При изменении модели данных, добавлении модулей или прав — обновлять этот README вместе с соответствующими спецификациями.

