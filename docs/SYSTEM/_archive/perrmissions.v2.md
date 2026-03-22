# Системный анализ: Внедрение гибкой системы ролей и прав доступа

## Версия 4.0 (Полный разбор для бэкенда) | Март 2026

---

## Содержание

1. **Введение и цели**
2. **Анализ текущей системы авторизации**
3. **Концептуальная модель прав доступа**
4. **Детальный проект базы данных**
5. **Архитектура middleware**
6. **Интеграция с Casbin**
7. **API для управления ролями и правами**
8. **Модификация существующих хендлеров**
9. **Миграция данных и обратная совместимость**
10. **Тестирование и обеспечение качества**
11. **План поэтапного внедрения**
12. **Технические риски и их mitigation**

---

## 1. Введение и цели

### 1.1. Контекст

Текущая ERP-система имеет базовую ролевую модель: каждый пользователь в workspace имеет одну из четырех ролей (OWNER, ADMIN, MEMBER, GUEST), которые жестко определяют набор доступных действий. Это ограничивает гибкость настройки системы под потребности конкретного бизнеса.

### 1.2. Цели внедрения

| Цель | Описание | Бизнес-ценность |
|------|----------|-----------------|
| **Гранулярность** | Возможность настроить доступ к каждому действию (create/read/update/delete/export) | Точная настройка безопасности |
| **Кастомные роли** | Создание ролей под структуру компании (например, "Менеджер по продажам", "Лид-генератор") | Адаптация под бизнес-процессы |
| **Наследование** | Иерархия ролей (Senior Manager = Manager + доп. права) | Упрощение управления |
| **Индивидуальные права** | Возможность дать конкретному пользователю особое право | Гибкость для исключений |
| **Аудит** | Логирование всех изменений прав | Безопасность и compliance |
| **UI-интеграция** | API для фронтенда, чтобы скрывать недоступные элементы | Улучшение UX |

### 1.3. Принципы проектирования

1. **Не ломать существующее** — обратная совместимость с текущим кодом
2. **Постепенность** — поэтапное внедрение без "большого взрыва"
3. **Производительность** — проверка прав < 5 мс
4. **Прозрачность** — понятная модель данных и API
5. **Масштабируемость** — поддержка тысяч ролей и политик

---

## 2. Анализ текущей системы авторизации

### 2.1. Существующая структура БД

```sql
-- Таблица пользователей
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(100),
    role VARCHAR(20) NOT NULL DEFAULT 'USER',  -- Глобальная роль (USER, ADMIN)
    avatar_url TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Таблица workspace
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(50) DEFAULT '#3B82F6',
    owner_id UUID NOT NULL,                    -- Владелец workspace
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- СВЯЗЬ пользователей с workspace (ТЕКУЩАЯ РОЛЕВАЯ МОДЕЛЬ)
CREATE TABLE user_workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'MEMBER',         -- OWNER, ADMIN, MEMBER, GUEST
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, workspace_id)
);

-- Модули и лицензии
CREATE TABLE modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE,          -- crm, habits, projects
    name VARCHAR(255) NOT NULL,
    is_core BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE workspace_modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    UNIQUE(workspace_id, module_id)
);

CREATE TABLE user_module_licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    scope VARCHAR(20) NOT NULL,                -- all_workspaces, single_workspace
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active'
);
```

### 2.2. Текущий middleware (анализ)

```go
// Текущий AuthMiddleware
func AuthMiddleware(tokenGen *token.Generator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Получение токена из куки или заголовка
            var tokenString string
            if cookie, err := r.Cookie("access_token"); err == nil {
                tokenString = cookie.Value
            } else {
                authHeader := r.Header.Get("Authorization")
                tokenString = strings.TrimPrefix(authHeader, "Bearer ")
            }

            // 2. Валидация токена
            claims, err := tokenGen.Validate(tokenString)
            if err != nil {
                next.ServeHTTP(w, r)  // Пропускаем как неавторизованный
                return
            }

            // 3. Добавление в контекст
            ctx := r.Context()
            ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
            ctx = context.WithValue(ctx, RoleKey, model.UserRole(claims.Role))
            ctx = context.WithValue(ctx, ClaimsKey, claims)

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

**Критическое наблюдение:** Middleware **НЕ БЛОКИРУЕТ** неавторизованные запросы, а пропускает их. Это означает, что ответственность за проверку авторизации лежит на хендлерах.

### 2.3. Текущая логика в хендлерах (проблема)

```go
// Типичный хендлер сейчас
func (h *DealHandler) UpdateDeal(c *gin.Context) {
    // 1. Получение user_id из контекста (может быть пустым!)
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(401, gin.H{"error": "unauthorized"})
        return
    }

    // 2. Получение workspace_id из URL
    workspaceID := c.Param("workspaceId")

    // 3. Проверка членства в workspace
    userWorkspace, err := h.userWorkspaceRepo.Find(userID.(string), workspaceID)
    if err != nil {
        c.JSON(403, gin.H{"error": "access denied"})
        return
    }

    // 4. ХАРДКОД: проверка роли
    if userWorkspace.Role != "ADMIN" && userWorkspace.Role != "OWNER" {
        c.JSON(403, gin.H{"error": "insufficient permissions"})
        return
    }

    // 5. Бизнес-логика...
}
```

**Проблемы:**
- Логика авторизации дублируется в каждом хендлере
- Жесткая привязка к ролям (нельзя настроить)
- Нет централизованного управления
- Нет кэширования результатов проверки

### 2.4. Что уже хорошо и можно использовать

| Компонент | Статус | Использование |
|-----------|--------|---------------|
| `users` таблица | ✅ | Хранит пользователей |
| `workspaces` таблица | ✅ | Хранит workspace |
| `user_workspaces` | ✅ | Проверка членства в workspace |
| `modules` и `workspace_modules` | ✅ | Проверка доступности модуля |
| `user_module_licenses` | ✅ | Проверка лицензий |
| AuthMiddleware | ✅ | Устанавливает `user_id` в контекст |
| URL паттерн `/workspaces/:workspaceId/...` | ✅ | Легко извлечь workspace_id |

---

## 3. Концептуальная модель прав доступа

### 3.1. Трехуровневая модель авторизации

```
УРОВЕНЬ 1: Membership
─────────────────────────────────────
Проверка: состоит ли пользователь в workspace?
Таблица: user_workspaces
Результат: пользователь имеет доступ к workspace

         ↓
         
