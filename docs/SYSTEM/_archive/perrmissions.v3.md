# Системный анализ: Внедрение гибкой системы ролей и прав доступа
## Полный анализ с учетом существующей структуры БД

**Версия 5.0** | Март 2026

---

## Содержание

1. **Введение и цели**
2. **Детальный анализ существующей структуры БД**
3. **Анализ текущей модели авторизации**
4. **Что можно использовать из существующего**
5. **Необходимые изменения в БД**
6. **Интеграция с существующими таблицами**
7. **План миграции данных**
8. **Рекомендации по использованию существующих индексов**
9. **Заключение по структуре БД**

---

## 1. Введение и цели

### 1.1. Контекст

На основе предоставленных SQL-миграций мы имеем хорошо структурированную ERP-систему с:
- Мультитенантностью (workspace_id во всех таблицах)
- Модульной архитектурой (таблицы modules, workspace_modules)
- Базовой ролевой моделью (user_workspaces.role)
- Системой лицензирования (user_module_licenses)

### 1.2. Цель анализа

Проанализировать существующую структуру БД и определить, как наиболее эффективно внедрить гибкую систему прав доступа, максимально используя существующие таблицы и минимизируя изменения.

---

## 2. Детальный анализ существующей структуры БД

### 2.1. Анализ ключевых таблиц авторизации

#### **Таблица `users`** (000002)
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(100),
    role VARCHAR(20) NOT NULL DEFAULT 'USER',  -- Глобальная роль в системе
    avatar_url TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Что есть:**
- `id` - UUID, что идеально для связей
- `role` - глобальная роль пользователя (USER/ADMIN и т.д.) - **НЕ ИСПОЛЬЗУЕТСЯ для workspace-прав**

**Что можно использовать:**
- `id` как идентификатор пользователя
- `status` для блокировки пользователей

#### **Таблица `workspaces`** (000005)
```sql
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(50) DEFAULT '#3B82F6',
    owner_id UUID NOT NULL,  -- Владелец workspace
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Что есть:**
- `id` - UUID workspace
- `owner_id` - владелец (всегда имеет полный доступ)

**Что можно использовать:**
- `owner_id` для определения особого статуса (OWNER)

#### **Таблица `user_workspaces`** (000006) - **КЛЮЧЕВАЯ**
```sql
CREATE TABLE user_workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'MEMBER',  -- OWNER, ADMIN, MEMBER, GUEST
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, workspace_id)
);
```

**Что есть:**
- Связь пользователь-workspace (уникальная)
- `role` - текущая жесткая роль
- Индексы на `user_id` и `workspace_id`

**Проблема:**
- Только одна роль на пользователя в workspace
- Фиксированный набор ролей

**Что можно использовать:**
- Саму связь для проверки членства в workspace
- Индексы для быстрого поиска

#### **Таблица `modules`** (000012)
```sql
CREATE TABLE modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE,  -- habits, crm, projects, notes
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_core BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Что есть:**
- Справочник всех модулей
- `code` - уникальный код модуля
- `is_core` - всегда доступен в workspace

**Что можно использовать:**
- `code` для формирования прав (например, `crm:deal:create`)
- `is_core` для пропуска проверки лицензий

#### **Таблица `workspace_modules`** (000012)
```sql
CREATE TABLE workspace_modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'trial', 'disabled')),
    activated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,
    settings JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, module_id)
);
```

**Что есть:**
- Какие модули активны в workspace
- Статус модуля

**Что можно использовать:**
- Проверка доступа к модулю перед проверкой прав

#### **Таблица `user_module_licenses`** (000013)
```sql
CREATE TABLE user_module_licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    scope VARCHAR(20) NOT NULL CHECK (scope IN ('all_workspaces', 'single_workspace')),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'cancelled')),
    source VARCHAR(20) NOT NULL DEFAULT 'purchase',
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Что есть:**
- Лицензии пользователей на модули
- Два типа: глобальные и для конкретного workspace

**Что можно использовать:**
- Проверка лицензии перед проверкой прав (для не-core модулей)

### 2.2. Существующие индексы для авторизации

```sql
-- Быстрый поиск пользователя в workspace
idx_user_workspaces_user_id
idx_user_workspaces_workspace_id

