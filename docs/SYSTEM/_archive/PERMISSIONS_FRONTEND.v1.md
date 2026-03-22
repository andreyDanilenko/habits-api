# ИТОГОВЫЙ СИСТЕМНЫЙ АНАЛИЗ: Внедрение гибкой системы прав доступа в ERP

## Версия 2.0 | Март 2026

---

## Содержание

1. **Контекст и цели проекта**
2. **Архитектурное решение**
3. **Модель данных (детально)**
4. **Бэкенд-архитектура**
5. **Фронтенд-архитектура**
6. **Согласование типов между слоями**
7. **Интеграция с существующими модулями**
8. **План миграции**
9. **Этапы реализации**
10. **Риски и способы их снижения**
11. **Критерии приемки**
12. **Приложения**

---

## 1. Контекст и цели проекта

### 1.1. Текущая ситуация

В существующей ERP-системе реализована базовая ролевая модель:

- **Глобальные роли** (`users.role`): `ADMIN` (полный доступ ко всем workspace), `USER` (обычный пользователь)
- **Роли в workspace** (`user_workspaces.role`): `OWNER`, `ADMIN`, `MEMBER`, `GUEST`

**Проблемы текущей модели:**
- Невозможно создать кастомные роли (например, "Менеджер по продажам")
- Нет гранулярности прав (нельзя дать доступ только на создание, но не на удаление)
- Проверки прав размазаны по коду, дублируются
- GUEST не имеет read-only доступа (сейчас просто нет в workspace)
- Фронтенд не знает о правах пользователя и показывает все кнопки всем

### 1.2. Цели внедрения

| Цель | Описание | Бизнес-ценность |
|------|----------|-----------------|
| **Гранулярность** | Права вида `модуль:сущность:действие` | Точная настройка доступа |
| **Кастомные роли** | Создание ролей под структуру компании | Адаптация под бизнес-процессы |
| **Наследование** | Иерархия ролей (Senior Manager = Manager + права) | Упрощение управления |
| **Индивидуальные права** | Временный доступ для конкретных сотрудников | Гибкость |
| **Read-only для GUEST** | Гости могут только просматривать | Безопасность + новые сценарии |
| **Адаптивный UI** | Фронтенд скрывает недоступные действия | Улучшение UX |

### 1.3. Ключевые принципы проектирования

1. **Не ломать существующее** — обратная совместимость на всех этапах
2. **Постепенность** — поэтапное внедрение без "большого взрыва"
3. **Типобезопасность** — согласованные типы между бэкендом и фронтендом
4. **Производительность** — проверка прав < 5 мс
5. **Прозрачность** — понятная модель данных и API

---

## 2. Архитектурное решение

### 2.1. Общая схема

```
┌─────────────────────────────────────────────────────────────────────┐
│                         ФРОНТЕНД                                      │
├─────────────────────────────────────────────────────────────────────┤
│  [Vue/React] → [Permission Store] → [UI Components]                  │
│                   ↓ проверяет права                                   │
│              [can(Perm.crm.deal.create)]                             │
└─────────────────────────────────────────────────────────────────────┘
                                    │ API
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         БЭКЕНД                                        │
├─────────────────────────────────────────────────────────────────────┤
│  [AuthMiddleware] → [WorkspaceMiddleware] → [ModuleMiddleware]       │
│                                              → [PermissionMiddleware]│
│         ↓ проверяет через                                              │
│  [Casbin Enforcer] ← [PermissionService] ← [API Handlers]            │
│         ↓ хранит в                                                     │
│  [PostgreSQL: casbin_rule]                                           │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         БАЗА ДАННЫХ                                   │
├─────────────────────────────────────────────────────────────────────┤
│  Существующие таблицы:              Новые таблицы:                   │
│  • users                            • permission_catalog             │
│  • workspaces                       • workspace_roles                 │
│  • user_workspaces                   • user_role_assignments          │
│  • modules                           • user_permissions               │
│  • workspace_modules                 • role_inheritance               │
│  • user_module_licenses              • casbin_rule                    │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2. Трёхуровневая модель прав

```
УРОВЕНЬ 1: Глобальный (всё приложение)
├── ADMIN - полный доступ к любым workspace
└── USER - обычный пользователь

УРОВЕНЬ 2: Системные роли workspace (статус)
├── OWNER - владелец (может удалить workspace)
├── ADMIN - администратор (управляет настройками)
├── MEMBER - сотрудник (активный участник)
└── GUEST - гость (только просмотр)

