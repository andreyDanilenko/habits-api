# Отчёт по внедрению гибкой системы ролей и прав доступа (Часть 2)

**Дата:** Март 2026  
**Файл:** `PERMISSION_REPORT.PART_2.md`  

---

## 1. Цель отчёта

- Зафиксировать текущее состояние внедрения новой системы прав после Части 1.
- Описать новую инфраструктуру (Casbin, GORM, middleware‑цепочка).
- Дать рекомендации **когда и в каком объёме фронтенд может начинать реализацию**.

---

## 2. Состояние бэкенда на текущий момент

### 2.1. База данных и миграции (резюме Части 1)

Это уже было детально описано в `PERMISSION_REPORT.PART_1.md`, кратко:

- Созданы таблицы: `permission_catalog`, `workspace_roles`, `user_role_assignments`, `user_permissions`, `role_inheritance`.
- Наполнен базовый каталог прав для модулей: CRM, Habits, Projects, а также права уровня workspace (`workspace:member:*`, `workspace:role:manage`, `workspace:module:manage`).
- Миграция `000023_system_workspace_roles_and_assignments`:
  - системные роли OWNER/ADMIN/MEMBER/GUEST созданы для всех существующих workspaces;
  - существующие назначения из `user_workspaces` перенесены в `user_role_assignments`;
  - добавлен триггер автосоздания системных ролей при создании нового workspace.

### 2.2. Интеграция GORM и Casbin

**GORM:**

- В `internal/database/database.go`:
  - `InitDB(cfg)` — как и раньше, инициализирует `*sql.DB` (PostgreSQL).
  - **Новый** `InitGormDB(cfg)` — инициализирует `*gorm.DB` на том же DSN (через `gorm.io/driver/postgres`).

**Casbin:**

- В `go.mod` добавлены:
  - `github.com/casbin/casbin/v3`,
  - `github.com/casbin/gorm-adapter/v3`,
  - `gorm.io/gorm`.
- В `internal/authz/casbin.go` реализован `InitEnforcer(db *gorm.DB)`:
  - использует GORM‑адаптер (`gormadapter.NewAdapterByDB(db)`) и таблицу `casbin_rule`;
  - модель Casbin с доменами (`dom` = `workspace_id`) и иерархией ролей:
    - `r = sub, dom, obj, act`,
    - `p = sub, dom, obj, act`,
    - `g = _, _, _`,
    - matcher: `g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act`;
  - включает кэширование и загружает политики из БД (`EnableCache(true)`, `LoadPolicy()`).

**Встраивание в приложение:**

- В `internal/app/app.go`:
  - после `InitDB` и `RunMigrations` вызывается `InitGormDB`;
  - DI‑контейнер создаётся через `di.NewContainer(db, gormDB, cfg)`.
- В `internal/di/container.go`:
  - тип `Container` расширен полем `Enforcer *casbin.Enforcer`;
  - `NewContainer` принимает `*sql.DB` и `*gorm.DB`, вызывает `authz.InitEnforcer(gormDB)` и возвращает `(*Container, error)`;
  - `Enforcer` становится доступен для middleware/сервисов через DI.

На этом этапе **политики Casbin для ролей/пользователей ещё не создаются автоматически** — это задача следующего шага (PermissionService + миграция политик).

### 2.3. Новая цепочка middleware

В `internal/di/container.go` для группы защищённых маршрутов `/api/v1` сейчас применяется следующая цепочка:

1. **`GinAuthMiddleware`**  
   - уже существующее middleware: валидирует JWT, кладёт в Gin‑контекст:
     - `user_id` (`GinUserIDKey`),
     - глобальную роль (`role` = `model.UserRole`).
   - При отсутствии/невалидном токене возвращает 401.

2. **`WorkspacePathMiddleware`** (новое, в `workspace_module_permission.go`)  
   - ожидает пути вида `/api/v1/workspaces/:workspaceId/...`;
   - извлекает `workspaceId` из path‑параметра;
   - через `WorkspaceService.HasAccess(ctx, workspaceID, userID, userRole)` проверяет:
     - принадлежит ли пользователь этому workspace (учитывая глобальную роль ADMIN);
   - при успехе кладёт `workspace_id` в Gin‑контекст (`GinWorkspaceIDKey`);
   - при ошибке:
     - 401 — если пользователь не аутентифицирован;
     - 403 — если нет доступа к workspace;
     - 400/500 — при проблемах с параметрами/доступом.

