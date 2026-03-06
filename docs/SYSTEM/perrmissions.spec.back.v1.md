# Спецификация задачи: Внедрение гибкой системы ролей и прав доступа
## Для команды бэкенд-разработки

**Версия 1.0** | Март 2026

---

## Содержание

1. **Краткое описание задачи**
2. **Термины и определения**
3. **Текущее состояние**
4. **Что нужно сделать**
5. **Детальные требования к реализации**
   - 5.1 Новые таблицы в БД
   - 5.2 Middleware
   - 5.3 Сервисный слой
   - 5.4 API эндпоинты
   - 5.5 Интеграция с Casbin
6. **Миграция данных**
7. **Критерии приемки**
8. **Оценка сроков**
9. **Чек-лист для разработчика**

---

## 1. Краткое описание задачи

**Цель:** Заменить жестко зашитые проверки ролей (ADMIN/OWNER/MEMBER/GUEST) на гибкую систему гранулярных прав с возможностью создания кастомных ролей.

**Зачем:** Сейчас нельзя настроить права детальнее, чем 4 предопределенные роли. Бизнес требует возможности создать роль "Менеджер по продажам" с правом только на создание сделок, но без удаления, или дать временный доступ конкретному сотруднику.

**Что получим на выходе:**
- Кастомные роли в каждом workspace
- Гранулярные права (create/read/update/delete/export на каждую сущность)
- Индивидуальные права для пользователей
- Наследование ролей
- API для управления всем этим (будет использоваться фронтендом)

---

## 2. Термины и определения

| Термин | Определение |
|--------|-------------|
| **Workspace** | Рабочее пространство компании. Все данные изолированы по workspace_id. |
| **Право (Permission)** | Разрешение вида `{модуль}:{сущность}:{действие}`, например `crm:deal:create` |
| **Роль (Role)** | Именованный набор прав, например "Sales Manager" |
| **Системная роль** | OWNER, ADMIN, MEMBER, GUEST — существуют всегда, нельзя удалить |
| **Индивидуальное право** | Право, выданное конкретному пользователю напрямую (не через роль) |
| **Наследование** | Роль может наследовать права другой роли |
| **Casbin** | Библиотека авторизации, которая будет проверять права |

---

## 3. Текущее состояние

### 3.1. Существующие таблицы, которые мы будем использовать

```sql
-- Связь пользователя с workspace (уже есть)
user_workspaces (
    user_id UUID,
    workspace_id UUID,
    role VARCHAR(20)  -- OWNER/ADMIN/MEMBER/GUEST
)

-- Модули системы (уже есть)
modules (
    id UUID,
    code VARCHAR(50)  -- 'crm', 'habits', 'projects'
)

-- Какие модули включены в workspace (уже есть)
workspace_modules (
    workspace_id UUID,
    module_id UUID,
    status VARCHAR(20)
)

-- Лицензии пользователей на модули (уже есть)
user_module_licenses (
    user_id UUID,
    module_id UUID,
    scope VARCHAR(20),
    workspace_id UUID
)
```

### 3.2. Текущая логика проверки прав (проблема)

```go
// Пример того, что сейчас размазано по всем хендлерам
func (h *DealHandler) UpdateDeal(c *gin.Context) {
    userWorkspace, _ := h.userWorkspaceRepo.Find(userID, workspaceID)
    if userWorkspace.Role != "ADMIN" && userWorkspace.Role != "OWNER" {
        return error
    }
    // бизнес-логика
}
```

**Проблемы:**
- Нельзя дать MEMBER права на редактирование, но запретить удаление
- Нельзя создать новую роль
- Код проверки дублируется

---

## 4. Что нужно сделать

### 4.1. Общая архитектура

```
[Запрос] → [AuthMiddleware] → [WorkspaceMiddleware] → [ModuleMiddleware] → [PermissionMiddleware] → [Хендлер]
                |                       |                       |                       |
            проверяет               извлекает              проверяет              проверяет
            JWT, кладет            workspace_id            модуль и               права через
            user_id                из URL,                 лицензию               Casbin
                                   проверяет
                                   членство
```

### 4.2. Компоненты для реализации

1. **Новые таблицы** (5 штук) — см. раздел 5.1
2. **Middleware** (3 новых) — WorkspaceMiddleware, ModuleMiddleware, PermissionMiddleware
3. **PermissionService** — сервис для управления ролями и синхронизации с Casbin
4. **API endpoints** (10-12 эндпоинтов) — для управления ролями и правами
5. **Интеграция с Casbin** — подключение библиотеки, настройка модели

---

## 5. Детальные требования к реализации