УРОВЕНЬ 3: Кастомные роли (должности)
├── Создаются ADMIN-ом workspace
├── Назначаются MEMBER-ам
├── Могут наследовать друг друга
└── Состоят из набора прав

УРОВЕНЬ 4: Права (permissions)
├── Формат: {модуль}:{сущность}:{действие}
├── Пример: crm:deal:create, crm:deal:read
└── Являются "кирпичиками" для построения ролей
```

---

## 3. Модель данных (детально)

### 3.1. Существующие таблицы (используются)

```sql
-- Глобальные пользователи
users (
    id UUID PRIMARY KEY,
    role VARCHAR(20) DEFAULT 'USER'  -- ADMIN или USER
)

-- Workspace
workspaces (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL
)

-- Связь пользователей с workspace (текущая ролевая модель)
user_workspaces (
    user_id UUID,
    workspace_id UUID,
    role VARCHAR(50) DEFAULT 'MEMBER'  -- OWNER, ADMIN, MEMBER, GUEST
)

-- Модули системы
modules (
    id UUID PRIMARY KEY,
    code VARCHAR(50) UNIQUE,  -- crm, habits, projects
    is_core BOOLEAN
)

-- Какие модули включены в workspace
workspace_modules (
    workspace_id UUID,
    module_id UUID,
    status VARCHAR(20)
)

-- Лицензии пользователей на модули
user_module_licenses (
    user_id UUID,
    module_id UUID,
    scope VARCHAR(20),  -- all_workspaces, single_workspace
    workspace_id UUID
)
```

### 3.2. Новые таблицы (добавляются)

```sql
-- ===== КАТАЛОГ ПРАВ (словарь) =====
CREATE TABLE permission_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_code VARCHAR(50) NOT NULL,      -- crm, habits, projects
    entity_type VARCHAR(50) NOT NULL,      -- deal, contact, habit
    action VARCHAR(50) NOT NULL,           -- create, read, update, delete
    name VARCHAR(255) NOT NULL,             -- "Создание сделки" (для UI)
    description TEXT,
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(module_code, entity_type, action)
);

-- ===== РОЛИ В WORKSPACE =====
CREATE TABLE workspace_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,              -- "Менеджер по продажам"
    description TEXT,
    is_system BOOLEAN DEFAULT false,         -- true для OWNER/ADMIN/MEMBER/GUEST
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, name)
);

-- ===== НАЗНАЧЕНИЕ РОЛЕЙ ПОЛЬЗОВАТЕЛЯМ =====
CREATE TABLE user_role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id),
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, role_id, workspace_id)
);

-- ===== ИНДИВИДУАЛЬНЫЕ ПРАВА =====
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

-- ===== НАСЛЕДОВАНИЕ РОЛЕЙ =====
CREATE TABLE role_inheritance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    child_role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    parent_role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CHECK (child_role_id != parent_role_id),
    UNIQUE(workspace_id, child_role_id, parent_role_id)
);

-- ===== ТАБЛИЦА CASBIN (создается адаптером) =====
CREATE TABLE casbin_rule (
    id SERIAL PRIMARY KEY,
    ptype VARCHAR(100) NOT NULL,
    v0 VARCHAR(100),
    v1 VARCHAR(100),
    v2 VARCHAR(100),
    v3 VARCHAR(100),
    v4 VARCHAR(100),
    v5 VARCHAR(100)
);
```

### 3.3. Индексы для производительности

```sql
-- Для user_role_assignments (частые запросы)
CREATE INDEX idx_user_role_assignments_lookup ON user_role_assignments(user_id, workspace_id);
CREATE INDEX idx_user_role_assignments_role ON user_role_assignments(role_id);

-- Для user_permissions
CREATE INDEX idx_user_permissions_lookup ON user_permissions(user_id, workspace_id);
CREATE INDEX idx_user_permissions_expires ON user_permissions(expires_at) WHERE expires_at IS NOT NULL;

-- Для casbin
CREATE INDEX idx_casbin_rule_lookup ON casbin_rule(ptype, v0, v1, v2, v3);
```

### 3.4. Первичное наполнение permission_catalog

```sql
-- CRM
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

-- Habits
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