УРОВЕНЬ 2: Module Access
─────────────────────────────────────
Проверка: включен ли модуль в workspace?
Таблица: workspace_modules
Проверка: есть ли у пользователя лицензия на модуль?
Таблица: user_module_licenses
Результат: пользователь может использовать модуль

         ↓
         
УРОВЕНЬ 3: Permissions
─────────────────────────────────────
Проверка: имеет ли пользователь право на конкретное действие?
Система: Casbin (роли + политики)
Результат: пользователь может выполнить действие
```

### 3.2. Иерархия сущностей

```
Workspace (контекст)
    │
    ├── Модули (CRM, Habits, Projects)
    │       │
    │       ├── Типы сущностей (deals, contacts, companies)
    │       │       │
    │       │       └── Действия (create, read, update, delete)
    │       │
    │       └── Специальные права (manage_pipelines, export)
    │
    ├── Роли (кастомные + системные)
    │       │
    │       ├── OWNER (встроенная, полный доступ)
    │       ├── ADMIN (встроенная, управление)
    │       ├── MEMBER (встроенная, базовый доступ)
    │       ├── GUEST (встроенная, только просмотр)
    │       └── Sales Manager (кастомная)
    │
    └── Пользователи (назначенные роли)
```

### 3.3. Матрица прав (пример для CRM)

| Действие | OWNER | ADMIN | MEMBER | GUEST | Sales Manager |
|----------|-------|-------|--------|-------|---------------|
| deal:create | ✅ | ✅ | ✅ | ❌ | ✅ |
| deal:read | ✅ | ✅ | ✅ | ✅ | ✅ |
| deal:update | ✅ | ✅ | ✅ | ❌ | ✅ |
| deal:delete | ✅ | ✅ | ❌ | ❌ | ❌ |
| deal:move | ✅ | ✅ | ✅ | ❌ | ✅ |
| pipeline:manage | ✅ | ✅ | ❌ | ❌ | ❌ |
| contact:create | ✅ | ✅ | ✅ | ❌ | ✅ |
| contact:read | ✅ | ✅ | ✅ | ✅ | ✅ |
| contact:update | ✅ | ✅ | ✅ | ❌ | ✅ |
| contact:delete | ✅ | ✅ | ❌ | ❌ | ❌ |
| export:deals | ✅ | ✅ | ❌ | ❌ | ✅ |

---

## 4. Детальный проект базы данных

### 4.1. Новые таблицы

#### 4.1.1. Каталог прав (permission_catalog)

```sql
-- Словарь всех возможных действий в системе
CREATE TABLE permission_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_code VARCHAR(50) NOT NULL,      -- crm, habits, projects, workspace
    entity_type VARCHAR(50) NOT NULL,      -- deal, contact, company, habit, journal
    action VARCHAR(50) NOT NULL,           -- create, read, update, delete, manage, export
    name VARCHAR(255) NOT NULL,             -- "Создание сделки" (для UI)
    description TEXT,
    is_system BOOLEAN DEFAULT false,       -- Системные права нельзя удалить
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Уникальность: в рамках модуля и типа сущности действие уникально
    UNIQUE(module_code, entity_type, action)
);

-- Индексы
CREATE INDEX idx_permission_catalog_module ON permission_catalog(module_code);
CREATE INDEX idx_permission_catalog_entity ON permission_catalog(entity_type);

-- Комментарии
COMMENT ON TABLE permission_catalog IS 'Каталог всех возможных прав в системе';
COMMENT ON COLUMN permission_catalog.module_code IS 'Код модуля (crm, habits, projects)';
COMMENT ON COLUMN permission_catalog.entity_type IS 'Тип сущности (deal, contact, habit)';
COMMENT ON COLUMN permission_catalog.action IS 'Действие (create, read, update, delete)';
```

#### 4.1.2. Роли workspace (workspace_roles)

```sql
CREATE TABLE workspace_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,              -- "Sales Manager"
    description TEXT,
    is_system BOOLEAN DEFAULT false,         -- true для OWNER, ADMIN, MEMBER, GUEST
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- В рамках workspace имена ролей уникальны
    UNIQUE(workspace_id, name)
);

-- Индексы
CREATE INDEX idx_workspace_roles_workspace ON workspace_roles(workspace_id);
CREATE INDEX idx_workspace_roles_name ON workspace_roles(name);

-- Комментарии
COMMENT ON TABLE workspace_roles IS 'Роли внутри workspace (системные и кастомные)';
COMMENT ON COLUMN workspace_roles.is_system IS 'Системные роли (OWNER, ADMIN, MEMBER, GUEST) нельзя удалить';

-- Триггер для автоматического создания системных ролей при создании workspace
CREATE OR REPLACE FUNCTION fn_create_system_roles()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO workspace_roles (workspace_id, name, is_system) VALUES
        (NEW.id, 'OWNER', true),
        (NEW.id, 'ADMIN', true),
        (NEW.id, 'MEMBER', true),
        (NEW.id, 'GUEST', true);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_create_system_roles
    AFTER INSERT ON workspaces
    FOR EACH ROW
    EXECUTE FUNCTION fn_create_system_roles();
```

#### 4.1.3. Назначение ролей пользователям (user_role_assignments)

```sql
CREATE TABLE user_role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id),   -- кто назначил
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Пользователь может иметь несколько ролей, но уникальность связки
    UNIQUE(user_id, role_id, workspace_id)
);

-- Индексы для быстрого поиска
CREATE INDEX idx_user_role_assignments_user ON user_role_assignments(user_id);
CREATE INDEX idx_user_role_assignments_role ON user_role_assignments(role_id);
CREATE INDEX idx_user_role_assignments_workspace ON user_role_assignments(workspace_id);

-- Составной индекс для частого запроса "все роли пользователя в workspace"
CREATE INDEX idx_user_role_assignments_lookup ON user_role_assignments(user_id, workspace_id);

