# Итоговый отчёт по внедрению гибкой системы ролей и прав доступа (TOTAL)

**Дата:** Март 2026  
**Файл:** `PERMISSION_REPORT.PART_TOTAL.md`  
**Основания:** `perrmissions.v2.md`, `PERMISSIONS_FRONTEND.v1.md/v2.md`, `PERMISSION_REPORT.PART_1/2/3.md`, `SPEC_ROLE_BACK.md`, `DOC_ALL_METHODS_BACK.md`

---

## 1. Цели и контекст

- Перевести ERP на **гибкую, гранулярную модель прав**:
  - права вида `module:entity:action` (например, `crm:deal:create`);
  - кастомные роли на уровне workspace;
  - наследование ролей;
  - индивидуальные права.
- Сохранить:
  - обратную совместимость (пошаговое включение, без «большого взрыва»);
  - производительность (проверка прав < 5 мс);
  - прозрачную архитектуру и понятный API для фронтенда.
- Использовать **Casbin** как движок авторизации с доменами (workspace_id) и иерархией ролей.

Итог: на момент этого отчёта **вся ролево‑правовая часть на бэкенде реализована**, кроме включения «боевого» режима PermissionMiddleware и кэширования прав (это зафиксировано как следующий этап).

---

## 2. Этап 1 — Схема БД и миграции

### 2.1. Новые таблицы (000022_permissions_schema_and_seed)

Созданы таблицы (см. `perrmissions.v2.md` и PART_1):

1. `permission_catalog` — каталог всех возможных прав:
   - поля: `id`, `module_code`, `entity_type`, `action`, `name`, `description`, `is_system`, `created_at`;
   - уникальность `(module_code, entity_type, action)`;
   - индексы по `module_code`, `entity_type`.

2. `workspace_roles` — роли в workspace:
   - поля: `id`, `workspace_id`, `name`, `description`, `is_system`, `created_at`, `updated_at`;
   - уникальность `(workspace_id, name)`;
   - `is_system = true` для OWNER/ADMIN/MEMBER/GUEST.

3. `user_role_assignments` — назначения ролей:
   - поля: `id`, `user_id`, `role_id`, `workspace_id`, `assigned_by`, `assigned_at`;
   - уникальность `(user_id, role_id, workspace_id)`;
   - индексы для поиска по `user_id`, `role_id`, `workspace_id` и `(user_id, workspace_id)`.

4. `user_permissions` — индивидуальные права:
   - поля: `id`, `user_id`, `workspace_id`, `permission_id`, `granted_by`, `granted_at`, `expires_at`;
   - уникальность `(user_id, workspace_id, permission_id)`;
   - индексы по `user_id`, `workspace_id`, `(user_id, workspace_id)`.

5. `role_inheritance` — наследование ролей:
   - поля: `id`, `workspace_id`, `child_role_id`, `parent_role_id`, `created_at`;
   - CHECK `child_role_id != parent_role_id`;
   - уникальность `(workspace_id, child_role_id, parent_role_id)`;
   - индексы по `child_role_id`, `parent_role_id`.

Все операции в миграции — **идемпотентные** (`IF NOT EXISTS`, `ON CONFLICT DO NOTHING`).

### 2.2. Первичное наполнение `permission_catalog`

Заложен базовый набор системных прав:

- **CRM:** `crm:deal:{create,read,update,delete,move}`, `crm:contact:{create,read,update,delete}`, `crm:company:{create,read,update,delete}`, `crm:pipeline:manage`, `crm:activity:{create,read,update,delete}`, `crm:export:deals`.
- **Habits:** `habits:habit:{create,read,update,delete,complete}`, `habits:journal:{create,read,update,delete}`.
- **Projects:** `projects:project:{create,read,update,delete}`, `projects:entity:{attach,detach}`.
- **Workspace:** `workspace:member:{invite,remove}`, `workspace:role:manage`, `workspace:module:manage`.

### 2.3. Миграция существующих ролей (000023_system_workspace_roles_and_assignments)