-- Projects
INSERT INTO permission_catalog (module_code, entity_type, action, name) VALUES
('projects', 'project', 'create', 'Создание проекта'),
('projects', 'project', 'read', 'Просмотр проекта'),
('projects', 'project', 'update', 'Редактирование проекта'),
('projects', 'project', 'delete', 'Удаление проекта'),
('projects', 'entity', 'attach', 'Привязка сущности'),
('projects', 'entity', 'detach', 'Отвязка сущности');

-- Workspace (администрирование)
INSERT INTO permission_catalog (module_code, entity_type, action, name) VALUES
('workspace', 'member', 'invite', 'Приглашение участников'),
('workspace', 'member', 'remove', 'Удаление участников'),
('workspace', 'role', 'manage', 'Управление ролями'),
('workspace', 'module', 'manage', 'Управление модулями');
```

---

## 4. Бэкенд-архитектура

### 4.1. Middleware цепочка

```
[Request] → [AuthMiddleware] → [WorkspaceMiddleware] → [ModuleMiddleware] → [PermissionMiddleware] → [Handler]
    1. Извлекает          2. Извлекает           3. Проверяет            4. Проверяет
       user_id из           workspace_id из         модуль и                права через
       JWT                  URL, проверяет          лицензию                Casbin
                             членство
```

#### WorkspaceMiddleware
- Извлекает `workspace_id` из URL (паттерн `/api/v1/workspaces/{workspaceId}/...`)
- Проверяет наличие пользователя в `user_workspaces`
- Добавляет `workspace_id` в контекст

#### ModuleMiddleware
- Определяет модуль по пути (`/crm/` → `crm`)
- Проверяет `workspace_modules` (включен ли модуль)
- Для не-core модулей проверяет `user_module_licenses`
- Добавляет `module_code` в контекст

#### PermissionMiddleware
- Маппит endpoint на объект и действие (`POST /deals` → `crm:deal:create`)
- Получает все роли пользователя через `user_role_assignments` и проверяет их через Casbin (`enforcer.Enforce("user:"+userId, workspace_id, obj, act)` с учётом g(user, role, workspace_id))
- Индивидуальные права учитываются на уровне сервиса (`/me/permissions`) и могут быть дополнительно синхронизированы в Casbin.
- **Текущий этап внедрения:** middleware работает в **режиме логирования** — решение Casbin только логируется, доступ не блокируется (для безопасного поэтапного запуска).
- **Боевой режим:** после включения проверок при отсутствии права запрос будет завершаться `403 Forbidden` (источник истины — Casbin + индивидуальные права).

### 4.2. Интеграция с Casbin

**Модель Casbin:**
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

**Политики для системных ролей:**
```sql
-- ADMIN имеет все права
('p', 'role:ADMIN', '*', '*', '*')

-- OWNER имеет все права
('p', 'role:OWNER', '*', '*', '*')

-- MEMBER: базовые права
('p', 'role:MEMBER', '*', 'crm:deal', 'read')
('p', 'role:MEMBER', '*', 'crm:contact', 'read')
('p', 'role:MEMBER', '*', 'crm:company', 'read')

-- GUEST: только чтение
('p', 'role:GUEST', '*', 'crm:deal', 'read')
('p', 'role:GUEST', '*', 'crm:contact', 'read')
('p', 'role:GUEST', '*', 'crm:company', 'read')
```

### 4.3. API Эндпоинты

#### Для администрирования ролей
```
GET    /api/v1/workspaces/{workspaceId}/permissions/catalog
GET    /api/v1/workspaces/{workspaceId}/roles
POST   /api/v1/workspaces/{workspaceId}/roles
GET    /api/v1/workspaces/{workspaceId}/roles/{roleId}
PUT    /api/v1/workspaces/{workspaceId}/roles/{roleId}
DELETE /api/v1/workspaces/{workspaceId}/roles/{roleId}
```

#### Для назначения ролей пользователям
```
GET    /api/v1/workspaces/{workspaceId}/users
GET    /api/v1/workspaces/{workspaceId}/users/{userId}/roles
POST   /api/v1/workspaces/{workspaceId}/users/{userId}/roles/{roleId}
DELETE /api/v1/workspaces/{workspaceId}/users/{userId}/roles/{roleId}
```

#### Для индивидуальных прав
```
GET    /api/v1/workspaces/{workspaceId}/users/{userId}/permissions
POST   /api/v1/workspaces/{workspaceId}/users/{userId}/permissions
DELETE /api/v1/workspaces/{workspaceId}/users/{userId}/permissions/{permissionId}
```

#### Для наследования
```
POST   /api/v1/workspaces/{workspaceId}/roles/{roleId}/inherit/{parentRoleId}
DELETE /api/v1/workspaces/{workspaceId}/roles/{roleId}/inherit/{parentRoleId}
```

#### Для фронтенда (текущий пользователь)
```
GET /api/v1/me/permissions?workspaceId={id}
```

---

## 5. Фронтенд-архитектура

### 5.1. Общая структура

```
src/
├── types/
│   ├── permission.types.ts          # Сгенерированные типы
│   └── api.types.ts                  # Типы для API
│
├── stores/
│   └── permission.store.ts           # Хранилище прав
│
├── utils/
│   └── permissions.ts                 # Утилиты can(), Perm
│
├── api/
│   └── permission.api.ts              # API клиент
│
├── components/
│   └── permissions/                    # Компоненты для админки
│       ├── RoleList.vue
│       ├── RoleEditor.vue
│       ├── PermissionTree.vue
│       └── UserRoleModal.vue
│
└── views/
    ├── RoleManager.vue                 # Управление ролями
    └── UserRoleManager.vue              # Назначение ролей пользователям