-- Комментарии
COMMENT ON TABLE user_role_assignments IS 'Назначение ролей пользователям в workspace';
COMMENT ON COLUMN user_role_assignments.assigned_by IS 'ID пользователя, назначившего роль';
```

#### 4.1.4. Индивидуальные права (user_permissions)

```sql
-- Для случаев, когда нужно дать конкретное право конкретному пользователю
CREATE TABLE user_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permission_catalog(id) ON DELETE CASCADE,
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,                    -- опционально: временный доступ
    
    UNIQUE(user_id, workspace_id, permission_id)
);

CREATE INDEX idx_user_permissions_user ON user_permissions(user_id);
CREATE INDEX idx_user_permissions_workspace ON user_permissions(workspace_id);

COMMENT ON TABLE user_permissions IS 'Индивидуальные права для конкретных пользователей';
```

#### 4.1.5. Наследование ролей (role_inheritance)

```sql
CREATE TABLE role_inheritance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    child_role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    parent_role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Проверка: нельзя наследовать от самой себя
    CONSTRAINT check_no_self_inheritance CHECK (child_role_id != parent_role_id),
    -- Уникальность пары
    UNIQUE(workspace_id, child_role_id, parent_role_id)
);

CREATE INDEX idx_role_inheritance_child ON role_inheritance(child_role_id);
CREATE INDEX idx_role_inheritance_parent ON role_inheritance(parent_role_id);

COMMENT ON TABLE role_inheritance IS 'Иерархия наследования ролей';
```

### 4.2. Таблица Casbin (автоматическая)

```sql
-- Создается адаптером casbin, но для понимания структуры:
CREATE TABLE casbin_rule (
    id SERIAL PRIMARY KEY,
    ptype VARCHAR(100) NOT NULL,   -- 'p' (policy) или 'g' (grouping)
    v0 VARCHAR(100) NOT NULL,       -- субъект (user:uuid / role:name)
    v1 VARCHAR(100) NOT NULL,       -- домен (workspace_id) / родительская роль
    v2 VARCHAR(100),                 -- объект (crm:deal) / дочерняя роль
    v3 VARCHAR(100),                 -- действие (create)
    v4 VARCHAR(100),
    v5 VARCHAR(100)
);

-- Индексы для производительности
CREATE INDEX idx_casbin_rule_ptype ON casbin_rule(ptype);
CREATE INDEX idx_casbin_rule_v0 ON casbin_rule(v0);
CREATE INDEX idx_casbin_rule_v1 ON casbin_rule(v1);
```

### 4.3. Инициализация данных

#### 4.3.1. Наполнение каталога прав

```sql
-- CRM права
INSERT INTO permission_catalog (module_code, entity_type, action, name) VALUES
('crm', 'deal', 'create', 'Создание сделки'),
('crm', 'deal', 'read', 'Просмотр сделки'),
('crm', 'deal', 'update', 'Редактирование сделки'),
('crm', 'deal', 'delete', 'Удаление сделки'),
('crm', 'deal', 'move', 'Перемещение по этапам'),
('crm', 'contact', 'create', 'Создание контакта'),
('crm', 'contact', 'read', 'Просмотр контакта'),
('crm', 'contact', 'update', 'Редактирование контакта'),
('crm', 'contact', 'delete', 'Удаление контакта'),
('crm', 'company', 'create', 'Создание компании'),
('crm', 'company', 'read', 'Просмотр компании'),
('crm', 'company', 'update', 'Редактирование компании'),
('crm', 'company', 'delete', 'Удаление компании'),
('crm', 'pipeline', 'manage', 'Управление воронками'),
('crm', 'activity', 'create', 'Создание активности'),
('crm', 'activity', 'read', 'Просмотр активности'),
('crm', 'activity', 'update', 'Редактирование активности'),
('crm', 'activity', 'delete', 'Удаление активности'),
('crm', 'export', 'deals', 'Экспорт сделок');

-- Habits права
INSERT INTO permission_catalog (module_code, entity_type, action, name) VALUES
('habits', 'habit', 'create', 'Создание привычки'),
('habits', 'habit', 'read', 'Просмотр привычки'),
('habits', 'habit', 'update', 'Редактирование привычки'),
('habits', 'habit', 'delete', 'Удаление привычки'),
('habits', 'habit', 'complete', 'Отметка выполнения'),
('habits', 'journal', 'create', 'Создание записи'),
('habits', 'journal', 'read', 'Просмотр записи'),
('habits', 'journal', 'update', 'Редактирование записи'),
('habits', 'journal', 'delete', 'Удаление записи');

-- Projects права
INSERT INTO permission_catalog (module_code, entity_type, action, name) VALUES
('projects', 'project', 'create', 'Создание проекта'),
('projects', 'project', 'read', 'Просмотр проекта'),
('projects', 'project', 'update', 'Редактирование проекта'),
('projects', 'project', 'delete', 'Удаление проекта'),
('projects', 'entity', 'attach', 'Привязка сущности'),
('projects', 'entity', 'detach', 'Отвязка сущности');

-- Workspace управление
INSERT INTO permission_catalog (module_code, entity_type, action, name) VALUES
('workspace', 'member', 'invite', 'Приглашение участников'),
('workspace', 'member', 'remove', 'Удаление участников'),
('workspace', 'role', 'manage', 'Управление ролями'),
('workspace', 'module', 'manage', 'Управление модулями');
```

#### 4.3.2. Базовые политики для системных ролей

```sql
-- Эти политики будут загружены через Casbin API, но для понимания:

-- OWNER имеет все права (wildcard)
('p', 'role:OWNER', '*', '*', '*')

-- ADMIN имеет все права, кроме управления billing (если есть)
('p', 'role:ADMIN', '*', '*', '*')

-- MEMBER: базовые права на чтение/создание/редактирование
('p', 'role:MEMBER', '*', 'crm:deal', 'create')
('p', 'role:MEMBER', '*', 'crm:deal', 'read')
('p', 'role:MEMBER', '*', 'crm:deal', 'update')
('p', 'role:MEMBER', '*', 'crm:contact', 'read')
('p', 'role:MEMBER', '*', 'crm:company', 'read')
('p', 'role:MEMBER', '*', 'habits:habit', 'create')
('p', 'role:MEMBER', '*', 'habits:habit', 'read')
('p', 'role:MEMBER', '*', 'habits:habit', 'update')
('p', 'role:MEMBER', '*', 'habits:habit', 'complete')
('p', 'role:MEMBER', '*', 'projects:project', 'read')