1. **Системные роли по всем workspace:**
   - Для каждого `workspaces.id` создаются роли OWNER/ADMIN/MEMBER/GUEST в `workspace_roles` с `is_system = true`.

2. **Перенос существующих назначений:**
   - Все записи из `user_workspaces` конвертированы в `user_role_assignments`,
   - используется соответствие `uw.role` ↔ системная роль в `workspace_roles`.

3. **Триггер на создание новых workspaces:**
   - `fn_create_system_roles` + `tr_create_system_roles` автоматически создают OWNER/ADMIN/MEMBER/GUEST при вставке в `workspaces`.

---

## 3. Этап 2 — GORM, Casbin и middleware‑цепочка

### 3.1. GORM и Enforcer

- Введён `InitGormDB(cfg)` для получения `*gorm.DB` на том же DSN, что и `*sql.DB`.
- В `internal/authz/casbin.go` реализован `InitEnforcer(gormDB)`:
  - GORM‑адаптер `gormadapter.NewAdapterByDB(gormDB)` использует таблицу `casbin_rule`;
  - Модель Casbin с доменами и иерархией ролей (см. `SPEC_ROLE_BACK.md` п.2.2);
  - `LoadPolicy()` загружает политики при старте.

### 3.2. Встраивание в DI и App

- В `internal/app/app.go`:
  - после `InitDB` и `RunMigrations` вызывается `InitGormDB`;
  - DI‑контейнер создаётся через `di.NewContainer(db, gormDB, cfg)`.
- В `internal/di/container.go`:
  - создаётся `Enforcer *casbin.Enforcer` через `authz.InitEnforcer(gormDB)`;
  - `Enforcer` прокидывается в `PermissionService` и в `PermissionMiddleware`.

### 3.3. Новая цепочка middleware

Для защищённых маршрутов `/api/v1` используется:

1. `GinAuthMiddleware` — аутентификация, установка `user_id` и глобальной роли.
2. `WorkspacePathMiddleware` — проверка доступа к конкретному `:workspaceId`, установка `workspace_id` в контекст.
3. `ModuleLicenseMiddleware` — проверка включения модуля и лицензий.
4. `PermissionMiddleware` — маппинг endpoint → `(obj, act)`, вызов Casbin в **режиме логирования**.

Важно: на этом этапе **Casbin не блокирует** запросы, а только логирует решения (для безопасного поэтапного включения).

---

## 4. Этап 3 — PermissionRepository, PermissionService и API

### 4.1. Repository и модели

Реализован `internal/repository/permission.Repository` и `internal/model/permission.go`:

- Модели: `PermissionCatalogItem`, `WorkspaceRole`, `UserRoleAssignment`, `UserPermission`, `RoleInheritance`.
- Методы репозитория:
  - Каталог: `ListCatalog`, `GetCatalogByID`.
  - Роли: `ListRolesByWorkspace`, `GetRoleByID`, `GetRoleByName`, `CreateRole`, `UpdateRole`, `DeleteRole`, `CountAssignmentsByRole`.
  - Назначения: `ListUserRoleAssignments`, `CreateUserRoleAssignment`, `DeleteUserRoleAssignment`, `ListAllUserRoleAssignments`.
  - Индивидуальные права: `ListUserPermissions`, `CreateUserPermission`, `DeleteUserPermission`.
  - Наследование: `ListRoleInheritanceByWorkspace`, `CreateRoleInheritance`, `DeleteRoleInheritance`.

### 4.2. PermissionService

Сервис `internal/service/permission.Service` реализует:

- **Каталог и роли:**
  - `GetCatalog`, `ListRoles`, `GetRole`, `GetRolePermissions`.
  - `CreateRole`:
    - создаёт запись в `workspace_roles`,
    - для каждого права добавляет `p("role:<Name>", workspaceId, obj, act)` в Casbin,
    - при ошибке Casbin выполняет откат — удаление роли из БД.
  - `UpdateRole`:
    - запрещает менять системные роли;
    - при смене имени проверяет уникальность и обновляет БД;
    - удаляет все старые `p` для `role:<oldName>` в домене workspace, добавляет новые для `role:<newName>`.
  - `DeleteRole`:
    - запрещает удаление системных ролей;
    - проверяет отсутствие назначений;
    - удаляет все `p` для роли, затем запись из `workspace_roles`.

