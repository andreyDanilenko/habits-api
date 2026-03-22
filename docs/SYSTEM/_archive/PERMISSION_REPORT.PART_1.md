# Отчёт по внедрению гибкой системы ролей и прав доступа (Часть 1)

**Дата:** Март 2026  
**Файл:** `PERMISSION_REPORT.PART_1.md`  

---

## 1. Исходные документы и анализ

В рамках подготовки к внедрению новой системы прав были проанализированы и согласованы следующие спецификации:

- `perrmissions.spec.back.v1.md` — подробная постановка задачи для бэкенда:
  - новые таблицы (`permission_catalog`, `workspace_roles`, `user_role_assignments`, `user_permissions`, `role_inheritance`);
  - три новых middleware (`WorkspaceMiddleware`, `ModuleLicenseMiddleware`, `PermissionMiddleware`);
  - `PermissionService` и набор API-эндпоинтов;
  - интеграция с Casbin (модель, enforcer, синхронизация политик).
- `perrmissions.v1.md` — системный анализ и план:
  - цели (бизнесовые и технические);
  - требования к гранулярным правам, кастомным ролям, наследованию, индивидуальным правам;
  - этапный план внедрения и критерии приемки.
- `perrmissions.v2.md` — расширенная техническая спецификация:
  - полный SQL для новых таблиц;
  - пример реализации middleware-цепочки;
  - пример `PermissionService` и API-хендлеров;
  - пример Casbin-модели и инициализации enforcer.
- `perrmissions.v3.md` — анализ текущей структуры БД:
  - подтверждение использования `user_workspaces`, `modules`, `workspace_modules`, `user_module_licenses`;
  - согласование новой модели данных с существующими таблицами и индексами;
  - подробный план миграций.

По итогам анализа текущего кода и миграций (`backend/migrations`, `SQL_AND_MIGRATIONS_GUIDE.md`) подтверждено:

- используется `golang-migrate`, миграции лежат в `backend/migrations`;
- существуют таблицы `users`, `workspaces`, `user_workspaces`, `modules`, `workspace_modules`, `user_module_licenses`;
- URL-структура и роутер соответствуют предпосылкам в спецификациях (`/api/v1/workspaces/:workspaceId/...`).

---

## 2. Этап 1: Схема БД для новой системы прав

### 2.1. Новая миграция 000022_permissions_schema_and_seed

Создана пара миграций:

- `000022_permissions_schema_and_seed.up.sql`
- `000022_permissions_schema_and_seed.down.sql`

**Основные объекты:**

1. **`permission_catalog`** — словарь всех возможных прав:
   - поля: `id`, `module_code`, `entity_type`, `action`, `name`, `description`, `is_system`, `created_at`;
   - уникальность по `(module_code, entity_type, action)`;
   - индексы по `module_code` и `entity_type`;
   - поддерживаются коды модулей: `crm`, `habits`, `projects`, а также специальный код `workspace` для прав уровня рабочего пространства (управление участниками, ролями и модулями).

2. **`workspace_roles`** — роли внутри workspace:
   - поля: `id`, `workspace_id`, `name`, `description`, `is_system`, `created_at`, `updated_at`;
   - уникальность по `(workspace_id, name)`;
   - `is_system = true` для системных ролей OWNER/ADMIN/MEMBER/GUEST.

3. **`user_role_assignments`** — назначение ролей пользователям:
   - поля: `id`, `user_id`, `role_id`, `workspace_id`, `assigned_by`, `assigned_at`;
   - уникальность по `(user_id, role_id, workspace_id)`;
   - индексы для быстрых запросов:
     - по `user_id`, по `role_id`, по `workspace_id`;
     - составной индекс `(user_id, workspace_id)` для выборки всех ролей пользователя в workspace.

4. **`user_permissions`** — индивидуальные права пользователя:
   - поля: `id`, `user_id`, `workspace_id`, `permission_id`, `granted_by`, `granted_at`, `expires_at`;
   - уникальность по `(user_id, workspace_id, permission_id)`;
   - индексы по `user_id`, `workspace_id` и составной `(user_id, workspace_id)`.