-- Проверка статуса модуля в workspace
idx_workspace_modules_workspace_id
idx_workspace_modules_status

-- Проверка лицензий пользователя
idx_user_module_licenses_user_id
idx_user_module_licenses_status

-- Индексы workspace_id во всех таблицах для изоляции
idx_habits_workspace_id
idx_notes_workspace_id
idx_crm_contacts_workspace
idx_crm_deals_workspace
-- и т.д.
```

---

## 3. Анализ текущей модели авторизации

### 3.1. Трехуровневая проверка доступа

```
Уровень 1: Доступ к workspace
────────────────────────────────
Таблица: user_workspaces
Проверка: EXISTS (user_id, workspace_id)
Результат: пользователь имеет доступ к workspace

Уровень 2: Доступ к модулю
────────────────────────────────
Таблица: workspace_modules
Проверка: status = 'active' для (workspace_id, module_id)
Таблица: user_module_licenses (для не-core модулей)
Проверка: EXISTS активной лицензии

Уровень 3: Права на действия
────────────────────────────────
Текущее: user_workspaces.role (ADMIN/MEMBER и т.д.)
Новое: Система гранулярных прав (цель внедрения)
```

### 3.2. Ограничения текущей модели

| Аспект | Текущее состояние | Проблема |
|--------|-------------------|----------|
| **Количество ролей** | 4 (OWNER, ADMIN, MEMBER, GUEST) | Нельзя создать "Sales Manager" |
| **Ролей на пользователя** | 1 | Нельзя быть и "Менеджером" и "Экспортёром" |
| **Гранулярность** | Нет | ADMIN имеет ВСЕ права или ничего |
| **Индивидуальные права** | Нет | Нельзя дать доступ к одной сделке |
| **Наследование** | Нет | Нельзя иерархию ролей |

---

## 4. Что можно использовать из существующего

### 4.1. Таблицы, которые остаются без изменений

| Таблица | Использование |
|---------|---------------|
| `users` | Хранение пользователей |
| `workspaces` | Хранение workspace |
| `modules` | Справочник модулей |
| `workspace_modules` | Проверка доступности модуля |
| `user_module_licenses` | Проверка лицензий |
| Все таблицы с данными | Все содержат `workspace_id` для изоляции |

### 4.2. Таблицы, которые требуют модификации

| Таблица | Изменения |
|---------|-----------|
| `user_workspaces` | Оставить для обратной совместимости, но перестать использовать для прав (только для проверки членства) |

### 4.3. Что нужно добавить

```sql
-- 1. Каталог прав (словарь всех возможных действий)
permission_catalog

-- 2. Кастомные роли
workspace_roles

-- 3. Назначение ролей пользователям
user_role_assignments

-- 4. Индивидуальные права
user_permissions

-- 5. Наследование ролей
role_inheritance

-- 6. Таблица Casbin (автоматически)
casbin_rule
```

---

## 5. Необходимые изменения в БД

### 5.1. Полная спецификация новых таблиц

#### **Таблица `permission_catalog`**
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

-- Индексы для быстрого поиска
CREATE INDEX idx_permission_catalog_module ON permission_catalog(module_code);
CREATE INDEX idx_permission_catalog_entity ON permission_catalog(entity_type);

-- Комментарии
COMMENT ON TABLE permission_catalog IS 'Каталог всех возможных прав в системе';
COMMENT ON COLUMN permission_catalog.module_code IS 'Код модуля (crm, habits, projects) из таблицы modules';
COMMENT ON COLUMN permission_catalog.entity_type IS 'Тип сущности (deal, contact, habit)';
COMMENT ON COLUMN permission_catalog.action IS 'Действие (create, read, update, delete)';
COMMENT ON COLUMN permission_catalog.name IS 'Человекочитаемое название для UI';
```

#### **Таблица `workspace_roles`**
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

#### **Таблица `user_role_assignments`**
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
```

#### **Таблица `user_permissions`**
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
CREATE INDEX idx_user_permissions_lookup ON user_permissions(user_id, workspace_id);
```

#### **Таблица `role_inheritance`**
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
```

#### **Таблица `casbin_rule`** (создается адаптером)
```sql
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