- **Назначения ролей (user→role):**
  - `AssignRole`:
    - создаёт запись в `user_role_assignments`,
    - добавляет `g("user:<userID>", "role:<roleName>", workspaceId)` в Casbin,
    - при ошибках Casbin/SavePolicy откатывает назначение и grouping policy.
  - `RemoveRole`:
    - удаляет запись в БД,
    - удаляет grouping policy и сохраняет политики.

- **Наследование ролей (role→role):**
  - `AddRoleInheritance`:
    - проверяет существование child/parent внутри workspace и разность ID;
    - создаёт запись в `role_inheritance`;
    - добавляет `g("role:<child>", "role:<parent>", workspaceId)` и сохраняет политики;
    - при сбое Casbin удаляет запись `role_inheritance`.
  - `RemoveRoleInheritance`:
    - удаляет запись `role_inheritance`;
    - удаляет соответствующую grouping policy и сохраняет политики.

- **Индивидуальные права:**
  - `GrantPermission`:
    - валидирует `permissionId` по каталогу;
    - создаёт запись в `user_permissions` (с `grantedBy`, `expiresAt`).
  - `RevokePermission` — удаляет запись.
  - `GetUserPermissions` — возвращает все активные индивидуальные права пользователя в workspace с денормализованной строкой `permission`.

- **Эффективные права:**
  - `GetEffectivePermissions(userID, workspaceID)`:
    - по ролям: читает `GetUserRoles`, для каждой роли — `GetFilteredPolicy(0, "role:"+name)`, фильтрует по домену, собирает `obj+":"+act`;
    - по индивидуальным: добавляет `PermissionStr` из `user_permissions`;
    - возвращает множество всех строк `module:entity:action`.

- **Стартовая инициализация Casbin:**
  - `EnsureSystemRolePolicies(ctx)`:
    - по `ListDistinctWorkspaceIDs` перезаписывает все `p` для системных ролей OWNER/ADMIN/MEMBER/GUEST в каждом workspace на основе массивов `ownerAdminPolicies`, `memberBasePolicies`, `guestBasePolicies`.
  - `SyncGroupingPoliciesFromAssignments(ctx)`:
    - очищает все `g` (grouping policies) в Casbin;
    - для всех `user_role_assignments` создаёт `g(user→role)`;
    - для всех `role_inheritance` во всех workspace создаёт `g(role→role)`;
    - сохраняет политики.

### 4.3. API и хендлеры

В `internal/handler/permission/handler.go` реализованы:

- `RegisterRoutes(wsGroup)` для `/api/v1/workspaces/:workspaceId/...`:
  - `GET /permissions/catalog` → `GetCatalog`;
  - `GET /roles` → `ListRoles`;
  - `POST /roles` → `CreateRole`;
  - `GET /roles/:roleId` → `GetRole`;
  - `PUT /roles/:roleId` → `UpdateRole`;
  - `DELETE /roles/:roleId` → `DeleteRole`;
  - `GET /roles/:roleId/permissions` → `GetRolePermissions`;
  - `POST /roles/:roleId/inherit/:parentRoleId` → `AddRoleInheritance`;
  - `DELETE /roles/:roleId/inherit/:parentRoleId` → `RemoveRoleInheritance`;
  - `GET /users/:userId/roles` → `GetUserRoles`;
  - `POST /users/:userId/roles/:roleId` → `AssignRole`;
  - `DELETE /users/:userId/roles/:roleId` → `RemoveRole`;
  - `GET /users/:userId/permissions` → `GetUserPermissions`;
  - `POST /users/:userId/permissions` → `GrantPermission`;
  - `DELETE /users/:userId/permissions/:permissionId` → `RevokePermission`.

- `GET /api/v1/me/permissions?workspaceId=...`:
  - использует `GetEffectivePermissions` и `GetUserRoles`;
  - возвращает:
    - `permissions: string[]` — все права пользователя в workspace;
    - `roles: string[]` — имена ролей пользователя в workspace;
    - `systemRole` — системную роль (`OWNER`/`ADMIN`/`MEMBER`/`GUEST`).