3. **`ModuleLicenseMiddleware`** (новое, в `workspace_module_permission.go`)  
   - использует `workspace_id` из контекста;
   - по `FullPath`/`Request.URL.Path` определяет код модуля (`crm` / `habits` / `projects`) через `detectModuleCode`, опираясь на текущие URL‑паттерны:
     - `/crm/...`, `/habits/...`, `/journal/...`, `/projects/...`;
   - через `WorkspaceService.GetWorkspaceModules` проверяет:
     - включен ли модуль в данном workspace (`Enabled`);
   - для не‑core модулей, при наличии аутентифицированного пользователя:
     - использует `WorkspaceService.CanEnableModuleInWorkspace` как единый gate по лицензиям;
   - при отсутствии модуля/лицензии возвращает 403.

4. **`PermissionMiddleware`** (новое, в `workspace_module_permission.go`)  
   - использует `user_id` и `workspace_id` из контекста;
   - по HTTP‑методу и полному пути (`fullPath`) вызывает `mapEndpointToPermission(method, path)`, который:
     - маппит типовые CRM/Habits/Projects эндпоинты на `(obj, act)`:
       - например, `GET /.../crm/deals` → `("crm:deal", "read")`,
       - `POST /.../habits/habits` → `("habits:habit", "create")` и т.д.;
     - пока покрывает только базовый набор; маппинг будет расширяться.
   - вызывает Casbin:
     - `sub = "user:"+userID`,
     - `dom = workspaceID`,
     - `obj`, `act` — из маппинга;
     - `enforcer.Enforce(sub, dom, obj, act)`.
   - **На текущем этапе работает только в режиме логирования**:
     - пишет в лог allow/deny и параметры (`sub`, `dom`, `obj`, `act`);
     - **не блокирует запрос** (даже если `allowed == false`).

Таким образом, сейчас:

- membership по workspace и доступ к модулям/лицензиям уже централизованы в middleware и реально влияют на доступ;
- проверка гранулярных прав через Casbin пока выполняется «в тени» (log‑only режим), что соответствует поэтапному включению из спецификаций.

### 2.4. Существующий WorkspaceMiddleware (через query/header)

Отдельно сохраняется ранее реализованный `WorkspaceMiddleware` в `workspace.go`:

- Он определяет «текущий workspace» по приоритету:
  - `?workspace_id=`, заголовок `X-Workspace-ID`, `user_preferences`, fallback на первый доступный workspace.
- Используется хендлерами/сервисами для операций, привязанных к пользовательскому «текущему workspace» (например, настройки, выпадающий список).

Новый `WorkspacePathMiddleware` не заменяет, а дополняет эту логику:

- `WorkspacePathMiddleware` — строгий gate для конкретного `:workspaceId` в URL;
- `WorkspaceMiddleware` — удобный механизм выбора/запоминания «текущего» workspace.

---

## 3. Чего ещё нет (важно для фронтенда)

На данный момент **ещё не реализовано**:

1. **PermissionService и управление ролями/правами:**
   - нет кода, который:
     - создаёт/обновляет/удаляет роли (`workspace_roles`);
     - назначает роли пользователям (`user_role_assignments`);
     - выдаёт/отзывает индивидуальные права (`user_permissions`);
     - ведёт наследование ролей (`role_inheritance`);
     - синхронизирует все эти изменения с Casbin (`AddPolicy`, `AddGroupingPolicy`, и т.д.).

2. **API для UI по ролям и правам:**
   - отсутствуют эндпоинты:
     - `GET /api/v1/workspaces/{workspaceId}/permissions/catalog`;
     - `GET/POST/PUT/DELETE /api/v1/workspaces/{workspaceId}/roles[...]`;
     - `GET/POST/DELETE /api/v1/workspaces/{workspaceId}/users/{userId}/permissions[...]`;
     - `GET /api/v1/me/permissions?workspaceId={id}`.

3. **Полное покрытие маппинга endpoint → permission:**
   - `mapEndpointToPermission` пока покрывает базовые сценарии для CRM/Habits/Projects;
   - вся матрица эндпоинтов из спецификаций ещё не перенесена в маппинг (это отдельная техническая задача).

4. **Боевой режим PermissionMiddleware:**
   - решения Casbin пока **не используются** для блокировки запросов;
   - фактический доступ по‑прежнему определяется старой логикой в хендлерах (плюс новые membership/module‑проверки в middleware).

---

## 4. Когда можно начинать реализацию на фронтенде