5. **`role_inheritance`** — наследование ролей:
   - поля: `id`, `workspace_id`, `child_role_id`, `parent_role_id`, `created_at`;
   - CHECK, запрещающий self-inheritance (`child_role_id != parent_role_id`);
   - уникальность по `(workspace_id, child_role_id, parent_role_id)`;
   - индексы по `child_role_id` и `parent_role_id`.

### 2.2. Первичное наполнение каталога прав

В `permission_catalog` сразу добавлен базовый набор системных прав (`is_system = true`), согласованный с документацией:

- **CRM**:
  - `crm:deal:{create,read,update,delete,move}`;
  - `crm:contact:{create,read,update,delete}`;
  - `crm:company:{create,read,update,delete}`;
  - `crm:pipeline:manage`;
  - `crm:activity:{create,read,update,delete}`;
  - `crm:export:deals`.
- **Habits**:
  - `habits:habit:{create,read,update,delete,complete}`;
  - `habits:journal:{create,read,update,delete}`.
- **Projects**:
  - `projects:project:{create,read,update,delete}`;
  - `projects:entity:{attach,detach}`.
- **Workspace (админка)**:
  - `workspace:member:{invite,remove}`;
  - `workspace:role:manage`;
  - `workspace:module:manage`.

Для всех вставок используется `ON CONFLICT (module_code, entity_type, action) DO NOTHING`, чтобы миграция оставалась идемпотентной.

### 2.3. Откат миграции

`000022_permissions_schema_and_seed.down.sql` удаляет новые таблицы в корректном порядке (сначала зависящие таблицы, затем базовый словарь), не затрагивая остальные объекты схемы.

---

## 3. Этап 1.5: Миграция существующих ролей и системные роли

### 3.1. Миграция 000023_system_workspace_roles_and_assignments

Создана пара миграций:

- `000023_system_workspace_roles_and_assignments.up.sql`
- `000023_system_workspace_roles_and_assignments.down.sql`

**Цели миграции:**

1. Создать системные роли для всех существующих `workspaces`.
2. Перенести существующие роли из таблицы `user_workspaces` в новую таблицу `user_role_assignments`.
3. Обеспечить автоматическое создание системных ролей при создании новых workspaces.

**Содержимое up-миграции:**

1. **Создание системных ролей по всем существующим workspaces:**

```sql
INSERT INTO workspace_roles (id, workspace_id, name, is_system, created_at, updated_at)
SELECT gen_random_uuid(), id, 'OWNER', true, NOW(), NOW() FROM workspaces
UNION ALL
SELECT gen_random_uuid(), id, 'ADMIN', true, NOW(), NOW() FROM workspaces
UNION ALL
SELECT gen_random_uuid(), id, 'MEMBER', true, NOW(), NOW() FROM workspaces
UNION ALL
SELECT gen_random_uuid(), id, 'GUEST', true, NOW(), NOW() FROM workspaces;
```

2. **Перенос существующих назначений из `user_workspaces`:**

```sql
INSERT INTO user_role_assignments (id, user_id, role_id, workspace_id, assigned_at)
SELECT 
    gen_random_uuid(),
    uw.user_id,
    wr.id,
    uw.workspace_id,
    uw.created_at
FROM user_workspaces uw
JOIN workspace_roles wr 
  ON wr.workspace_id = uw.workspace_id 
 AND wr.name = uw.role;
```

Это реализует один-в-один логику из спецификаций, сохраняя историю `created_at` как `assigned_at`.

3. **Триггер для системных ролей при создании нового workspace:**

- Функция `fn_create_system_roles()` вставляет OWNER/ADMIN/MEMBER/GUEST в `workspace_roles` с `is_system = true` и текущими временными метками.
- Триггер `tr_create_system_roles` вызывается `AFTER INSERT ON workspaces FOR EACH ROW`.

**Содержимое down-миграции:**

- Удаляется триггер и функция:
  - `DROP TRIGGER IF EXISTS tr_create_system_roles ON workspaces;`
  - `DROP FUNCTION IF EXISTS fn_create_system_roles();`
- Очищаются назначения ролей:
  - `DELETE FROM user_role_assignments;`
- Удаляются только системные роли:
  - `DELETE FROM workspace_roles WHERE is_system = true;`

При этом таблицы `workspace_roles` и `user_role_assignments` остаются, как предусмотрено в `000022`.

---

## 4. Этап 2: Инфраструктура Casbin

