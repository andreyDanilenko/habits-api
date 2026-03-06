# Отчёт по внедрению гибкой системы ролей и прав доступа (Часть 3)

**Дата:** Март 2026  
**Файл:** `PERMISSION_REPORT.PART_3.md`

---

## 1. Спецификация: как работают роли

### 1.1. Термины

| Термин | Определение |
|--------|-------------|
| **Workspace** | Рабочее пространство. Все роли и права изолированы по `workspace_id`. |
| **Право (Permission)** | Разрешение вида `{модуль}:{сущность}:{действие}`, например `crm:deal:create`. Хранится в `permission_catalog` и используется в политиках Casbin как пара (объект = `module:entity`, действие = `action`). |
| **Роль (Role)** | Именованный набор прав в рамках одного workspace. Хранится в `workspace_roles`; набор прав роли хранится в Casbin как политики `p(role:name, workspace_id, obj, act)`. |
| **Системная роль** | OWNER, ADMIN, MEMBER, GUEST. Создаются автоматически при создании workspace (`is_system = true`). Нельзя удалить и нельзя менять имя через API. |
| **Кастомная роль** | Роль, созданная администратором workspace. Можно создавать, редактировать (имя, описание, набор прав) и удалять, если нет назначений. |
| **Назначение роли** | Связь «пользователь — роль — workspace» в таблице `user_role_assignments`. В Casbin отражается групповой политикой `g(user:id, role:name, workspace_id)`. |
| **Индивидуальное право** | Право, выданное конкретному пользователю напрямую в таблице `user_permissions` (минуя роли). Учитывается при расчёте эффективных прав и при проверке в Casbin (через отдельную логику в сервисе). |
| **Эффективные права** | Объединение всех прав пользователя в workspace: из всех его ролей (через Casbin) плюс индивидуальные права из `user_permissions`. |

### 1.2. Жизненный цикл роли

1. **Создание workspace**  
   Триггер в БД создаёт в `workspace_roles` четыре системные роли (OWNER, ADMIN, MEMBER, GUEST) с `is_system = true`.

2. **Загрузка при старте приложения**  
   - `EnsureSystemRolePolicies`: для каждого workspace в Casbin добавляются политики `p` для OWNER/ADMIN/MEMBER/GUEST (фиксированный набор прав по спецификации).  
   - `SyncGroupingPoliciesFromAssignments`: все записи из `user_role_assignments` переносятся в Casbin как `g(user:id, role:name, workspace_id)`.  
   Оба шага идемпотентны (повторный запуск не дублирует данные).

3. **Создание кастомной роли**  
   - В БД создаётся запись в `workspace_roles` (имя, описание, `is_system = false`).  
   - Для каждого права из списка `permissions` (формат `module:entity:action`) в Casbin добавляется политика `p(role:name, workspace_id, obj, act)`.

4. **Редактирование кастомной роли**  
   - В БД обновляются имя и описание.  
   - В Casbin удаляются все политики этой роли в данном workspace и заново добавляются согласно новому списку `permissions`.

5. **Удаление кастомной роли**  
   - Разрешено только если нет записей в `user_role_assignments` для этой роли.  
   - В Casbin удаляются все политики `p` для этой роли.  
   - В БД удаляется запись из `workspace_roles`.

6. **Назначение роли пользователю**  
   - В БД создаётся запись в `user_role_assignments`.  
   - В Casbin добавляется группировка `g(user:id, role:name, workspace_id)`.

7. **Снятие роли с пользователя**  
   - Запись из `user_role_assignments` удаляется.  
   - Соответствующая группировка удаляется из Casbin.

### 1.3. Связь с Casbin

- **Модель:** запрос `(sub, dom, obj, act)`; политики `p(sub, dom, obj, act)`; группировки `g(user, role, dom)`. Домен `dom` = `workspace_id`.
- **Субъект при проверке:** передаётся `sub = "user:" + userID`. Casbin по группировкам находит все роли пользователя в данном домене и проверяет, есть ли у какой-либо роли политика с данным `(dom, obj, act)`.
- **Политики ролей:** хранятся как `p("role:OWNER", workspaceID, "crm:deal", "create")` и т.п. Для системных ролей задаётся фиксированный набор при старте; для кастомных — при создании/обновлении роли через PermissionService.
- **Индивидуальные права:** в текущей реализации при расчёте эффективных прав объединяются права из ролей (через Casbin) и права из `user_permissions`. При включённом боевом режиме PermissionMiddleware проверка индивидуальных прав выполняется отдельно (по списку из БД), так как в Casbin они не хранятся в виде отдельных политик на пользователя.

### 1.4. Системные роли: базовые наборы прав

| Роль   | Описание набора |
|--------|------------------|
| OWNER  | Все права по каталогу (CRM, Habits, Projects, workspace:member/role/module). |
| ADMIN  | То же, что OWNER. |
| MEMBER | Создание, чтение, обновление по основным сущностям; без удаления по сделкам/контактам/компаниям; без `workspace:role:manage` и `workspace:member:remove`; с `workspace:member:invite` и `workspace:module:manage`. |
| GUEST  | Только чтение по сделкам, контактам, компаниям, привычкам, журналу, проектам. |

### 1.5. Эффективные права пользователя

- **Источники:** (1) все роли пользователя в workspace → политики Casbin для этих ролей в данном workspace; (2) записи в `user_permissions` для этого пользователя и workspace (с учётом `expires_at`).
- **Формат ответа API:** массив строк `"module:entity:action"` и массив имён ролей. Используется фронтом для скрытия/показа элементов UI и для вызова `can(permission)`.

---

## 2. Текущий этап (после Части 2): что реализовано

### 2.1. Репозиторий и модели