```

### 5.2. Модели данных (TypeScript)

```typescript
// types/permission.types.ts (генерируется из бэкенда)
export type PermissionString = 
    | 'crm:deal:create'
    | 'crm:deal:read'
    | 'crm:deal:update'
    | 'crm:deal:delete'
    | 'crm:deal:move'
    | 'crm:contact:create'
    | 'crm:contact:read'
    | 'crm:contact:update'
    | 'crm:contact:delete'
    | 'habits:habit:create'
    | 'habits:habit:read'
    | 'habits:habit:update'
    | 'habits:habit:delete'
    | 'habits:habit:complete'
    | 'workspace:member:invite'
    | 'workspace:role:manage';

export interface Role {
    id: string;
    name: string;
    description?: string;
    isSystem: boolean;
    permissions: PermissionString[];
    userCount?: number;
    inherits?: string;
    createdAt: string;
    updatedAt: string;
}

export interface UserWithRoles {
    id: string;
    name: string;
    email: string;
    avatar?: string;
    systemRole: 'OWNER' | 'ADMIN' | 'MEMBER' | 'GUEST';
    customRoles: Role[];
    individualPermissions: {
        permission: PermissionString;
        expiresAt?: string;
        grantedAt: string;
    }[];
}

export interface MyPermissionsResponse {
    permissions: PermissionString[];
    roles: string[];
    systemRole: 'OWNER' | 'ADMIN' | 'MEMBER' | 'GUEST';
}
```

### 5.3. Утилиты для работы с правами

```typescript
// utils/permissions.ts
import { PermissionString } from '@/types/permission.types';
import { usePermissionStore } from '@/stores/permission.store';

// Иерархический объект для автокомплита
export const Perm = {
    crm: {
        deal: {
            create: 'crm:deal:create' as PermissionString,
            read: 'crm:deal:read' as PermissionString,
            update: 'crm:deal:update' as PermissionString,
            delete: 'crm:deal:delete' as PermissionString,
            move: 'crm:deal:move' as PermissionString,
        },
        contact: {
            create: 'crm:contact:create' as PermissionString,
            read: 'crm:contact:read' as PermissionString,
            update: 'crm:contact:update' as PermissionString,
            delete: 'crm:contact:delete' as PermissionString,
        },
        company: {
            create: 'crm:company:create' as PermissionString,
            read: 'crm:company:read' as PermissionString,
            update: 'crm:company:update' as PermissionString,
            delete: 'crm:company:delete' as PermissionString,
        },
        pipeline: {
            manage: 'crm:pipeline:manage' as PermissionString,
        },
        activity: {
            create: 'crm:activity:create' as PermissionString,
            read: 'crm:activity:read' as PermissionString,
            update: 'crm:activity:update' as PermissionString,
            delete: 'crm:activity:delete' as PermissionString,
        },
        export: {
            deals: 'crm:export:deals' as PermissionString,
        },
    },
    habits: {
        habit: {
            create: 'habits:habit:create' as PermissionString,
            read: 'habits:habit:read' as PermissionString,
            update: 'habits:habit:update' as PermissionString,
            delete: 'habits:habit:delete' as PermissionString,
            complete: 'habits:habit:complete' as PermissionString,
        },
        journal: {
            create: 'habits:journal:create' as PermissionString,
            read: 'habits:journal:read' as PermissionString,
            update: 'habits:journal:update' as PermissionString,
            delete: 'habits:journal:delete' as PermissionString,
        },
    },
    projects: {
        project: {
            create: 'projects:project:create' as PermissionString,
            read: 'projects:project:read' as PermissionString,
            update: 'projects:project:update' as PermissionString,
            delete: 'projects:project:delete' as PermissionString,
        },
        entity: {
            attach: 'projects:entity:attach' as PermissionString,
            detach: 'projects:entity:detach' as PermissionString,
        },
    },
    workspace: {
        member: {
            invite: 'workspace:member:invite' as PermissionString,
            remove: 'workspace:member:remove' as PermissionString,
        },
        role: {
            manage: 'workspace:role:manage' as PermissionString,
        },
        module: {
            manage: 'workspace:module:manage' as PermissionString,
        },
    },
} as const;