### 5.2. Модификация существующих таблиц

**Таблица `user_workspaces` остается, но меняется ее назначение:**
- Ранее: хранила роль и использовалась для проверки прав
- Теперь: только для проверки членства в workspace (обратная совместимость)

**Важно:** Не удалять и не менять структуру `user_workspaces` до полного перехода на новую систему.

---

## 6. Интеграция с существующими таблицами

### 6.1. Схема связей новых таблиц с существующими

```
┌─────────────────────────────────────────────────────────────────┐
│                       Существующие таблицы                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  users ───────────────────────┐                                 │
│    ↑                          │                                 │
│    │                          ▼                                 │
│  user_workspaces        user_role_assignments                   │
│    ↑                          ↑                                 │
│    │                          │                                 │
│  workspaces ──────────────────┼─────────────────┐               │
│    ↑                          │                 │               │
│    │                          ▼                 ▼               │
│    └─────────────────── workspace_roles   user_permissions      │
│                             ↑                 ↑                 │
│                             │                 │                 │
│                             └─────────────────┼─────────────────┘
│                                                │
│                                        permission_catalog
│                                                ↑
│                                                │
│  modules ──────────────────────────────────────┘
│    ↑
│    │
│  workspace_modules
│    ↑
│    │
│  user_module_licenses
│
└─────────────────────────────────────────────────────────────────┘
```

### 6.2. Как связаны данные

| Новая таблица | Связь с существующей | Назначение связи |
|---------------|----------------------|-------------------|
| `workspace_roles.workspace_id` | → `workspaces.id` | Роль принадлежит workspace |
| `user_role_assignments.user_id` | → `users.id` | Кто назначен |
| `user_role_assignments.role_id` | → `workspace_roles.id` | Какая роль назначена |
| `user_role_assignments.workspace_id` | → `workspaces.id` | В каком workspace |
| `user_permissions.permission_id` | → `permission_catalog.id` | Какое право |
| `user_permissions.workspace_id` | → `workspaces.id` | Контекст |
| `permission_catalog.module_code` | → `modules.code` | Модуль права |

### 6.3. Ключевые индексы для производительности

```sql
-- Для частого запроса "проверить членство в workspace"
-- (уже есть в user_workspaces)
CREATE INDEX idx_user_workspaces_lookup ON user_workspaces(user_id, workspace_id);

-- Для частого запроса "получить все роли пользователя"
CREATE INDEX idx_user_role_assignments_lookup ON user_role_assignments(user_id, workspace_id);

-- Для частого запроса "проверить лицензию"
CREATE INDEX idx_user_module_licenses_lookup ON user_module_licenses(user_id, module_id, workspace_id) 
WHERE status = 'active';

-- Для Casbin (автоматически)
CREATE INDEX idx_casbin_rule_lookup ON casbin_rule(ptype, v0, v1, v2, v3);
```

---

## 7. План миграции данных

### 7.1. Этап 1: Создание системных ролей

Для каждого существующего workspace создаем системные роли:

```sql
-- Для каждого workspace создаем OWNER, ADMIN, MEMBER, GUEST
INSERT INTO workspace_roles (id, workspace_id, name, is_system)
SELECT gen_random_uuid(), id, 'OWNER', true FROM workspaces
UNION ALL
SELECT gen_random_uuid(), id, 'ADMIN', true FROM workspaces
UNION ALL
SELECT gen_random_uuid(), id, 'MEMBER', true FROM workspaces
UNION ALL
SELECT gen_random_uuid(), id, 'GUEST', true FROM workspaces;
```

### 7.2. Этап 2: Перенос существующих назначений

Переносим данные из `user_workspaces` в `user_role_assignments`:

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

### 7.3. Этап 3: Создание групповых политик в Casbin

После переноса данных добавляем групповые политики:

```sql
-- Это делается кодом, но для понимания:
-- g, user:<user_id>, role:<role_name>, <workspace_id>
```

### 7.4. Этап 4: Верификация миграции

Проверяем, что количество записей совпадает:

```sql
-- Должно быть равно
SELECT COUNT(*) FROM user_workspaces;

-- Должно быть равно предыдущему запросу
SELECT COUNT(*) FROM user_role_assignments;
```