-- GUEST: только чтение
('p', 'role:GUEST', '*', 'crm:deal', 'read')
('p', 'role:GUEST', '*', 'crm:contact', 'read')
('p', 'role:GUEST', '*', 'crm:company', 'read')
('p', 'role:GUEST', '*', 'habits:habit', 'read')
('p', 'role:GUEST', '*', 'projects:project', 'read')
```

---

## 5. Архитектура middleware

### 5.1. Полная цепочка middleware

```
[Request]
    │
    ▼
[AuthMiddleware (существующий)]
    ├─ Извлекает user_id из токена
    ├─ Кладёт в контекст: user_id
    │
    ▼
[WorkspaceMiddleware (НОВЫЙ)]
    ├─ Извлекает workspace_id из URL (/workspaces/:workspaceId)
    ├─ Проверяет user_workspaces (членство в workspace)
    ├─ Кладёт в контекст: workspace_id
    │
    ▼
[ModuleLicenseMiddleware (НОВЫЙ)]
    ├─ Определяет модуль по пути (/crm/, /habits/, /projects/)
    ├─ Проверяет workspace_modules (включен ли модуль)
    ├─ Проверяет user_module_licenses (если модуль не core)
    ├─ Кладёт в контекст: module_code
    │
    ▼
[PermissionMiddleware (НОВЫЙ)]
    ├─ Маппит endpoint → (object, action)
    ├─ Получает все роли пользователя из user_role_assignments
    ├─ Для каждой роли: sub = "role:" + roleName
    ├─ Также проверяет индивидуальные права (user_permissions)
    ├─ Спрашивает Casbin: Enforce(sub, workspace_id, object, action)
    │
    ▼
[Business Logic Handler]
```

### 5.2. Детальная реализация каждого middleware

#### 5.2.1. AuthMiddleware (существующий, с доработкой)

```go
package middleware

import (
    "context"
    "net/http"
    "strings"

    "backend/pkg/auth/token"
)

type contextKey string

const (
    UserIDKey      contextKey = "user_id"
    GlobalRoleKey  contextKey = "global_role"  // из токена, не для workspace!
    WorkspaceIDKey contextKey = "workspace_id"
    WorkspaceRoleKey contextKey = "workspace_role" // из user_workspaces
    ModuleCodeKey  contextKey = "module_code"
)