### 4.1. Что фронтенд может делать уже сейчас (параллельно)

На текущем этапе фронтенд может:

- **Проработать UX и макеты** для:
  - экрана управления ролями workspace (список ролей, редактирование прав);
  - таблицы «прав пользователя» (роли + индивидуальные права);
  - экрана администрирования модулей и лицензий (на основе уже существующего API и нового поведения middleware);
  - UI для отображения доступных действий на уровне сущностей (кнопки «создать/редактировать/удалить/экспортировать»).

- **Опираться на стабильные части бэкенда:**
  - текущие эндпоинты CRM/Habits/Projects уже защищены по workspace и модулям/лицензиям;
  - URLs и структура `/api/v1/workspaces/:workspaceId/...` стабильны (используются middleware).

Другими словами, **дизайн и статические компоненты** фронтенда можно делать уже сейчас, а также подготовить:

- интерфейс управления ролями/правами с заглушками под будущие эндпоинты;
- централизованный слой авторизации на фронте (hook/контекст), который будет:
  - хранить `me.permissions` и `me.roles`;
  - предоставлять хелперы `can("crm:deal:create")` и т.п.

### 4.2. Момент, когда фронтенду стоит начинать интеграцию по данным

Полноценную интеграцию с бэкендом по новой системе прав (то есть, скрытие/показ кнопок и экранов **на основе реальных прав**, а не харкода по ролям) разумно начинать, когда будут выполнены следующие условия на бэкенде:

1. **Реализован и стабилизирован PermissionService**, который:
   - управляет ролями/назначениями/индивидуальными правами;
   - синхронизирует данные с Casbin (политики `p` и `g`);
   - предоставляет метод `GetEffectivePermissions(userID, workspaceID)`.

2. **Появятся ключевые эндпоинты для фронта:**
   - `GET /api/v1/workspaces/{workspaceId}/permissions/catalog` — дерево всех возможных прав для построения UI;
   - `GET /api/v1/workspaces/{workspaceId}/roles` и `GET/POST/PUT/DELETE` для управления ролями;
   - `GET /api/v1/me/permissions?workspaceId={id}` — основной источник прав текущего пользователя.

3. **PermissionMiddleware переведён хотя бы в «мягкий боевой» режим**:
   - логика Casbin совпадает с ожидаемым доступом по логам;
   - отклонённые решения понятны и предсказуемы;
   - маппинг endpoint → permission покрывает основные маршруты, используемые фронтом.

После выполнения этих условий:

- фронтенд сможет:
  - запрашивать `me.permissions` при старте приложения;
  - использовать единый список прав (permission_catalog) для отрисовки UI конфигурации ролей;
  - тестировать реальные сценарии доступа (разные роли/права).

Практически это означает, что **реализацию фронтенда можно условно разделить на две фазы**:

- **Фаза A (уже можно начинать):**
  - дизайн и реализация компонентов UI;
  - подготовка клиентского слоя авторизации (hooks, контекст, типы прав);
  - использование текущих API для workspace/modules/лицензий.

- **Фаза B (после готовности PermissionService и ключевых эндпоинтов):**
  - подключение к `me/permissions` и `permissions/catalog`;
  - замена «хардкода ролей» на проверки конкретных прав (`permission strings`);
  - интеграционные тесты на разных ролях/наборах прав.

---

## 5. Резюме для синхронизации с фронтом

- Бэкенд уже:
  - имеет полную схему прав/ролей в БД и миграцию существующей ролевой модели;
  - интегрирован с Casbin через GORM‑адаптер;
  - применяет новые middleware для проверки членства в workspace и доступа к модулям/лицензиям;
  - в фоновом режиме собирает решения Casbin по правам через `PermissionMiddleware` (log‑only).

- Бэкенд ещё не:
  - предоставляет API для управления ролями/правами и получения эффективных прав пользователя;
  - использует Casbin для реального deny/allow по всем эндпоинтам.

- **Фронтенд:**
  - может уже сейчас начинать работу над UI и клиентским слоем авторизации, опираясь на спецификации и текущее API;
  - сможет начинать полноценную интеграцию по данным сразу после реализации PermissionService и эндпоинтов:
    - `GET /workspaces/{workspaceId}/permissions/catalog`,
    - `GET /me/permissions?workspaceId={id}`,
    - CRUD по ролям и назначениям.

Эти следующие шаги и их статус будут зафиксированы в `PERMISSION_REPORT.PART_3.md` после реализации PermissionService и соответствующих API.