### 5.1. Новые таблицы (SQL)

#### 5.1.1. permission_catalog — словарь всех возможных прав

```sql
CREATE TABLE permission_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_code VARCHAR(50) NOT NULL,      -- crm, habits, projects
    entity_type VARCHAR(50) NOT NULL,      -- deal, contact, company
    action VARCHAR(50) NOT NULL,           -- create, read, update, delete, manage
    name VARCHAR(255) NOT NULL,             -- "Создание сделки" (для UI)
    description TEXT,
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(module_code, entity_type, action)
);

CREATE INDEX idx_permission_catalog_module ON permission_catalog(module_code);
```

**Что нужно заполнить сразу:**
- CRM: deal (create/read/update/delete/move), contact (create/read/update/delete), company (create/read/update/delete), pipeline (manage)
- Habits: habit (create/read/update/delete/complete), journal (create/read/update/delete)
- Projects: project (create/read/update/delete), entity (attach/detach)
- Workspace: member (invite/remove), role (manage), module (manage)

#### 5.1.2. workspace_roles — роли в workspace

```sql
CREATE TABLE workspace_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT false,       -- true для OWNER/ADMIN/MEMBER/GUEST
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, name)
);

CREATE INDEX idx_workspace_roles_workspace ON workspace_roles(workspace_id);
```

**Триггер:** При создании workspace автоматически добавлять роли OWNER, ADMIN, MEMBER, GUEST.

#### 5.1.3. user_role_assignments — назначение ролей пользователям

```sql
CREATE TABLE user_role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id),
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, role_id, workspace_id)
);

CREATE INDEX idx_user_role_assignments_user ON user_role_assignments(user_id);
CREATE INDEX idx_user_role_assignments_role ON user_role_assignments(role_id);
CREATE INDEX idx_user_role_assignments_workspace ON user_role_assignments(workspace_id);
CREATE INDEX idx_user_role_assignments_lookup ON user_role_assignments(user_id, workspace_id);
```

#### 5.1.4. user_permissions — индивидуальные права (минуя роли)

```sql
CREATE TABLE user_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permission_catalog(id) ON DELETE CASCADE,
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,
    UNIQUE(user_id, workspace_id, permission_id)
);

CREATE INDEX idx_user_permissions_user ON user_permissions(user_id);
CREATE INDEX idx_user_permissions_workspace ON user_permissions(workspace_id);
CREATE INDEX idx_user_permissions_lookup ON user_permissions(user_id, workspace_id);
```

#### 5.1.5. role_inheritance — наследование ролей

```sql
CREATE TABLE role_inheritance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    child_role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    parent_role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CHECK (child_role_id != parent_role_id),
    UNIQUE(workspace_id, child_role_id, parent_role_id)
);

CREATE INDEX idx_role_inheritance_child ON role_inheritance(child_role_id);
CREATE INDEX idx_role_inheritance_parent ON role_inheritance(parent_role_id);
```

### 5.2. Middleware (3 новых)

#### 5.2.1. WorkspaceMiddleware

**Назначение:** Извлечь workspace_id из URL, проверить членство пользователя.

**Вход:** Контекст с user_id (из AuthMiddleware)
**Выход:** Контекст с workspace_id

**Логика:**
1. Извлечь workspace_id из URL (паттерн `/api/v1/workspaces/{workspaceId}/...`)
2. Если нет workspace_id в URL — пропустить (публичный эндпоинт)
3. Если есть user_id, проверить `user_workspaces` на наличие записи (user_id, workspace_id)
4. Если записи нет — вернуть 403
5. Положить workspace_id в контекст

#### 5.2.2. ModuleMiddleware

**Назначение:** Проверить, включен ли модуль в workspace и есть ли лицензия.

**Вход:** Контекст с user_id, workspace_id
**Выход:** Контекст с module_code

**Логика:**
1. Определить модуль по пути (contains "/crm/" → "crm", "/habits/" → "habits" и т.д.)
2. Если модуль не определен — пропустить
3. Проверить `workspace_modules`: есть ли активная запись для (workspace_id, module_id)
4. Если нет — вернуть 403
5. Если модуль не core (проверить по `modules.is_core`) и есть user_id:
   - Проверить `user_module_licenses` для пользователя
   - Если нет активной лицензии — вернуть 403
6. Положить module_code в контекст

#### 5.2.3. PermissionMiddleware

**Назначение:** Проверить, есть ли у пользователя право на конкретное действие.

**Вход:** Контекст с user_id, workspace_id, запрос
**Выход:** Доступ разрешен/запрещен