- **Пакет** `internal/repository/permission`: работа с таблицами `permission_catalog`, `workspace_roles`, `user_role_assignments`, `user_permissions`.
- **Модели** в `internal/model/permission.go`: `PermissionCatalogItem`, `WorkspaceRole`, `UserRoleAssignment`, `UserPermission`.
- Реализованы: листинг каталога; CRUD ролей; создание/удаление назначений и индивидуальных прав; получение назначений по пользователю и workspace; `ListDistinctWorkspaceIDs`, `ListAllUserRoleAssignments` для синхронизации с Casbin; `CountAssignmentsByRole` для проверки перед удалением роли.

### 2.2. PermissionService

- **Пакет** `internal/service/permission`: все операции с ролями и правами с синхронизацией в Casbin.
- **Каталог и роли:** `GetCatalog`, `ListRoles`, `GetRole`, `GetRolePermissions`, `CreateRole`, `UpdateRole`, `DeleteRole`. При создании/обновлении роли политики `p` в Casbin обновляются; системные роли нельзя редактировать или удалять.
- **Назначения:** `AssignRole`, `RemoveRole`, `GetUserRoles`, `GetUserRolesFull`. При назначении/снятии роли добавляется/удаляется группировка `g` в Casbin.
- **Индивидуальные права:** `GrantPermission`, `RevokePermission`, `GetUserPermissions`.
- **Эффективные права:** `GetEffectivePermissions(userID, workspaceID)` — объединение прав из ролей (через Casbin) и из `user_permissions`.
- **Стартовая загрузка:** `EnsureSystemRolePolicies(ctx)` — для каждого workspace из `workspace_roles` перезаписываются политики системных ролей (идемпотентно). `SyncGroupingPoliciesFromAssignments(ctx)` — очистка всех `g` в Casbin и загрузка из `user_role_assignments` (идемпотентно при повторном вызове).

### 2.3. Загрузка политик при старте приложения

В `internal/app/app.go` после `RegisterRoutes` вызываются:

1. `container.PermissionService.EnsureSystemRolePolicies(context.Background())`  
2. `container.PermissionService.SyncGroupingPoliciesFromAssignments(context.Background())`

Таким образом, при каждом запуске приложения Casbin заполняется системными политиками по всем workspace и актуальными назначениями ролей из БД.

### 2.4. API эндпоинты

Все эндпоинты защищены цепочкой middleware (аутентификация, workspace, модуль, права в режиме логирования).

**В контексте workspace** (`/api/v1/workspaces/:workspaceId/...`):

| Метод | Путь | Назначение |
|-------|------|------------|
| GET | `/permissions/catalog` | Каталог прав (для UI настройки ролей). |
| GET | `/roles` | Список ролей workspace. |
| POST | `/roles` | Создание кастомной роли (body: name, description, permissions[]). |
| GET | `/roles/:roleId` | Получение роли по ID. |
| PUT | `/roles/:roleId` | Обновление роли (name, description, permissions). |
| DELETE | `/roles/:roleId` | Удаление кастомной роли (если нет назначений). |
| GET | `/roles/:roleId/permissions` | Список прав роли (из Casbin). |
| GET | `/users/:userId/roles` | Роли пользователя в workspace. |
| POST | `/users/:userId/roles/:roleId` | Назначить роль пользователю. |
| DELETE | `/users/:userId/roles/:roleId` | Снять роль. |
| GET | `/users/:userId/permissions` | Индивидуальные права пользователя. |
| POST | `/users/:userId/permissions` | Выдать право (body: permissionId, expiresAt?). |
| DELETE | `/users/:userId/permissions/:permissionId` | Отозвать право. |

**Текущий пользователь** (без workspaceId в path):

| Метод | Путь | Назначение |
|-------|------|------------|
| GET | `/me/permissions?workspaceId=...` | Эффективные права и список ролей текущего пользователя в указанном workspace. |

### 2.5. Интеграция в DI и маршруты

- В `internal/di/container.go`: создаются `permissionRepo`, `PermissionService`, `PermissionHandler`; в контейнер добавлены `PermissionService` и `PermissionHandler`.
- Группа `me`: `GET /me/permissions` регистрируется на `protected`.
- Группа `wsIDGroup`: на неё вешаются все маршруты хендлера прав (`RegisterRoutes(wsIDGroup)`), т.е. все пути выше под префиксом `/api/v1/workspaces/:workspaceId/...`.

---

## 3. Что осталось на следующий этап

- **Включить боевой режим PermissionMiddleware:** сейчас проверка через Casbin только логируется; нужно по результату `Enforce` и проверке индивидуальных прав возвращать 403 при отказе.
- **Учёт индивидуальных прав в PermissionMiddleware:** при отказе по ролям проверять `user_permissions` для данного пользователя и workspace и по каталогу сверять (obj, act); при совпадении — разрешать доступ.
- **Расширение маппинга endpoint → permission:** при необходимости покрыть оставшиеся эндпоинты (например, пайплайны CRM, экспорт) в `mapEndpointToPermission`.
- **Право на управление ролями:** эндпоинты управления ролями/правами должны быть защищены правом `workspace:role:manage`; после включения боевого режима это будет обеспечиваться через тот же PermissionMiddleware.

---

## 4. Резюме для синхронизации

- **Спецификация ролей** зафиксирована в разделе 1: термины, жизненный цикл, связь с Casbin, системные наборы прав, эффективные права.
- **Часть 3** добавляет: репозиторий и модели прав, полный PermissionService с синхронизацией в Casbin, загрузку системных политик и группировок при старте, полный набор API для каталога, ролей, назначений и индивидуальных прав, а также `GET /me/permissions` для фронта.
- Фронтенд может опираться на `GET /me/permissions?workspaceId=...` и на эндпоинты управления ролями/правами; боевая проверка прав в middleware будет включена на следующем этапе.
