# Архитектура ролей на бэкенде

**Навигация по всем документам SYSTEM:** [README.md](./README.md).

См. также полный обзор с **data scope**, таблицей `role_object_scopes` и цепочкой middleware: [ROLES_PERMISSIONS_AND_DATA_SCOPE.md](./ROLES_PERMISSIONS_AND_DATA_SCOPE.md). Полная модель прав и API: [PERMISSIONS/README_PERMISSIONS_ROLES.md](./PERMISSIONS/README_PERMISSIONS_ROLES.md).

## 1. Обзор

Система ролей обеспечивает гибкое управление доступом внутри workspace. Поддерживаются системные роли (OWNER, ADMIN, MEMBER, GUEST) и кастомные роли с произвольным набором прав.

---

## 2. Хранение данных

### 2.1. Таблицы

| Таблица | Назначение |
|---------|------------|
| `workspace_roles` | Роли workspace (системные + кастомные). `is_system=true` для OWNER/ADMIN/MEMBER/GUEST |
| `user_role_assignments` | Назначение ролей пользователям: `(user_id, role_id, workspace_id)` |
| `user_workspaces` | Членство в workspace + `role` (имя роли для отображения и синхронизации) |
| `permission_catalog` | Справочник всех возможных прав `(module_code, entity_type, action)` |
| `user_permissions` | Индивидуальные права пользователя (минуя роли) |
| `role_inheritance` | Наследование ролей: child ← parent |
| `role_object_scopes` | Видимость данных по роли: `(role_id, object_key, data_scope)` — см. [ROLES_PERMISSIONS_AND_DATA_SCOPE.md](./ROLES_PERMISSIONS_AND_DATA_SCOPE.md) |
| `casbin_rule` | Политики Casbin: `p(role, workspace, obj, act)` и `g(user, role, workspace)` |

### 2.2. Связи

```
workspaces ─┬─ workspace_roles (1:N)
            ├─ user_workspaces (N:M через users)
            └─ user_role_assignments (через workspace_roles)

users ──────┬─ user_workspaces (членство)
            ├─ user_role_assignments (роли)
            └─ user_permissions (индивидуальные права)

workspace_roles ─┬─ user_role_assignments
                └─ role_inheritance (child/parent)
```

---

## 3. Casbin

### 3.1. Модель

- **Субъект (sub)**: `user:<user_id>` или `role:<role_name>`
- **Домен (dom)**: `workspace_id`
- **Объект (obj)**: `module:entity` (например `crm:deal`)
- **Действие (act)**: `create`, `read`, `update`, `delete`, `move`, `manage` и т.д.

### 3.2. Политики

- **p** (policy): `p("role:OWNER", workspace_id, "crm:deal", "create")` — роль имеет право
- **g** (grouping): `g("user:uuid", "role:OWNER", workspace_id)` — пользователь имеет роль в workspace
- **g** (inheritance): `g("role:Seller", "role:GUEST", workspace_id)` — Seller наследует GUEST

### 3.3. Проверка

`Enforcer.Enforce("user:"+userID, workspaceID, "crm:deal", "read")` — Casbin находит роли пользователя через `g` и проверяет наличие политики `p`.

---

## 4. Поток данных

### 4.1. Создание workspace

1. `workspaces` — новая запись
2. Триггер `tr_create_system_roles` → создаёт 4 записи в `workspace_roles` (OWNER, ADMIN, MEMBER, GUEST)
3. `user_workspaces` — владелец с role=OWNER
4. `EnsureSystemRolePolicies` — заливает политики для системных ролей в Casbin
5. `AssignRoleByName(OWNER)` — добавляет `g(user, role:OWNER, workspace)` в Casbin

### 4.2. Назначение роли участнику

1. `UpdateMemberRole` или `AssignRole`:
   - `user_workspaces.role` = имя роли (системной или кастомной)
   - `user_role_assignments` — добавляется запись `(user_id, role_id, workspace_id)`
   - Casbin: `AddGroupingPolicy("user:"+userID, "role:"+roleName, workspaceID)`

### 4.3. GetEffectivePermissions (для UI)

1. Получить роли пользователя из `user_role_assignments`
2. **Override semantics**: если есть кастомная роль — использовать только её права (игнорировать системные)
3. Собрать политики Casbin для ролей
4. Добавить индивидуальные права из `user_permissions`
5. Вернуть объединённый список `module:entity:action`

---

## 5. Масштабирование и расширение

### 5.1. Горизонтальное масштабирование

- **Casbin**: политики хранятся в PostgreSQL (`casbin_rule`). При нескольких инстансах API каждый загружает политики при старте. При изменении прав нужно вызывать `LoadPolicy()` на всех инстансах или использовать Redis/Watch для инвалидации.
- **Рекомендация**: единый инстанс или shared cache для Casbin-политик.

### 5.2. Расширение прав

- Добавить запись в `permission_catalog` (миграция или админка)
- Добавить маппинг в `PermissionMiddleware` для новых эндпоинтов: `(method, path) → (obj, act)`
- На фронте — расширить `WorkspacePermission` и `WORKSPACE_TO_API_PREFIXES`

### 5.3. Новые модули

- Создать модуль в `modules`
- Добавить права в `permission_catalog` для сущностей модуля
- Добавить политики в `addSystemPoliciesForWorkspace` для системных ролей (если нужно)

### 5.4. Роли под должности (Seller, HR)

- Создать кастомную роль с нужным набором прав
- Назначать через UI участников вместо системной роли
- Override semantics: пользователь получает только права кастомной роли

---

## 6. Зависимости

```
WorkspaceService ──► PermissionService (AssignRole, GetUserRoles)
PermissionService ──► WorkspaceRepository (SetUserWorkspaceRole при DeleteRole)
PermissionService ──► Casbin Enforcer
PermissionMiddleware ──► Casbin Enforcer, PermissionService
```

---

## 7. Ключевые операции

| Операция | Таблицы | Casbin |
|----------|---------|--------|
| CreateRole | workspace_roles | AddPolicy для каждой права |
| UpdateRole | workspace_roles | RemovePolicy старые, AddPolicy новые |
| DeleteRole | user_workspaces, user_role_assignments, workspace_roles | RemoveGroupingPolicy, RemovePolicy, AddGroupingPolicy(GUEST) |
| AssignRole | user_role_assignments, user_workspaces | AddGroupingPolicy |
| RemoveRole | user_role_assignments | RemoveGroupingPolicy, SavePolicy |