**Логика:**
1. Сопоставить HTTP метод и путь с объектом и действием через таблицу маппинга (см. пример в 5.2.4)
2. Если соответствия нет (публичный эндпоинт) — пропустить
3. Если user_id отсутствует — вернуть 401
4. Получить все роли пользователя в этом workspace:
   ```sql
   SELECT wr.name FROM user_role_assignments ura
   JOIN workspace_roles wr ON wr.id = ura.role_id
   WHERE ura.user_id = $1 AND ura.workspace_id = $2
   ```
5. Для каждой роли проверить через Casbin:
   ```
   enforcer.Enforce("role:" + roleName, workspace_id, object, action)
   ```
6. Если ни одна роль не дала доступа, проверить индивидуальные права:
   ```sql
   SELECT p.* FROM user_permissions up
   JOIN permission_catalog p ON p.id = up.permission_id
   WHERE up.user_id = $1 AND up.workspace_id = $2 
     AND p.module_code = $3 AND p.entity_type = $4 AND p.action = $5
   ```
7. Если есть совпадение — доступ разрешен, иначе 403

#### 5.2.4. Маппинг эндпоинтов (пример)

Создать структуру, которая по методу и пути возвращает (object, action):

```go
// Псевдокод, реализация на усмотрение разработчика
endpointMap := map[string]map[string]Permission{
    "POST": {
        "/api/v1/workspaces/:workspaceId/deals":      {Object: "crm:deal", Action: "create"},
        "/api/v1/workspaces/:workspaceId/contacts":   {Object: "crm:contact", Action: "create"},
        "/api/v1/workspaces/:workspaceId/pipelines":  {Object: "crm:pipeline", Action: "manage"},
    },
    "GET": {
        "/api/v1/workspaces/:workspaceId/deals":      {Object: "crm:deal", Action: "read"},
        "/api/v1/workspaces/:workspaceId/deals/:id":  {Object: "crm:deal", Action: "read"},
    },
    "PUT": {
        "/api/v1/workspaces/:workspaceId/deals/:id":  {Object: "crm:deal", Action: "update"},
    },
    "DELETE": {
        "/api/v1/workspaces/:workspaceId/deals/:id":  {Object: "crm:deal", Action: "delete"},
    },
}
```

**Важно:** Маппинг должен покрывать ВСЕ существующие эндпоинты (список можно получить из вывода go run).

### 5.3. Сервисный слой (PermissionService)

#### 5.3.1. Методы для работы с ролями

```go
type PermissionService interface {
    // Получение каталога прав (для UI)
    GetCatalog(ctx context.Context) ([]PermissionCatalogItem, error)
    
    // CRUD для ролей
    CreateRole(ctx context.Context, workspaceID, name, description string, permissions []string, createdBy string) (*WorkspaceRole, error)
    UpdateRole(ctx context.Context, roleID, name, description string, permissions []string) error
    DeleteRole(ctx context.Context, roleID string) error
    GetRole(ctx context.Context, roleID string) (*WorkspaceRole, error)
    ListRoles(ctx context.Context, workspaceID string) ([]WorkspaceRole, error)
    GetRolePermissions(ctx context.Context, roleID string) ([]Permission, error)
    
    // Назначение ролей пользователям
    AssignRole(ctx context.Context, userID, roleID, workspaceID, assignedBy string) error
    RemoveRole(ctx context.Context, userID, roleID, workspaceID string) error
    GetUserRoles(ctx context.Context, userID, workspaceID string) ([]WorkspaceRole, error)
    
    // Индивидуальные права
    GrantPermission(ctx context.Context, userID, workspaceID string, permissionID string, grantedBy string, expiresAt *time.Time) error
    RevokePermission(ctx context.Context, userID, workspaceID, permissionID string) error
    GetUserPermissions(ctx context.Context, userID, workspaceID string) ([]Permission, error)
    
    // Наследование
    AddInheritance(ctx context.Context, workspaceID, childRoleID, parentRoleID string) error
    RemoveInheritance(ctx context.Context, workspaceID, childRoleID, parentRoleID string) error
    
    // Для UI (получение всех прав пользователя)
    GetEffectivePermissions(ctx context.Context, userID, workspaceID string) ([]string, error)
}
```

**Важно:** Каждый метод, изменяющий данные, должен синхронизировать изменения с Casbin:
- При создании роли: добавить политики для каждого права
- При назначении роли: добавить групповую политику `g, user:<id>, role:<name>, workspace_id`
- При удалении: удалить соответствующие политики

### 5.4. API эндпоинты

Все эндпоинты должны быть задокументированы в Swagger.