### 7.5. Этап 5: Обратная совместимость

Оставляем `user_workspaces` как есть, но в коде меняем логику:
- Для проверки членства используем `user_workspaces` (быстро, индексы есть)
- Для проверки прав используем новую систему

---

## 8. Рекомендации по использованию существующих индексов

### 8.1. Индексы, которые уже есть и будут использоваться

| Индекс | Использование |
|--------|---------------|
| `idx_user_workspaces_user_id` | Поиск workspace пользователя |
| `idx_user_workspaces_workspace_id` | Поиск пользователей в workspace |
| `idx_workspace_modules_workspace_id` | Проверка модуля |
| `idx_user_module_licenses_user_id` | Проверка лицензий |

### 8.2. Индексы, которые нужно добавить

```sql
-- Составной индекс для быстрой проверки членства
CREATE INDEX idx_user_workspaces_lookup ON user_workspaces(user_id, workspace_id);

-- Индекс для поиска ролей пользователя
CREATE INDEX idx_user_role_assignments_lookup ON user_role_assignments(user_id, workspace_id);

-- Индекс для каталога прав
CREATE INDEX idx_permission_catalog_lookup ON permission_catalog(module_code, entity_type, action);
```

### 8.3. Покрытие запросов

**Типичный запрос проверки прав:**
```sql
-- Шаг 1: Проверить членство в workspace (user_workspaces)
SELECT 1 FROM user_workspaces WHERE user_id = $1 AND workspace_id = $2;

-- Шаг 2: Проверить модуль (workspace_modules)
SELECT 1 FROM workspace_modules wm
JOIN modules m ON m.id = wm.module_id
WHERE wm.workspace_id = $1 AND m.code = $2 AND wm.status = 'active';

-- Шаг 3: Получить роли пользователя (user_role_assignments + workspace_roles)
SELECT wr.name FROM user_role_assignments ura
JOIN workspace_roles wr ON wr.id = ura.role_id
WHERE ura.user_id = $1 AND ura.workspace_id = $2;

-- Шаг 4: Проверить в Casbin (в памяти, не SQL)
```

---

## 9. Заключение по структуре БД

### 9.1. Что остается без изменений

| Таблица | Изменения | Причина |
|---------|-----------|---------|
| `users` | Без изменений | Уже содержит все необходимое |
| `workspaces` | Без изменений | Базовая структура |
| `modules` | Без изменений | Справочник модулей |
| `workspace_modules` | Без изменений | Проверка модулей |
| `user_module_licenses` | Без изменений | Проверка лицензий |
| Все таблицы с данными | Без изменений | Содержат workspace_id |

### 9.2. Что требует внимания

| Таблица | Действие |
|---------|----------|
| `user_workspaces` | Оставить, но перестать использовать для прав (только членство) |

### 9.3. Что добавляется

| Таблица | Назначение |
|---------|------------|
| `permission_catalog` | Словарь всех возможных прав |
| `workspace_roles` | Кастомные роли в workspace |
| `user_role_assignments` | Назначение ролей пользователям |
| `user_permissions` | Индивидуальные права |
| `role_inheritance` | Наследование ролей |
| `casbin_rule` | Политики Casbin |

### 9.4. Ключевые преимущества подхода

1. **Минимальные изменения** существующей структуры
2. **Обратная совместимость** через сохранение `user_workspaces`
3. **Использование существующих индексов** для производительности
4. **Постепенная миграция** без остановки системы
5. **Четкое разделение** членства в workspace и прав доступа

### 9.5. Риски и их mitigation

| Риск | Вероятность | Решение |
|------|-------------|---------|
| Деградация производительности | Средняя | Использовать индексы, кэширование Casbin |
| Потеря данных при миграции | Низкая | Транзакционная миграция, бэкапы |
| Конфликт с существующим кодом | Высокая | Поэтапное включение, режим логирования |
| Усложнение архитектуры | Средняя | Четкая документация, код-ревью |

---

**Документ подготовлен:** Март 2026  
**Версия:** 5.0 (с полным анализом БД)  
**Статус:** Утвержден к реализации