// Проверка права
export function can(permission: PermissionString): boolean {
    const store = usePermissionStore();
    return store.permissions.includes(permission);
}

// Проверка нескольких прав (хотя бы одно)
export function canAny(permissions: PermissionString[]): boolean {
    const store = usePermissionStore();
    return permissions.some(p => store.permissions.includes(p));
}

// Проверка всех прав
export function canAll(permissions: PermissionString[]): boolean {
    const store = usePermissionStore();
    return permissions.every(p => store.permissions.includes(p));
}
```

### 5.4. Store для прав

```typescript
// stores/permission.store.ts
import { defineStore } from 'pinia';
import { PermissionString, MyPermissionsResponse } from '@/types/permission.types';
import { permissionApi } from '@/api/permission.api';

export const usePermissionStore = defineStore('permissions', {
    state: () => ({
        permissions: [] as PermissionString[],
        roles: [] as string[],
        systemRole: null as 'OWNER' | 'ADMIN' | 'MEMBER' | 'GUEST' | null,
        workspaceId: null as string | null,
        isLoading: false,
    }),

    getters: {
        isWorkspaceAdmin: (state) => 
            state.systemRole === 'ADMIN' || state.systemRole === 'OWNER',
        
        isGuest: (state) => state.systemRole === 'GUEST',
        
        isMember: (state) => state.systemRole === 'MEMBER',
    },

    actions: {
        async loadPermissions(workspaceId: string) {
            this.isLoading = true;
            try {
                const response = await permissionApi.getMyPermissions(workspaceId);
                this.permissions = response.permissions;
                this.roles = response.roles;
                this.systemRole = response.systemRole;
                this.workspaceId = workspaceId;
                
                // Кэшируем
                localStorage.setItem(`perms_${workspaceId}`, JSON.stringify({
                    permissions: this.permissions,
                    roles: this.roles,
                    systemRole: this.systemRole
                }));
            } catch (error) {
                console.error('Failed to load permissions', error);
            } finally {
                this.isLoading = false;
            }
        },

        clear() {
            this.permissions = [];
            this.roles = [];
            this.systemRole = null;
            this.workspaceId = null;
        }
    }
});
```

---

## 6. Согласование типов между слоями

### 6.1. Проблема

Бэкенд передаёт права как строки:
```json
{
  "permissions": ["crm:deal:create", "crm:deal:read"]
}
```

Фронтенд использует их в коде:
```javascript
// ❌ Опасно - магические строки
if (permissions.includes('crm:deal:create')) { ... }

// ❌ Опечатка не обнаружится
if (permissions.includes('crm:deal:creat')) { ... }
```

### 6.2. Решение: генерация типов

**Процесс:**
1. Бэкенд содержит актуальный `permission_catalog`
2. При изменении каталога запускается скрипт генерации
3. Скрипт создаёт TypeScript-файл с типами
4. Фронтенд импортирует сгенерированные типы

**Сгенерированный файл:**
```typescript
// generated/permission.types.ts
// Автоматически сгенерировано из БД. НЕ РЕДАКТИРОВАТЬ!

export type PermissionString = 
    | 'crm:deal:create'
    | 'crm:deal:read'
    | 'crm:deal:update'
    | 'crm:deal:delete'
    | 'crm:deal:move'
    | 'crm:contact:create'
    | 'crm:contact:read'
    | 'crm:contact:update'
    | 'crm:contact:delete'
    | 'habits:habit:create'
    | 'habits:habit:read'
    | 'habits:habit:update'
    | 'habits:habit:delete'
    | 'habits:habit:complete';