#### 5.4.1. Получение каталога прав (для UI)
```
GET /api/v1/workspaces/{workspaceId}/permissions/catalog
Response: {
  "modules": [
    {
      "code": "crm",
      "name": "CRM",
      "entities": [
        {
          "type": "deal",
          "name": "Сделки",
          "actions": [
            {"code": "create", "name": "Создание"},
            {"code": "read", "name": "Просмотр"}
          ]
        }
      ]
    }
  ]
}
```

#### 5.4.2. Управление ролями
```
GET    /api/v1/workspaces/{workspaceId}/roles
POST   /api/v1/workspaces/{workspaceId}/roles
GET    /api/v1/workspaces/{workspaceId}/roles/{roleId}
PUT    /api/v1/workspaces/{workspaceId}/roles/{roleId}
DELETE /api/v1/workspaces/{workspaceId}/roles/{roleId}
```

**Body для POST/PUT:**
```json
{
  "name": "Sales Manager",
  "description": "Управляет продажами",
  "permissions": [
    "crm:deal:create",
    "crm:deal:read",
    "crm:deal:update",
    "crm:contact:read"
  ]
}
```

#### 5.4.3. Назначение ролей пользователям
```
GET    /api/v1/workspaces/{workspaceId}/users/{userId}/roles
POST   /api/v1/workspaces/{workspaceId}/users/{userId}/roles/{roleId}
DELETE /api/v1/workspaces/{workspaceId}/users/{userId}/roles/{roleId}
```

#### 5.4.4. Индивидуальные права
```
GET    /api/v1/workspaces/{workspaceId}/users/{userId}/permissions
POST   /api/v1/workspaces/{workspaceId}/users/{userId}/permissions
DELETE /api/v1/workspaces/{workspaceId}/users/{userId}/permissions/{permissionId}
```

**Body для POST:**
```json
{
  "permissionId": "uuid",
  "expiresAt": "2026-12-31T23:59:59Z" // опционально
}
```

#### 5.4.5. Наследование ролей
```
POST   /api/v1/workspaces/{workspaceId}/roles/{roleId}/inherit/{parentRoleId}
DELETE /api/v1/workspaces/{workspaceId}/roles/{roleId}/inherit/{parentRoleId}
```

#### 5.4.6. Для фронтенда (текущий пользователь)
```
GET /api/v1/me/permissions?workspaceId={id}
Response: {
  "permissions": ["crm:deal:create", "crm:deal:read"],
  "roles": ["Sales Manager", "MEMBER"]
}
```

### 5.5. Интеграция с Casbin

#### 5.5.1. Подключение библиотек
```bash
go get github.com/casbin/casbin/v2
go get github.com/casbin/gorm-adapter/v3
```

#### 5.5.2. Модель Casbin (model.conf)
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

#### 5.5.3. Инициализация enforcer
```go
func InitCasbin(db *gorm.DB) (*casbin.Enforcer, error) {
    adapter, err := gormadapter.NewAdapterByDB(db)
    if err != nil {
        return nil, err
    }
    
    // Загрузить модель из строки или файла
    enforcer, err := casbin.NewEnforcer("path/to/model.conf", adapter)
    if err != nil {
        return nil, err
    }
    
    enforcer.EnableCache(true)
    enforcer.LoadPolicy()
    
    return enforcer, nil
}
```

#### 5.5.4. Синхронизация с БД

При каждом изменении ролей/назначений вызывать методы enforcer:

```go
// При создании роли с правами
for _, perm := range permissions {
    // perm имеет формат "obj:act", например "crm:deal:create"
    enforcer.AddPolicy("role:"+roleName, workspaceID, obj, act)
}

// При назначении роли пользователю
enforcer.AddGroupingPolicy("user:"+userID, "role:"+roleName, workspaceID)

// При удалении - аналогично RemovePolicy/RemoveGroupingPolicy
```

---

## 6. Миграция данных

### 6.1. Последовательность действий

1. **Создать новые таблицы** (скрипты из раздела 5.1)

2. **Заполнить permission_catalog** (вставить все возможные права из списка в 5.1.1)

3. **Создать системные роли для всех workspace**
```sql
INSERT INTO workspace_roles (id, workspace_id, name, is_system)
SELECT gen_random_uuid(), id, 'OWNER', true FROM workspaces
UNION ALL
SELECT gen_random_uuid(), id, 'ADMIN', true FROM workspaces
UNION ALL
SELECT gen_random_uuid(), id, 'MEMBER', true FROM workspaces
UNION ALL
SELECT gen_random_uuid(), id, 'GUEST', true FROM workspaces;
```