Все маршруты подключены в `di.Container.RegisterRoutes`:

- `meGroup.GET("/permissions", PermissionHandler.GetMyPermissions)` на `protected`;
- `wsIDGroup := workspaceGroup.Group("/:workspaceId")`, и на `wsIDGroup` вешаются все маршруты PermissionHandler.

---

## 5. Итоговая архитектура ролей и прав

В результате всех этапов сложилась следующая картина (см. также `SPEC_ROLE_BACK.md` и `PERMISSIONS_FRONTEND.v1.md`):

1. **Данные:**
   - Полный словарь прав (`permission_catalog`) с понятными кодами для фронта.
   - Роли workspace (`workspace_roles`) — системные и кастомные.
   - Назначения ролей (`user_role_assignments`) и индивидуальные права (`user_permissions`).
   - Наследование ролей (`role_inheritance`).

2. **Casbin как единый источник истинных политик:**
   - Ролевые политики `p(role, workspace, obj, act)`.
   - Группировки user→role и role→role (наследование).
   - Модель с доменом = `workspace_id`.

3. **Middleware‑цепочка:**
   - Auth → Workspace → Module → Permissions (log‑only).
   - Хендлеры очищены от проверки членства и модулей; проверяют только бизнес‑валидацию.

4. **PermissionService:**
   - Централизует все изменения прав/ролей и синхронизирует их в Casbin.
   - Даёт единый API для получения эффективных прав пользователя.

5. **Контракт для фронтенда:**
   - `/permissions/catalog` — дерево всех прав для построения UI.
   - `/roles` и связанные маршруты — управление ролями.
   - `/users/:userId/roles` и `/users/:userId/permissions` — назначение/индивидуальные права.
   - `/me/permissions` — основной эндпоинт, на который опирается фронтенд‑store прав.

---

## 6. Оставшиеся шаги (следующий инкремент)

На момент этого TOTAL‑отчёта остаются следующие задачи (сознательно вынесенные в следующий этап, чтобы не блокировать фронт):

1. **Включить боевой режим PermissionMiddleware:**
   - при `Enforce == false` и отсутствии индивидуального права возвращать 403, а не только логировать;
   - параллельно сравнить логи и реальное поведение, чтобы избежать регрессий.

2. **Учесть индивидуальные права в PermissionMiddleware:**
   - либо через чтение `user_permissions` при отказе по ролям;
   - либо через синхронизацию индивидуальных прав в Casbin как отдельные `p("user:<id>", workspace, obj, act)`.

3. **Расширить `mapEndpointToPermission`:**
   - покрыть все эндпоинты CRM/Habits/Projects, включая экспорт, пайплайны и т.п., в соответствии с каталогом прав.

4. **Кэширование эффективных прав:**
   - TTL‑кэш для результатов `GetEffectivePermissions(userID, workspaceID)` на бэкенде;
   - инвалидация при изменении ролей/назначений/индивидуальных прав;
   - фронтовое кэширование уже предусмотрено в `PERMISSIONS_FRONTEND.v1.md`.

---

## 7. Вывод

- С точки зрения **бэкенда**:
  - модель данных, Casbin, PermissionService и API для ролей/прав полностью реализованы и согласованы со спецификациями;
  - все ключевые механизмы (системные/кастомные роли, наследование, индивидуальные права, эффективные права и `/me/permissions`) работают и покрыты документацией.

- С точки зрения **фронтенда**:
  - можно опираться на текущий контракт:
    - использовать `/me/permissions` и `/permissions/catalog` для динамического UI;
    - строить админку ролей и индивидуальных прав поверх реализованных эндпоинтов;
  - включение «боевого» PermissionMiddleware и кэширования прав будет прозрачным для фронта, так как не меняет контракт API.

**Статус работ по системе ролей и прав:**  
ядро реализовано и готово к использованию в проде; оставшиеся задачи — оптимизации и включение полного контроля доступа на уровне middleware.