export const ALL_PERMISSIONS: PermissionString[] = [
    'crm:deal:create',
    'crm:deal:read',
    // ...
];

export function isValidPermission(perm: string): perm is PermissionString {
    return ALL_PERMISSIONS.includes(perm as PermissionString);
}
```

### 6.3. Интеграция в CI/CD

```yaml
# .github/workflows/generate-permissions.yml
name: Generate Permission Types
on:
  push:
    branches: [main]
    paths:
      - 'backend/migrations/*permissions*.sql'
jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Generate TypeScript types
        run: |
          cd backend
          go run scripts/generate-permission-types.go > ../frontend/src/generated/permission.types.ts
      - name: Create PR
        uses: peter-evans/create-pull-request@v5
        with:
          title: 'chore: update permission types'
          body: 'Автоматически сгенерировано из изменений в permission_catalog'
```

### 6.4. ESLint правило для защиты

```javascript
// eslint-plugin-permissions.js
module.exports = {
    rules: {
        'no-hardcoded-permissions': {
            create(context) {
                return {
                    Literal(node) {
                        if (typeof node.value === 'string' && 
                            node.value.match(/^[a-z]+:[a-z]+:[a-z]+$/)) {
                            context.report({
                                node,
                                message: 'Используйте Perm.xxx вместо строк. Импортируйте из @/utils/permissions'
                            });
                        }
                    }
                };
            }
        }
    }
};
```

---

## 7. Интеграция с существующими модулями

### 7.1. Что нужно изменить в каждом модуле

| Модуль | Что проверять | Пример права |
|--------|---------------|--------------|
| CRM | Создание/редактирование/удаление сделок | `crm:deal:create` |
| Habits | Управление привычками | `habits:habit:complete` |
| Projects | Управление проектами | `projects:project:create` |
| Workspace | Приглашение участников | `workspace:member:invite` |

### 7.2. Стратегия внедрения

1. **Сначала админка** (управление ролями) — не ломает существующую логику
2. **Потом интеграция в новые компоненты** — сразу используем `can(Perm.xxx)`
3. **Постепенный рефакторинг старых компонентов** — по мере доработок
4. **Удаление старых проверок** — только после полного перехода

### 7.3. Пример рефакторинга компонента

**Было:**
```vue
<template>
    <button 
        v-if="userRole === 'ADMIN' || userRole === 'OWNER'"
        @click="deleteDeal"
    >
        Удалить сделку
    </button>
</template>

<script>
export default {
    props: ['userRole']
}
</script>
```

**Стало:**
```vue
<template>
    <button 
        v-if="can(Perm.crm.deal.delete)"
        @click="deleteDeal"
    >
        Удалить сделку
    </button>
</template>

<script setup lang="ts">
import { Perm, can } from '@/utils/permissions';
</script>
```

---

## 8. План миграции

### 8.1. Миграция данных (бэкенд)

**Шаг 1:** Создание системных ролей для всех workspace
```sql
INSERT INTO workspace_roles (workspace_id, name, is_system)
SELECT id, 'OWNER', true FROM workspaces
UNION ALL
SELECT id, 'ADMIN', true FROM workspaces
UNION ALL
SELECT id, 'MEMBER', true FROM workspaces
UNION ALL
SELECT id, 'GUEST', true FROM workspaces;
```

**Шаг 2:** Перенос существующих назначений
```sql
INSERT INTO user_role_assignments (user_id, role_id, workspace_id, assigned_at)
SELECT 
    uw.user_id,
    wr.id,
    uw.workspace_id,
    uw.created_at