4. **Перенести существующие назначения из user_workspaces**
```sql
INSERT INTO user_role_assignments (id, user_id, role_id, workspace_id, assigned_at)
SELECT 
    gen_random_uuid(),
    uw.user_id,
    wr.id,
    uw.workspace_id,
    uw.created_at
FROM user_workspaces uw
JOIN workspace_roles wr ON wr.workspace_id = uw.workspace_id AND wr.name = uw.role;
```

5. **Добавить базовые политики в Casbin** (через код)
   - Для роли ADMIN: все права на всё
   - Для роли OWNER: все права на всё
   - Для MEMBER: базовые права (create/read/update для своих сущностей)
   - Для GUEST: только read права

### 6.2. Проверка миграции

```sql
-- Количество записей должно совпадать
SELECT COUNT(*) FROM user_workspaces;
SELECT COUNT(*) FROM user_role_assignments;

-- Проверка выборочно
SELECT * FROM user_workspaces uw
LEFT JOIN user_role_assignments ura ON ura.user_id = uw.user_id AND ura.workspace_id = uw.workspace_id
WHERE ura.id IS NULL; -- не должно быть записей
```

---

## 7. Критерии приемки

### 7.1. Функциональные критерии

- [ ] Созданы все 5 новых таблиц с правильными индексами
- [ ] WorkspaceMiddleware корректно извлекает workspace_id и проверяет членство
- [ ] ModuleMiddleware корректно проверяет модуль и лицензии
- [ ] PermissionMiddleware корректно проверяет права через Casbin
- [ ] Можно создать кастомную роль через API
- [ ] Можно назначить роль пользователю
- [ ] После назначения роли пользователь получает соответствующие права
- [ ] Можно выдать индивидуальное право пользователю
- [ ] Работает наследование ролей
- [ ] API `/me/permissions` возвращает все права текущего пользователя
- [ ] Все существующие эндпоинты продолжают работать (с новыми правилами)

### 7.2. Технические критерии

- [ ] Время проверки прав < 5 мс (в среднем под нагрузкой)
- [ ] Все новые эндпоинты задокументированы в Swagger
- [ ] Написаны unit-тесты для PermissionService
- [ ] Написаны интеграционные тесты для middleware
- [ ] Миграция данных прошла без ошибок на копии продакшн БД
- [ ] Код покрыт комментариями

### 7.3. Критерии безопасности

- [ ] Системные роли (OWNER/ADMIN/MEMBER/GUEST) нельзя удалить через API
- [ ] Нельзя создать роль с пустым именем
- [ ] Проверка прав происходит до бизнес-логики
- [ ] Все изменения прав логируются (кто, когда, что изменил)

---

## 8. Оценка сроков

| Этап | Задачи | Примерное время |
|------|--------|-----------------|
| **Этап 1** | Создание таблиц, наполнение каталога, миграция данных | 3 дня |
| **Этап 2** | Интеграция Casbin, настройка enforcer, синхронизация | 3 дня |
| **Этап 3** | Реализация middleware (Workspace, Module, Permission) | 4 дня |
| **Этап 4** | Реализация PermissionService | 4 дня |
| **Этап 5** | Реализация API эндпоинтов | 4 дня |
| **Этап 6** | Тестирование, отладка, документирование | 3 дня |
| **Резерв** | Непредвиденные сложности | 2 дня |
| **ИТОГО** | | **23 рабочих дня** |

---

## 9. Чек-лист для разработчика

### Перед началом
- [ ] Ознакомиться с существующей структурой БД
- [ ] Разобраться в текущей логике авторизации
- [ ] Установить библиотеки: casbin, gorm-adapter

### В процессе
- [ ] Создать новые таблицы через миграции
- [ ] Написать код для заполнения permission_catalog
- [ ] Реализовать миграцию данных из user_workspaces
- [ ] Настроить Casbin enforcer
- [ ] Реализовать 3 middleware
- [ ] Реализовать PermissionService со всеми методами
- [ ] Реализовать API эндпоинты
- [ ] Добавить валидацию на все входные данные
- [ ] Написать тесты
- [ ] Обновить Swagger документацию

### Перед сдачей
- [ ] Проверить, что все старые хендлеры работают (пока без удаления их проверок)
- [ ] Проверить производительность (замерить время с middleware)
- [ ] Проверить, что миграция прошла полностью
- [ ] Убедиться, что системные роли нельзя удалить
- [ ] Провести код-ревью с командой

---

**Документ подготовлен:** Март 2026  
**Версия:** 1.0  
**Ответственный:** Team Lead  
**Статус:** Утверждено к реализации