### 4.1. Обновление зависимостей

В `backend/go.mod` добавлены следующие зависимости:

- `github.com/casbin/casbin/v3 v3.10.0` — ядро Casbin (используется модульная версия v3);
- `github.com/casbin/gorm-adapter/v3 v3.41.0` — адаптер Casbin для GORM и PostgreSQL (таблица `casbin_rule`);
- `gorm.io/gorm v1.31.1` — ORM-слой, необходимый для адаптера.

После добавления зависимостей успешно выполнен `go mod tidy` в каталоге `backend`.

### 4.2. Пакет internal/authz и InitEnforcer

Создан файл `backend/internal/authz/casbin.go` с функцией инициализации Casbin:

- Функция `InitEnforcer(db *gorm.DB) (*casbin.Enforcer, error)`:
  - создаёт адаптер `gormadapter.NewAdapterByDB(db)` (используется стандартная таблица `casbin_rule`);
  - формирует Casbin-модель с поддержкой доменов (workspace_id) и иерархии ролей:
    - `r = sub, dom, obj, act`;
    - `p = sub, dom, obj, act`;
    - `g = _, _, _` (роль иерархическая, с доменом);
    - matcher: `g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act`.
  - создаёт `*casbin.Enforcer` на основе модели и адаптера;
  - включает кэширование (`EnableCache(true)`);
  - загружает политики из БД (`LoadPolicy()`).

На этом этапе `enforcer` ещё не интегрирован в DI-контейнер и middleware, однако есть единая точка для дальнейшего использования в `PermissionService` и `PermissionMiddleware`.

---

## 5. Текущее состояние и соответствие спецификациям

На момент окончания Части 1:

- **Модель данных** для гибкой системы прав реализована и соответствует спецификациям:
  - добавлены все необходимые таблицы (кроме `casbin_rule`, которую создаёт адаптер Casbin);
  - реализованы индексы и уникальные ограничения для производительности и целостности;
  - `permission_catalog` содержит базовый набор прав для модулей CRM, Habits, Projects и уровня workspace.
- **Миграция существующей ролевой модели**:
  - системные роли OWNER/ADMIN/MEMBER/GUEST создаются для всех существующих и новых workspaces;
  - текущие назначения из `user_workspaces` перенесены в `user_role_assignments` без потери информации о времени создания.
- **Инфраструктура Casbin**:
  - зависимости Casbin и GORM добавлены в проект;
  - реализована функция `InitEnforcer` с моделью, полностью согласованной с документацией.

Важно: на данном этапе **логика приложения (middleware и хендлеры) не изменялась** — все проверки авторизации по-прежнему выполняются по старой схеме, что соответствует принципу поэтапного включения без «большого взрыва».

---

## 6. План дальнейших шагов (Часть 2 и далее)

В следующих частях планируется:

1. **Интеграция Casbin в DI-контейнер и приложение**:
   - создание GORM-обёртки над существующим `*sql.DB` (без ломки текущей инициализации);
   - добавление `*casbin.Enforcer` в `di.Container` и прокидывание его в сервисы/handlers.

2. **Реализация middleware-цепочки**:
   - `WorkspaceMiddleware` — определение `workspace_id`, проверка членства (через `user_workspaces`), наполнение контекста;
   - `ModuleLicenseMiddleware` — проверка включённости модуля и лицензий (`modules`, `workspace_modules`, `user_module_licenses`);
   - `PermissionMiddleware` — маппинг эндпоинтов на `(object, action)`, проверка через Casbin и индивидуальные права.
   - На первом этапе — запуск `PermissionMiddleware` в режиме логирования (без блокировки доступа).

3. **Реализация `PermissionService` и API**:
   - CRUD по ролям и их правам;
   - назначение/снятие ролей у пользователей;
   - выдача индивидуальных прав;
   - получение эффективных прав текущего пользователя (`/api/v1/me/permissions`).

4. **Поэтапное включение новой системы и удаление старых проверок**:
   - сравнение логов PermissionMiddleware с текущим поведением;
   - включение блокировки на основе Casbin;
   - постепенный рефакторинг хендлеров с удалением харкодированных проверок ролей.

Эти шаги будут отражены в следующих отчётах (`PERMISSION_REPORT.PART_2.md` и далее).