func AuthMiddleware(tokenGen *token.Generator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Получение токена
            var tokenString string
            
            // Приоритет: кука > заголовок
            if cookie, err := r.Cookie("access_token"); err == nil {
                tokenString = cookie.Value
            } else {
                authHeader := r.Header.Get("Authorization")
                if authHeader != "" {
                    tokenString = strings.TrimPrefix(authHeader, "Bearer ")
                }
            }

            // Если нет токена - пропускаем как анонимного пользователя
            if tokenString == "" {
                next.ServeHTTP(w, r)
                return
            }

            // 2. Валидация токена
            claims, err := tokenGen.Validate(tokenString)
            if err != nil {
                // Невалидный токен - тоже аноним
                next.ServeHTTP(w, r)
                return
            }

            // 3. Сохраняем в контекст
            ctx := r.Context()
            ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
            ctx = context.WithValue(ctx, GlobalRoleKey, claims.Role) // может быть пустым

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Helper functions
func GetUserID(ctx context.Context) (string, bool) {
    id, ok := ctx.Value(UserIDKey).(string)
    return id, ok
}

func GetWorkspaceID(ctx context.Context) (string, bool) {
    id, ok := ctx.Value(WorkspaceIDKey).(string)
    return id, ok
}
```

#### 5.2.2. WorkspaceMiddleware (новый)

```go
package middleware

import (
    "context"
    "net/http"
    "strings"

    "backend/internal/repository"
)

func WorkspaceMiddleware(userWorkspaceRepo *repository.UserWorkspaceRepository) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Получаем user_id (может быть пустым для неавторизованных)
            userID, _ := GetUserID(r.Context())
            
            // 2. Извлекаем workspace_id из URL
            workspaceID := extractWorkspaceID(r.URL.Path)
            if workspaceID == "" {
                // Если в URL нет workspace_id, пропускаем (публичный эндпоинт)
                next.ServeHTTP(w, r)
                return
            }

            // 3. Если пользователь авторизован, проверяем членство в workspace
            if userID != "" {
                exists, err := userWorkspaceRepo.Exists(userID, workspaceID)
                if err != nil || !exists {
                    http.Error(w, "user not in workspace", http.StatusForbidden)
                    return
                }

                // Получаем роль пользователя в этом workspace (для обратной совместимости)
                userWorkspace, _ := userWorkspaceRepo.Find(userID, workspaceID)
                if userWorkspace != nil {
                    ctx := context.WithValue(r.Context(), WorkspaceRoleKey, userWorkspace.Role)
                    r = r.WithContext(ctx)
                }
            }

            // 4. Сохраняем workspace_id в контекст
            ctx := context.WithValue(r.Context(), WorkspaceIDKey, workspaceID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func extractWorkspaceID(path string) string {
    // Паттерн: /api/v1/workspaces/{workspaceId}/...
    parts := strings.Split(path, "/")
    for i, part := range parts {
        if part == "workspaces" && i+1 < len(parts) {
            return parts[i+1]
        }
    }
    return ""
}
```

#### 5.2.3. ModuleLicenseMiddleware (новый)

```go
package middleware

import (
    "context"
    "net/http"
    "strings"

    "backend/internal/service"
)

func ModuleLicenseMiddleware(moduleService *service.ModuleService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Получаем данные из контекста
            userID, _ := GetUserID(r.Context())
            workspaceID, ok := GetWorkspaceID(r.Context())
            if !ok {
                // Нет workspace - пропускаем (публичный эндпоинт)
                next.ServeHTTP(w, r)
                return
            }

            // 2. Определяем модуль по пути
            moduleCode := detectModule(r.URL.Path)
            if moduleCode == "" {
                // Не модульный эндпоинт - пропускаем
                next.ServeHTTP(w, r)
                return
            }

            // 3. Проверяем, включен ли модуль в workspace
            enabled, err := moduleService.IsModuleEnabled(workspaceID, moduleCode)
            if err != nil || !enabled {
                http.Error(w, "module not enabled in this workspace", http.StatusForbidden)
                return
            }

            // 4. Если пользователь авторизован, проверяем лицензию
            if userID != "" {
                // Проверяем, является ли модуль core (всегда доступен)
                isCore, err := moduleService.IsCoreModule(moduleCode)
                if err != nil {
                    http.Error(w, "internal error", http.StatusInternalServerError)
                    return
                }

                if !isCore {
                    hasLicense, err := moduleService.UserHasLicense(userID, workspaceID, moduleCode)
                    if err != nil || !hasLicense {
                        http.Error(w, "no license for this module", http.StatusForbidden)
                        return
                    }
                }
            }

            // 5. Сохраняем модуль в контекст
            ctx := context.WithValue(r.Context(), ModuleCodeKey, moduleCode)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func detectModule(path string) string {
    switch {
    case strings.Contains(path, "/crm/"):
        return "crm"
    case strings.Contains(path, "/habits/"):
        return "habits"
    case strings.Contains(path, "/projects/"):
        return "projects"
    case strings.Contains(path, "/notes/"):
        return "notes"
    case strings.Contains(path, "/journal/"):
        return "habits" // journal часть habits
    default:
        return ""
    }
}
```

#### 5.2.4. PermissionMiddleware (новый, с Casbin)

```go
package middleware

import (
    "net/http"
    "strings"

    "github.com/casbin/casbin/v2"
)

type EndpointMapper struct {
    // map[method][pathPattern]Permission
    rules map[string]map[string]Permission
}

type Permission struct {
    Object string
    Action string
}

func NewEndpointMapper() *EndpointMapper {
    m := &EndpointMapper{
        rules: make(map[string]map[string]Permission),
    }

    // Регистрация всех эндпоинтов из вывода go run
    m.register("GET", "/api/v1/workspaces/:workspaceId/deals", "crm:deal", "read")
    m.register("POST", "/api/v1/workspaces/:workspaceId/deals", "crm:deal", "create")
    m.register("GET", "/api/v1/workspaces/:workspaceId/deals/:id", "crm:deal", "read")
    m.register("PUT", "/api/v1/workspaces/:workspaceId/deals/:id", "crm:deal", "update")
    m.register("DELETE", "/api/v1/workspaces/:workspaceId/deals/:id", "crm:deal", "delete")
    
    m.register("GET", "/api/v1/workspaces/:workspaceId/contacts", "crm:contact", "read")
    m.register("POST", "/api/v1/workspaces/:workspaceId/contacts", "crm:contact", "create")
    m.register("PUT", "/api/v1/workspaces/:workspaceId/contacts/:id", "crm:contact", "update")
    m.register("DELETE", "/api/v1/workspaces/:workspaceId/contacts/:id", "crm:contact", "delete")
    
    m.register("POST", "/api/v1/workspaces/:workspaceId/pipelines", "crm:pipeline", "manage")
    m.register("PUT", "/api/v1/workspaces/:workspaceId/pipelines/:pipelineId", "crm:pipeline", "manage")
    m.register("DELETE", "/api/v1/workspaces/:workspaceId/pipelines/:pipelineId", "crm:pipeline", "manage")
    
    // ... и так далее для всех эндпоинтов

    return m
}

func (m *EndpointMapper) register(method, pattern, obj, act string) {
    if _, ok := m.rules[method]; !ok {
        m.rules[method] = make(map[string]Permission)
    }
    m.rules[method][pattern] = Permission{Object: obj, Action: act}
}

func (m *EndpointMapper) Map(method, path string) (string, string) {
    if methods, ok := m.rules[method]; ok {
        // Пробуем точное совпадение
        if perm, ok := methods[path]; ok {
            return perm.Object, perm.Action
        }
        
        // Пробуем по шаблону (заменяем UUID на :id)
        pattern := normalizePath(path)
        if perm, ok := methods[pattern]; ok {
            return perm.Object, perm.Action
        }
    }
    return "", ""
}

func normalizePath(path string) string {
    parts := strings.Split(path, "/")
    for i, part := range parts {
        if isUUID(part) {
            parts[i] = ":id"
        }
    }
    return strings.Join(parts, "/")
}

func isUUID(s string) bool {
    // Простая проверка формата UUID
    if len(s) != 36 {
        return false
    }
    // Можно добавить regex, но для производительности - так
    return true
}

type PermissionService interface {
    GetUserRoles(userID, workspaceID string) ([]string, error)
    GetUserIndividualPermissions(userID, workspaceID string) ([]string, error)
}

func PermissionMiddleware(enforcer *casbin.Enforcer, mapper *EndpointMapper, permService PermissionService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Получаем данные из контекста
            userID, _ := GetUserID(r.Context())
            workspaceID, ok := GetWorkspaceID(r.Context())
            if !ok {
                // Нет workspace - пропускаем (публичный эндпоинт)
                next.ServeHTTP(w, r)
                return
            }

            // 2. Определяем требуемое право
            obj, act := mapper.Map(r.Method, r.URL.Path)
            if obj == "" || act == "" {
                // Эндпоинт не требует прав
                next.ServeHTTP(w, r)
                return
            }

            // 3. Если пользователь не авторизован - доступ запрещен
            if userID == "" {
                http.Error(w, "authentication required", http.StatusUnauthorized)
                return
            }

            // 4. Получаем все роли пользователя в этом workspace
            roles, err := permService.GetUserRoles(userID, workspaceID)
            if err != nil {
                http.Error(w, "internal error", http.StatusInternalServerError)
                return
            }

            // 5. Проверяем каждую роль через Casbin
            allowed := false
            for _, role := range roles {
                sub := "role:" + role
                allowed, err = enforcer.Enforce(sub, workspaceID, obj, act)
                if err != nil {
                    continue
                }
                if allowed {
                    break
                }
            }

            // 6. Если не разрешено через роли, проверяем индивидуальные права
            if !allowed {
                individualPerms, err := permService.GetUserIndividualPermissions(userID, workspaceID)
                if err == nil {
                    for _, perm := range individualPerms {
                        if perm == obj+":"+act {
                            allowed = true
                            break
                        }
                    }
                }
            }

            if !allowed {
                http.Error(w, "access denied", http.StatusForbidden)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### 5.3. Service для работы с правами

```go
package service

import (
    "backend/internal/model"
    "backend/internal/repository"
    "fmt"
)

type PermissionService struct {
    userRoleAssignmentsRepo *repository.UserRoleAssignmentsRepository
    userPermissionsRepo     *repository.UserPermissionsRepository
    workspaceRolesRepo      *repository.WorkspaceRolesRepository
    enforcer                *casbin.Enforcer
}

func NewPermissionService(
    uraRepo *repository.UserRoleAssignmentsRepository,
    upRepo *repository.UserPermissionsRepository,
    wrRepo *repository.WorkspaceRolesRepository,
    enforcer *casbin.Enforcer,
) *PermissionService {
    return &PermissionService{
        userRoleAssignmentsRepo: uraRepo,
        userPermissionsRepo:     upRepo,
        workspaceRolesRepo:      wrRepo,
        enforcer:                enforcer,
    }
}

// GetUserRoles возвращает все роли пользователя в workspace
func (s *PermissionService) GetUserRoles(userID, workspaceID string) ([]string, error) {
    assignments, err := s.userRoleAssignmentsRepo.FindByUserAndWorkspace(userID, workspaceID)
    if err != nil {
        return nil, err
    }

    roles := make([]string, 0, len(assignments))
    for _, a := range assignments {
        role, err := s.workspaceRolesRepo.FindByID(a.RoleID)
        if err != nil {
            continue
        }
        roles = append(roles, role.Name)
    }
    return roles, nil
}

// GetUserIndividualPermissions возвращает индивидуальные права пользователя
func (s *PermissionService) GetUserIndividualPermissions(userID, workspaceID string) ([]string, error) {
    perms, err := s.userPermissionsRepo.FindByUserAndWorkspace(userID, workspaceID)
    if err != nil {
        return nil, err
    }

    result := make([]string, 0, len(perms))
    for _, p := range perms {
        result = append(result, fmt.Sprintf("%s:%s", p.Permission.Object, p.Permission.Action))
    }
    return result, nil
}

// AssignRole назначает роль пользователю
func (s *PermissionService) AssignRole(userID, roleID, workspaceID, assignedBy string) error {
    // 1. Сохраняем в БД
    assignment := &model.UserRoleAssignment{
        UserID:      userID,
        RoleID:      roleID,
        WorkspaceID: workspaceID,
        AssignedBy:  assignedBy,
    }
    
    if err := s.userRoleAssignmentsRepo.Create(assignment); err != nil {
        return err
    }

    // 2. Получаем имя роли
    role, err := s.workspaceRolesRepo.FindByID(roleID)
    if err != nil {
        return err
    }

    // 3. Добавляем в Casbin (grouping policy)
    // g, user:uuid, role:name, workspace_id
    _, err = s.enforcer.AddGroupingPolicy(
        fmt.Sprintf("user:%s", userID),
        fmt.Sprintf("role:%s", role.Name),
        workspaceID,
    )
    
    return err
}

// RemoveRole удаляет роль у пользователя
func (s *PermissionService) RemoveRole(userID, roleID, workspaceID string) error {
    // 1. Удаляем из БД
    if err := s.userRoleAssignmentsRepo.Delete(userID, roleID, workspaceID); err != nil {
        return err
    }

    // 2. Получаем имя роли
    role, err := s.workspaceRolesRepo.FindByID(roleID)
    if err != nil {
        return err
    }

    // 3. Удаляем из Casbin
    _, err = s.enforcer.RemoveGroupingPolicy(
        fmt.Sprintf("user:%s", userID),
        fmt.Sprintf("role:%s", role.Name),
        workspaceID,
    )
    
    return err
}

// CreateRole создает новую роль с правами
func (s *PermissionService) CreateRole(workspaceID, name string, permissions []string, createdBy string) (*model.WorkspaceRole, error) {
    // 1. Создаем роль в БД
    role := &model.WorkspaceRole{
        WorkspaceID: workspaceID,
        Name:        name,
        IsSystem:    false,
    }
    
    if err := s.workspaceRolesRepo.Create(role); err != nil {
        return nil, err
    }

    // 2. Добавляем политики для каждого права
    for _, perm := range permissions {
        // Ожидаем формат "obj:act", например "crm:deal:create"
        parts := strings.Split(perm, ":")
        if len(parts) != 2 {
            continue
        }
        obj, act := parts[0], parts[1]
        
        _, err := s.enforcer.AddPolicy(
            fmt.Sprintf("role:%s", name),
            workspaceID,
            obj,
            act,
        )
        if err != nil {
            // Логируем ошибку, но продолжаем
        }
    }

    return role, nil
}
```

---

## 6. Интеграция с Casbin

### 6.1. Инициализация Casbin

```go
package casbin

import (
    "github.com/casbin/casbin/v2"
    "github.com/casbin/casbin/v2/model"
    gormadapter "github.com/casbin/gorm-adapter/v3"
    "gorm.io/gorm"
)

func InitCasbin(db *gorm.DB) (*casbin.Enforcer, error) {
    // 1. Создаем адаптер для PostgreSQL
    adapter, err := gormadapter.NewAdapterByDB(db)
    if err != nil {
        return nil, err
    }

    // 2. Определяем модель с поддержкой доменов (workspace)
    modelText := `
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
    `

    m, err := model.NewModelFromString(modelText)
    if err != nil {
        return nil, err
    }

    // 3. Создаем enforcer
    e, err := casbin.NewEnforcer(m, adapter)
    if err != nil {
        return nil, err
    }

    // 4. Включаем кэш
    e.EnableCache(true)

    // 5. Загружаем политики
    if err := e.LoadPolicy(); err != nil {
        return nil, err
    }

    return e, nil
}
```

### 6.2. Синхронизация существующих данных

```go
package migration

import (
    "backend/internal/repository"
    "fmt"

    "github.com/casbin/casbin/v2"
)

type PolicyMigrator struct {
    userWorkspaceRepo *repository.UserWorkspaceRepository
    workspaceRolesRepo *repository.WorkspaceRolesRepository
    enforcer          *casbin.Enforcer
}

func NewPolicyMigrator(
    uwRepo *repository.UserWorkspaceRepository,
    wrRepo *repository.WorkspaceRolesRepository,
    enforcer *casbin.Enforcer,
) *PolicyMigrator {
    return &PolicyMigrator{
        userWorkspaceRepo: uwRepo,
        workspaceRolesRepo: wrRepo,
        enforcer:          enforcer,
    }
}

// MigrateSystemRoles создает базовые политики для системных ролей
func (m *PolicyMigrator) MigrateSystemRoles() error {
    // Для ADMIN - все права
    _, err := m.enforcer.AddPolicy("role:ADMIN", "*", "*", "*")
    if err != nil {
        return err
    }

    // Для OWNER - все права (может быть то же самое)
    _, err = m.enforcer.AddPolicy("role:OWNER", "*", "*", "*")
    if err != nil {
        return err
    }

    // Для MEMBER - базовые права (будут расширяться)
    memberPolicies := [][]string{
        {"role:MEMBER", "*", "crm:deal", "create"},
        {"role:MEMBER", "*", "crm:deal", "read"},
        {"role:MEMBER", "*", "crm:deal", "update"},
        {"role:MEMBER", "*", "crm:contact", "read"},
        {"role:MEMBER", "*", "crm:company", "read"},
        {"role:MEMBER", "*", "habits:habit", "create"},
        {"role:MEMBER", "*", "habits:habit", "read"},
        {"role:MEMBER", "*", "habits:habit", "update"},
        {"role:MEMBER", "*", "habits:habit", "complete"},
    }

    for _, p := range memberPolicies {
        _, err = m.enforcer.AddPolicy(p[0], p[1], p[2], p[3])
        if err != nil {
            return err
        }
    }

    // Для GUEST - только чтение
    guestPolicies := [][]string{
        {"role:GUEST", "*", "crm:deal", "read"},
        {"role:GUEST", "*", "crm:contact", "read"},
        {"role:GUEST", "*", "crm:company", "read"},
        {"role:GUEST", "*", "habits:habit", "read"},
        {"role:GUEST", "*", "projects:project", "read"},
    }

    for _, p := range guestPolicies {
        _, err = m.enforcer.AddPolicy(p[0], p[1], p[2], p[3])
        if err != nil {
            return err
        }
    }

    return nil
}

// MigrateExistingAssignments переносит существующие назначения из user_workspaces в Casbin
func (m *PolicyMigrator) MigrateExistingAssignments() error {
    // Получаем все записи из user_workspaces
    assignments, err := m.userWorkspaceRepo.FindAll()
    if err != nil {
        return err
    }

    for _, a := range assignments {
        // Добавляем в Casbin: g, user:uuid, role:name, workspace_id
        _, err := m.enforcer.AddGroupingPolicy(
            fmt.Sprintf("user:%s", a.UserID),
            fmt.Sprintf("role:%s", a.Role), // OWNER, ADMIN, MEMBER, GUEST
            a.WorkspaceID,
        )
        if err != nil {
            // Логируем ошибку, но продолжаем
            fmt.Printf("Error adding policy for user %s: %v\n", a.UserID, err)
        }
    }

    return nil
}
```

---

## 7. API для управления ролями и правами

### 7.1. Получение каталога прав (для UI)

```go
// GET /api/v1/workspaces/:workspaceId/permissions/catalog
func (h *PermissionHandler) GetCatalog(c *gin.Context) {
    catalog, err := h.permissionService.GetCatalog()
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // Группируем по модулям для удобства фронтенда
    result := make(map[string]map[string][]PermissionInfo)
    for _, p := range catalog {
        if _, ok := result[p.ModuleCode]; !ok {
            result[p.ModuleCode] = make(map[string][]PermissionInfo)
        }
        if _, ok := result[p.ModuleCode][p.EntityType]; !ok {
            result[p.ModuleCode][p.EntityType] = []PermissionInfo{}
        }
        result[p.ModuleCode][p.EntityType] = append(
            result[p.ModuleCode][p.EntityType],
            PermissionInfo{
                Action: p.Action,
                Name:   p.Name,
            },
        )
    }

    c.JSON(200, result)
}
```

### 7.2. Управление ролями

```go
// GET /api/v1/workspaces/:workspaceId/roles
func (h *PermissionHandler) ListRoles(c *gin.Context) {
    workspaceID := c.GetString("workspace_id")
    
    roles, err := h.permissionService.ListRoles(workspaceID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // Для каждой роли получаем назначенные права
    result := make([]RoleWithPermissions, 0, len(roles))
    for _, role := range roles {
        permissions, _ := h.permissionService.GetRolePermissions(role.ID)
        result = append(result, RoleWithPermissions{
            Role:        role,
            Permissions: permissions,
        })
    }

    c.JSON(200, result)
}

// POST /api/v1/workspaces/:workspaceId/roles
type CreateRoleRequest struct {
    Name        string   `json:"name" binding:"required"`
    Description string   `json:"description"`
    Permissions []string `json:"permissions"` // ["crm:deal:create", "crm:deal:read"]
}

func (h *PermissionHandler) CreateRole(c *gin.Context) {
    var req CreateRoleRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    workspaceID := c.GetString("workspace_id")
    createdBy := c.GetString("user_id")

    role, err := h.permissionService.CreateRole(workspaceID, req.Name, req.Permissions, createdBy)
    if err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    c.JSON(201, role)
}

// PUT /api/v1/workspaces/:workspaceId/roles/:roleId
type UpdateRoleRequest struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Permissions []string `json:"permissions"`
}

func (h *PermissionHandler) UpdateRole(c *gin.Context) {
    roleID := c.Param("roleId")
    
    var req UpdateRoleRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // Проверяем, что роль не системная
    role, err := h.permissionService.GetRole(roleID)
    if err != nil {
        c.JSON(404, gin.H{"error": "role not found"})
        return
    }
    if role.IsSystem {
        c.JSON(400, gin.H{"error": "cannot modify system role"})
        return
    }

    // Обновляем роль и права
    updatedRole, err := h.permissionService.UpdateRole(roleID, req.Name, req.Description, req.Permissions)
    if err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, updatedRole)
}

// DELETE /api/v1/workspaces/:workspaceId/roles/:roleId
func (h *PermissionHandler) DeleteRole(c *gin.Context) {
    roleID := c.Param("roleId")

    // Проверяем, что роль не системная
    role, err := h.permissionService.GetRole(roleID)
    if err != nil {
        c.JSON(404, gin.H{"error": "role not found"})
        return
    }
    if role.IsSystem {
        c.JSON(400, gin.H{"error": "cannot delete system role"})
        return
    }

    // Проверяем, что роль не назначена пользователям
    hasAssignments, err := h.permissionService.RoleHasAssignments(roleID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    if hasAssignments {
        c.JSON(400, gin.H{"error": "cannot delete role with assigned users"})
        return
    }

    if err := h.permissionService.DeleteRole(roleID); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"status": "ok"})
}
```

### 7.3. Назначение ролей пользователям

```go
// GET /api/v1/workspaces/:workspaceId/users/:userId/roles
func (h *PermissionHandler) GetUserRoles(c *gin.Context) {
    userID := c.Param("userId")
    workspaceID := c.GetString("workspace_id")

    roles, err := h.permissionService.GetUserRoles(userID, workspaceID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"roles": roles})
}

// POST /api/v1/workspaces/:workspaceId/users/:userId/roles/:roleId
func (h *PermissionHandler) AssignRoleToUser(c *gin.Context) {
    userID := c.Param("userId")
    roleID := c.Param("roleId")
    workspaceID := c.GetString("workspace_id")
    assignedBy := c.GetString("user_id")

    if err := h.permissionService.AssignRole(userID, roleID, workspaceID, assignedBy); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"status": "ok"})
}

// DELETE /api/v1/workspaces/:workspaceId/users/:userId/roles/:roleId
func (h *PermissionHandler) RemoveRoleFromUser(c *gin.Context) {
    userID := c.Param("userId")
    roleID := c.Param("roleId")
    workspaceID := c.GetString("workspace_id")

    if err := h.permissionService.RemoveRole(userID, roleID, workspaceID); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"status": "ok"})
}
```

### 7.4. Получение прав текущего пользователя (для UI)

```go
// GET /api/v1/me/permissions?workspaceId=123
func (h *PermissionHandler) GetMyPermissions(c *gin.Context) {
    userID := c.GetString("user_id")
    if userID == "" {
        c.JSON(401, gin.H{"error": "unauthorized"})
        return
    }

    workspaceID := c.Query("workspaceId")
    if workspaceID == "" {
        c.JSON(400, gin.H{"error": "workspaceId required"})
        return
    }

    // Получаем все права пользователя через Casbin
    // Это сложный запрос, поэтому используем специальный метод
    permissions, err := h.permissionService.GetEffectivePermissions(userID, workspaceID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // Также возвращаем роли для информации
    roles, _ := h.permissionService.GetUserRoles(userID, workspaceID)

    c.JSON(200, gin.H{
        "permissions": permissions,
        "roles":       roles,
    })
}
```

### 7.5. Наследование ролей

```go
// POST /api/v1/workspaces/:workspaceId/roles/:roleId/inherit/:parentRoleId
func (h *PermissionHandler) AddInheritance(c *gin.Context) {
    workspaceID := c.GetString("workspace_id")
    childRoleID := c.Param("roleId")
    parentRoleID := c.Param("parentRoleId")

    // Проверяем, что обе роли существуют в этом workspace
    child, err := h.permissionService.GetRole(childRoleID)
    if err != nil || child.WorkspaceID != workspaceID {
        c.JSON(404, gin.H{"error": "child role not found"})
        return
    }

    parent, err := h.permissionService.GetRole(parentRoleID)
    if err != nil || parent.WorkspaceID != workspaceID {
        c.JSON(404, gin.H{"error": "parent role not found"})
        return
    }

    // Добавляем наследование в Casbin
    // g, role:child, role:parent, workspace_id
    _, err = h.enforcer.AddGroupingPolicy(
        fmt.Sprintf("role:%s", child.Name),
        fmt.Sprintf("role:%s", parent.Name),
        workspaceID,
    )
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // Сохраняем в БД для аудита
    inheritance := &model.RoleInheritance{
        WorkspaceID:  workspaceID,
        ChildRoleID:  childRoleID,
        ParentRoleID: parentRoleID,
    }
    h.roleInheritanceRepo.Create(inheritance)

    c.JSON(200, gin.H{"status": "ok"})
}
```

---

## 8. Модификация существующих хендлеров

### 8.1. Упрощение хендлеров (убираем проверки)

**БЫЛО:**
```go
func (h *DealHandler) UpdateDeal(c *gin.Context) {
    userID := c.GetString("user_id")
    workspaceID := c.Param("workspaceId")
    
    // Проверка членства в workspace
    userWorkspace, err := h.userWorkspaceRepo.Find(userID, workspaceID)
    if err != nil {
        c.JSON(403, gin.H{"error": "access denied"})
        return
    }
    
    // ХАРДКОД: проверка роли
    if userWorkspace.Role != "ADMIN" && userWorkspace.Role != "OWNER" {
        c.JSON(403, gin.H{"error": "insufficient permissions"})
        return
    }
    
    // Бизнес-логика...
    deal, err := h.dealRepo.FindByID(dealID, workspaceID)
    if err != nil {
        c.JSON(404, gin.H{"error": "deal not found"})
        return
    }
    
    // Обновление...
    c.JSON(200, deal)
}
```

**СТАЛО:**
```go
func (h *DealHandler) UpdateDeal(c *gin.Context) {
    // Все проверки уже выполнены в middleware!
    // Если мы здесь - значит:
    // 1. Пользователь авторизован
    // 2. Пользователь состоит в workspace
    // 3. Модуль CRM включен
    // 4. У пользователя есть право "crm:deal:update"
    
    workspaceID := c.GetString("workspace_id")
    dealID := c.Param("id")
    
    // Бизнес-логика...
    deal, err := h.dealRepo.FindByID(dealID, workspaceID)
    if err != nil {
        c.JSON(404, gin.H{"error": "deal not found"})
        return
    }
    
    var input UpdateDealInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Обновление...
    updatedDeal, err := h.dealRepo.Update(dealID, workspaceID, input)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, updatedDeal)
}
```

### 8.2. Постепенное внедрение

```go
// Временная обертка для обратной совместимости
func WithFallbackPermission(next gin.HandlerFunc) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Пробуем новый middleware (он уже в цепочке)
        // Если новый пропустил - ok
        // Если новый за