FROM user_workspaces uw
JOIN workspace_roles wr ON wr.workspace_id = uw.workspace_id AND wr.name = uw.role;
```

**Шаг 3:** Создание базовых политик в Casbin
```sql
-- Через код, не в SQL
```

### 8.2. Миграция кода (фронтенд)

**Шаг 1:** Создать заглушки типов (можно вручную на первое время)
**Шаг 2:** Реализовать store и утилиты
**Шаг 3:** Разработать админку с мок-данными
**Шаг 4:** Подключить реальное API
**Шаг 5:** Начать рефакторинг модулей

---

## 9. Этапы реализации

### Этап 1: Бэкенд - база данных (1 неделя)
- [ ] Создание новых таблиц
- [ ] Наполнение `permission_catalog`
- [ ] Миграция существующих данных
- [ ] Триггеры для системных ролей

### Этап 2: Бэкенд - Casbin и middleware (2 недели)
- [ ] Интеграция Casbin
- [ ] Реализация WorkspaceMiddleware
- [ ] Реализация ModuleMiddleware
- [ ] Реализация PermissionMiddleware
- [ ] API для получения каталога прав

### Этап 3: Бэкенд - API управления ролями (2 недели)
- [ ] CRUD для ролей
- [ ] Назначение ролей пользователям
- [ ] Индивидуальные права
- [ ] Наследование
- [ ] Эндпоинт `/me/permissions`

### Этап 4: Фронтенд - подготовка (1 неделя)
- [ ] Настройка генерации типов
- [ ] Store для прав
- [ ] Утилиты `can()` и `Perm`
- [ ] API клиент

### Этап 5: Фронтенд - админка (2 недели)
- [ ] Страница управления ролями
- [ ] Дерево прав
- [ ] Страница назначения ролей пользователям
- [ ] Индивидуальные права

### Этап 6: Интеграция (2 недели)
- [ ] Подключение проверок в CRM
- [ ] Подключение проверок в Habits
- [ ] Подключение проверок в Projects
- [ ] Тестирование

### Этап 7: Завершение (1 неделя)
- [ ] Удаление старых проверок
- [ ] Документация
- [ ] Финальное тестирование

**Итого:** ~11 недель (2.5 месяца)

---

## 10. Риски и способы их снижения

| Риск | Вероятность | Влияние | Способ снижения |
|------|-------------|---------|-----------------|
| **Потеря производительности** | Средняя | Высокое | Кэширование Casbin, индексы в БД, мониторинг |
| **Ошибки миграции данных** | Низкая | Высокое | Тестирование на копии, бэкапы, транзакции |
| **Неполнота маппинга эндпоинтов** | Средняя | Среднее | Автоматическая генерация из роутера |
| **Конфликт с существующей логикой** | Высокая | Среднее | Поэтапное включение, режим логирования |
| **Сложность отладки прав** | Средняя | Среднее | Детальное логирование решений PermissionMiddleware |
| **Рассинхронизация типов** | Низкая | Среднее | Автоматическая генерация, CI/CD |

---

## 11. Критерии приемки

### 11.1. Бэкенд

- [ ] Все новые таблицы созданы, данные перенесены
- [ ] WorkspaceMiddleware корректно определяет workspace_id
- [ ] ModuleMiddleware проверяет модули и лицензии
- [ ] PermissionMiddleware проверяет права через Casbin
- [ ] API управления ролями работает (CRUD)
- [ ] API назначения ролей работает
- [ ] API `/me/permissions` возвращает корректные данные
- [ ] Время проверки прав < 5 мс
- [ ] Системные роли нельзя удалить
- [ ] GUEST не может получить права на запись

### 11.2. Фронтенд

- [ ] Типы генерируются автоматически
- [ ] Store загружает права при старте
- [ ] Утилиты `can()` и `Perm` работают
- [ ] Страница управления ролями позволяет создавать/редактировать роли
- [ ] Страница назначения ролей позволяет назначать роли пользователям
- [ ] В существующих модулях кнопки скрываются согласно правам
- [ ] ESLint правило запрещает магические строки
- [ ] Производительность не упала (замеры)

### 11.3. Интеграция

- [ ] Все существующие тесты проходят
- [ ] Нет регрессий в старом функционале
- [ ] Документация обновлена (Swagger)
- [ ] Процесс генерации типов настроен в CI/CD

---

## 12. Приложения

### 12.1. Глоссарий

| Термин | Определение |
|--------|-------------|
| **Право (Permission)** | Строка вида `модуль:сущность:действие` |
| **Кастомная роль** | Набор прав, создаваемый ADMIN-ом workspace |
| **Системная роль** | OWNER, ADMIN, MEMBER, GUEST |
| **Индивидуальное право** | Право, выданное конкретному пользователю |
| **Casbin** | Библиотека авторизации |

### 12.2. Ссылки

- [Casbin документация](https://casbin.org/docs/en/overview)
- [Текущая структура БД](./database-schema.md)
- [API спецификация (Swagger)](./api-spec.yaml)

---

**Документ подготовлен:** Март 2026  
**Версия:** 2.0  
**Автор:** Системный аналитик  
**Статус:** Утверждено к реализации
