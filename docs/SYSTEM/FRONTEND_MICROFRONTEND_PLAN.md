## План микрофронтендов для текущего фронтенда

**Цель документа** — зафиксировать текущее состояние фронтенда, наметить возможные границы микрофронтендов и перечислить проблемы/рефакторинги, которые понадобятся перед миграцией.

- **Фокус**: Vue 3 SPA в `frontend/src` (Vue Router, Pinia, модульная ERP‑структура).
- **Бэкенд‑контекст**: модули `habits`, `crm`, `projects`, `notes`, `inventory`, `finance`, `hr`, лицензии и включение модулей через `modules`/`workspace_modules` (см. `ALL_MIGRATIONS_UP.sql`).
- **Аудитория**: фронтенд‑разработчики и архитекторы, которые будут проектировать и внедрять микрофронтенды.

---

## 1. Текущее состояние фронтенда (укрупнённо)

### 1.1. Слои приложения

- **`main.ts`**
  - Инициализирует тему через `initTheme()`.
  - Создаёт Vue‑приложение, подключает Pinia и роутер.
  - Настраивает глобальный обработчик 401 через `api.setUnauthorizedHandler(() => handleUnauthorized(router))`.
  - Ленивая загрузка `useAuthStore`, вызов `authStore.initAuth()`, ожидание `router.isReady()`, затем `app.mount('#app')`.

- **`app/` — «каркас» приложения**
  - `App.vue` — корневой layout (header, shell; не детализируем).
  - `router/index.ts`:
    - Описывает все статические маршруты (`/login`, `/register`, `/settings`, `/workspace-settings`, `/billing`, `/admin`, редиректы и т.п.).
    - Подмешивает маршруты модулей на основе конфигурации `app/modules/config.ts`.
    - Подключает глобальные guard‑ы (`authGuard` и `beforeEnter` с проверкой прав/модулей).
  - `modules/config.ts`:
    - Декларативный реестр модулей (`habits`, `crm`, `projects`, `notes`, `inventory`, `finance`, `hr`).
    - Для каждого модуля хранит:
      - `id`, `label`, `icon`, `basePath`, `permissions`.
      - Массив маршрутов (путь, компонент, meta, permissions).
    - Утилиты:
      - `getModuleByPath(path)` — находит модуль по URL.
      - `getAvailableModules(enabledModuleCodes, hasPermission)` — фильтрует модули по включённости и правам.
      - `getAvailableModuleRoutes(module, hasPermission)` — фильтрация маршрутов модуля по правам/мета.

- **`pages/` — страничный слой (route level)**
  - Группы страниц по доменам:
    - Habits: `pages/habits/**` (`dashboard`, `list`, `calendar`, `journal`).
    - CRM: `pages/crm/**` (контакты, компании, сделки, пайплайны, детали).
    - Projects: `pages/projects/**` (список, детали, project‑CRM).
    - Notes: `pages/notes/**` (список заметок).
    - Settings/Admin/Billing: `pages/settings/**`, `pages/workspace-settings/**`, `pages/workspace-modules/**`, `pages/billing/**`, `pages/admin/**`.
    - Общие: `pages/home/**`, `pages/dashboard/**`, `pages/login/**`, `pages/register/**`, `pages/calendar/**`, `pages/journal/**`, `pages/module-stub/**`.

- **`features/` — фичи/сценарии**
  - Доменные фичи:
    - Habits/Journal/Calendar/Notes: `features/habit`, `features/journal`, `features/calendar`, `features/notes`.
    - CRM: `features/contacts`, `features/deals`, `features/companies`, `features/activity`.
    - Projects: `features/projects`.
  - Системные фичи:
    - Auth: `features/auth`.
    - Permissions & Roles: `features/permissions`, `features/roles`, `features/user-permissions`, `features/members`.
    - Workspace/Settings/Admin: `features/workspace`, `features/admin`, `features/dashboard`.
  - Типовой паттерн:
    - `ui/` — сложные компоненты и виджеты (таблицы, панели, модалки, тулбары).
    - `model/` — композиционные хуки (`use-*-page`, `use-*-table-state`, `use-*-editor`).
    - `config/` — конфигурации таблиц, маппинги.

- **`entities/` — слой сущностей (выравнен с бэкендом)**
  - CRM: `entities/contact`, `entities/company`, `entities/deal`, `entities/activity`.
  - Habits/Journal: `entities/habit`, `entities/journal`.
  - Projects: `entities/project`.
  - Core: `entities/user`, `entities/workspace`, `entities/role`, `entities/permission`, `entities/assignment`.
  - Внутри сущности:
    - `types/*.ts` — типы DTO и доменных моделей.
    - `api/*-service.ts` — сервисы доступа к API на базе общего клиента `api`.
    - `model/*-store.ts` — Pinia‑хранилища там, где нужен shared‑стейт (например, `workspace-store`, `habit-store`, `user-store`).

- **`shared/` — общий слой (дизайн‑система, API, утилиты)**
  - `shared/ui/*` — дизайн‑система:
    - Базовые формы: `Input`, `Select`, `Checkbox`, `Radio`, `DatePicker`, `FormField` и т.д.
    - Комплексы данных: `DataTable`, `Pagination`, `KanbanBoard`, `Dnd`, `StatsCards`.
    - Layout/UX: `Modal`, `Drawer`, `PageFilters`, `NavLink`, `Card`, `EmptyState`, `Tooltip`, `Dropdown`, `ConfirmModal`, `Badge`, `Button`, `Spinner`.
    - Иконки модулей и сущностей: `shared/ui/icon/*`.
    - `UI.md` — документация по UI‑киту.
  - `shared/api/*`:
    - `client.ts` — singleton‑клиент (`api`) на Axios:
      - Настраивается в `main.ts` (401 handler) и в `workspace-store` (`setWorkspaceId` → заголовок `X-Workspace-ID`).
      - Поддерживает реальный и моковый бэкенд через флаг `VITE_USE_MOCK_API`.
    - `endpoints.ts` — перечисление REST‑эндпоинтов.
  - `shared/lib/*`:
    - Тема (`use-theme`), даты, модалки, дебаунсы, хранение в `localStorage`, speech‑recognition, и пр.

- **`widgets/`**
  - Важно для shell:
    - `widgets/header/ui/WorkspaceSwitcher.vue` — переключатель workspace‑ов, завязан на `workspace-store` и модалки создания workspace.
    - `widgets/header/ui/ProfileDropdown.vue` — меню пользователя, использует `user-store` и `auth-store`.

### 1.2. Стейт и авторизация

- **Pinia‑сторы ядра**
  - `features/auth/model/auth-store`:
    - Отвечает за логин/регистрацию/выход, хранит `effectivePermissions`, инициирует загрузку `/me/permissions`.
  - `entities/workspace/model/workspace-store`:
    - Хранит список workspace‑ов, текущий workspace, включённые модули и лицензии.
    - При переключении workspace:
      - Меняет заголовок `X-Workspace-ID` в `api`.
      - Заново грузит модули и права.
      - **Побочный эффект**: динамически импортирует `useHabitStore` и перезагружает привычки.
  - `entities/user/model/user-store`:
    - Хранит текущего пользователя и базовую информацию.

- **Guard‑ы и права**
  - `features/auth/lib/guards`:
    - `authGuard` — пропускает публичные маршруты, для остальных проверяет авторизацию и наличие workspace‑ов.
    - `requireAdmin`, `requireOwnerOrAdmin` — guards для админских страниц.
  - `entities/workspace/lib/permissions`:
    - `requirePermission`, `requireModuleEnabled` для маршрутов модулей.
  - `features/permissions/model/use-permissions` + `PermissionGuard.vue`:
    - Унифицированные проверки `can/canAny/canAll` по строкам `module:entity:action`.

---

## 2. Логические домены и кандидаты на микрофронтенды

Домены во фронтенде уже достаточно хорошо совпадают с модулями на бэкенде и таблицами из `ALL_MIGRATIONS_UP.sql`.

### 2.1. Shell / Workspace / Core

- **Ответственность**
  - Глобальный layout и навигация.
  - Авторизация, профиль пользователя, загрузка и переключение workspace‑ов.
  - Знание того, какие модули включены (`workspace_modules`, `user_module_licenses`).
  - Управление ролями и правами в workspace.
  - Админка и биллинг.
- **Основные страницы**
  - `/`, `/home`, общий `/dashboard`.
  - `/login`, `/register`.
  - `/settings`, `/settings/members`.
  - `/workspace-settings`, `/workspace-settings/roles`, `/workspace-modules`.
  - `/billing`, `/admin`.
- **Ключевые модули фронтенда**
  - `main.ts`, `app/router`, `app/modules/config`.
  - `features/auth`, `entities/user`.
  - `entities/workspace`, `features/permissions`, `features/roles`, `features/members`, `features/user-permissions`, `features/admin`, `features/workspace`.
  - `widgets/header` (WorkspaceSwitcher, ProfileDropdown).

### 2.2. Habits (Привычки и журнал)

- **Бэкенд‑база**
  - Таблицы: `habits`, `habit_completions`, `habit_history`, `habit_versions`, `journal_entries`.
  - Права: `habits:habit:*`, `habits:journal:*` (см. `permission_catalog` в миграциях).
- **Фронтенд**
  - Маршруты: `/habits/dashboard`, `/habits/list`, `/habits/calendar`, `/habits/journal`.
  - `entities/habit`, `entities/journal` (типы, сервисы, сторы).
  - `features/habit`, `features/journal`, `features/calendar`, часть `features/dashboard`.
  - Использует `workspace-store` для контекста и `shared/ui` для отрисовки.

### 2.3. CRM (Контакты, компании, сделки)

- **Бэкенд‑база**
  - Таблицы: `crm_contacts`, `crm_contact_phones`, `crm_contact_emails`, `crm_companies`, `crm_company_contacts`, `crm_pipelines`, `crm_stages`, `crm_deals`, `crm_activities`, `crm_activity_files`, `crm_activity_reminders`.
  - Права: `crm:deal:*`, `crm:contact:*`, `crm:company:*`, `crm:activity:*`, `crm:pipeline:manage`, `crm:export:deals`.
- **Фронтенд**
  - Маршруты: `/crm/contacts`, `/crm/contacts/:id`, `/crm/companies`, `/crm/companies/:id`, `/crm/deals`, `/crm/deals/:id`, `/crm/pipelines`.
  - `entities/contact`, `entities/company`, `entities/deal`, `entities/activity`.
  - `features/contacts`, `features/companies`, `features/deals`, `features/activity`.
  - Плотно использует дизайн‑систему (таблицы, канбан, фильтры, модалки).
  - Локальные композиционные хуки для страниц (`use-*-page`, `use-*-table-state`), Pinia в основном на уровне сущностей.

### 2.4. Projects (Проекты)

- **Бэкенд‑база**
  - Таблицы: `projects`, `project_entities` (связки с другими сущностями, в т.ч. CRM).
  - Права: `projects:project:*`, `projects:entity:attach/detach`.
- **Фронтенд**
  - Маршруты: `/projects`, `/projects/:id`, `/projects/:id/crm`.
  - `entities/project`, `features/projects`.
  - Сильные связи с CRM:
    - Через `project_entities` и API для подсчёта/получения ID связанных `crm_*` сущностей.

### 2.5. Notes (Заметки)

- **Бэкенд‑база**
  - Таблица: `notes`.
  - Модуль `notes` в `modules` и правах.
- **Фронтенд**
  - Маршруты: `/notes/list`.
  - `features/notes` + соответствующий `entities`‑слой.
  - Мало связей с другими доменами, кроме workspace и прав.
  - Хороший кандидат для пилотного микрофронтенда.

### 2.6. Admin/Settings

- Отвечает за:
  - Управление ролями и правами (`workspace_roles`, `user_role_assignments`, `user_permissions`, `role_inheritance`, `permission_catalog`).
  - Управление модулями и лицензиями (`workspace_modules`, `user_module_licenses`).
  - Общие настройки workspace и пользователей.
- Фронтенд:
  - Страницы настроек и админки.
  - Фичи `permissions`, `roles`, `members`, `user-permissions`, `workspace`, `admin`.

### 2.7. Auth / Onboarding

- Отдельный мини‑домен:
  - `/login`, `/register`, возможно `/onboarding`.
  - Работает на `features/auth`, `entities/user`, общих UI‑компонентах.
  - Может быть вынесен в отдельный микрофронтенд, который после логина редиректит в Shell.

---

## 3. Предлагаемое разбиение на микрофронтенды

Ниже — не финальный дизайн, а целевая модель, к которой можно двигаться.

### 3.1. Host / Shell MFE

- **Задачи**
  - Ядро SPA: `main.ts`, инициализация темы, создание Vue‑приложения.
  - Глобальный роутер и композиция микрофронтендов.
  - Auth/Workspace SDK:
    - Хранение и выдача токенов.
    - Запрос `/me/permissions`, вычисление `effectivePermissions`.
    - Хранение и переключение текущего workspace, загрузка лицензий и модулей.
    - Настройка `api` и заголовка `X-Workspace-ID`.
  - Общие страницы Shell:
    - Главная, общий дашборд.
    - Настройки профиля (минимальный UI).
  - Общий дизайн‑системный слой:
    - `shared/ui`, `shared/lib`, `shared/api` — экспортируются как разделяемые singletons.

- **Интеграция микрофронтендов**
  - На уровне роутера:
    - Для каждого домена — «маунт‑точка»:
      - `/habits/*` → Habits MFE.
      - `/crm/*` → CRM MFE.
      - `/projects/*` → Projects MFE.
      - `/notes/*` → Notes MFE.
    - Для settings/admin либо:
      - Оставить внутри Shell.
      - Либо вынести как отдельный Admin MFE, маунтящийся на `/settings/*`, `/workspace-settings/*`, `/billing`, `/admin`.
  - На уровне shared‑модулей:
    - Shell публикует Auth/Workspace/Permissions SDK (например, псевдопакеты `@app/auth`, `@app/workspace`, `@app/permissions`), которые микрофронтенды импортируют как обычные модули, но физически они приходят из Host.

### 3.2. Habits MFE

- **Область ответственности**
  - Все маршруты `/habits/*`.
  - Habit‑и, их выполнение, календарь и журнал.
- **Интеграция с Host**
  - Использует:
    - Auth SDK для проверки авторизации и получения `effectivePermissions`.
    - Workspace SDK для текущего `workspaceId`.
    - Общий `api` (shared singleton) для HTTP‑запросов.
  - Не должен:
    - Инициировать хранение/переключение workspace.
    - Влезать в чужие домены напрямую.
- **Требуемые изменения**
  - Вынести зависимости на `workspace-store` в слой SDK (см. план рефакторинга ниже).
  - Убрать из `workspace-store` прямое знание о `habit-store` (сейчас при смене workspace вызывается `habitStore.fetchHabits()`).

### 3.3. CRM MFE

- **Область ответственности**
  - `/crm/*` маршруты: контакты, компании, сделки, пайплайны, активности.
  - CRM‑виджеты, которые могут использоваться в других доменах (например, выбор контакта/компании, attach‑модалки, ленты активности).
- **Интеграция с Host**
  - Точно так же использует Auth/Workspace SDK и `api`.
  - Может публиковать «remote компоненты/хуки» для интеграции с Projects MFE (например, «CRM блок внутри проекта»).

### 3.4. Projects MFE

- **Область ответственности**
  - `/projects/*` маршруты.
  - Управление проектами и их связями с сущностями других модулей (через `project_entities`).
- **Специфическая сложность**
  - Сильная связь с CRM (счётчики, списки ID, вьюхи project‑CRM).
  - Нужно перейти от «Projects знает внутренние детали CRM» к «Projects договаривается с CRM по явному интерфейсу».

### 3.5. Notes MFE

- **Область ответственности**
  - `/notes/*` маршруты.
  - CRUD заметок.
- **Причина выбрать как пилот**
  - Простая модель данных (одна сущность `notes`).
  - Минимум кросс‑доменных связей.
  - Быстро даёт опыт сборки/деплоя микрофронтенда с малым риском.

### 3.6. Admin/Settings MFE (опционально отдельный)

- **Область ответственности**
  - Роли, права, участники, модули, биллинг.
  - Маршруты `/settings/*`, `/workspace-settings/*`, `/workspace-modules`, `/billing`, `/admin`.
- **Интеграция**
  - Использует те же shared‑SDK, что и доменные MFE.
  - Может жить как часть Shell, если не требуется отдельный цикл релизов.

### 3.7. Auth MFE (опционально)

- **Область ответственности**
  - `/login`, `/register`, возможно recovery/onboarding.
- **Причины выделить**
  - Отдельный деплой и статика могут упростить сценарии SSO, брендинг и обслуживание публичных screens.

---

## 4. Ключевые проблемы и точки рефакторинга перед миграцией

Ниже перечислены наиболее важные технические «узлы», которые придётся переработать.

### 4.1. Глобальный singleton `api` и настройка 401/Workspace

- **Что есть сейчас**
  - `shared/api/client.ts` экспортирует singleton `api`.
  - В `main.ts` настраивается `setUnauthorizedHandler(handleUnauthorized(router))`.
  - В `workspace-store` вызывается `api.setWorkspaceId(workspaceId)` для установки заголовка `X-Workspace-ID`.
  - Все сервисы (`*Service`) импортируют `api` напрямую.
- **Проблема для микрофронтендов**
  - При независимом билде MFE нужно гарантировать:
    - Одинаковую версию клиента.
    - Единообразную конфигурацию (интерсепторы, baseURL, заголовки).
  - Если каждый MFE будет создавать свой клиент — получим дублирующуюся конфигурацию и неконсистентное поведение.
- **План рефакторинга**
  - Ввести «API SDK»:
    - Единый модуль (например, `@app/api`), который:
      - Инициализируется только в Host (baseURL, интерсепторы, 401‑обработка).
      - Экспортирует уже готовый `api` как singleton для всех MFE.
  - В коде заменить:
    - Прямые импорты `@/shared/api` на импорты из SDK.
  - Рассмотреть прокидывание `workspaceId` в вызовы сервисов как аргумента, а не скрытого заголовка, **или** оставить заголовок, но обеспечить, что только Host меняет его.

### 4.2. Централизованный роутер и модульные маршруты

- **Что есть сейчас**
  - `app/router/index.ts`:
    - Знает про все маршруты (включая модульные).
    - На основе `modules` регистрирует маршруты Habits/CRM/Projects/Notes и пр.
  - `modules/config.ts` одновременно:
    - Хранит метаданные модулей (icon, label, permissions).
    - Хранит их маршруты и компоненты (lazy imports).
- **Проблема для микрофронтендов**
  - В MFE‑архитектуре домен должен сам владеть своими маршрутами.
  - Host должен только уметь:
    - Делегировать определённый префикс (`/crm/*`) конкретному микрофронтенду.
    - Не знать о конкретных page‑компонентах домена.
- **План рефакторинга**
  - Разделить конфиг модулей:
    - **Метаданные модуля** (id, label, icon, basePath, permissions) — остаются в Host (управление модулями, меню, лицензирование).
    - **Реестр маршрутов/компонентов** — уезжает в каждое MFE.
  - В роутере Host:
    - Заменить детализированные маршруты доменов на:
      - «маунт‑роуты» вида `/crm/:pathMatch(.*)*`, `/habits/:pathMatch(.*)*` и т.п., внутри которых MFE монтирует свой собственный роутер.

### 4.3. Связь `workspace-store` → `habit-store`

- **Что есть сейчас**
  - При смене workspace `workspace-store` динамически импортирует `useHabitStore` и перезагружает данные привычек.
- **Почему это проблема**
  - Ядро Shell таким образом «знает» про конкретный доменный стор (`habit-store`).
  - В мире микрофронтендов Shell не должен иметь прямых зависимостей от кода доменов.
- **План рефакторинга**
  - Заменить этот вызов на событийную модель:
    - При смене workspace Shell издаёт событие (`workspaceChanged`) через:
      - глобальный event‑bus, или
      - shared‑SDK (простой Pub/Sub), или
      - контекст (Vue provide/inject).
    - Habits MFE подписывается на это событие и сам решает, когда обновлять свои сторы.
  - Либо:
    - Вообще убрать автоперезагрузку привычек на уровне core и делать её только внутри Habits MFE на входе в его маршруты.

### 4.4. Кросс‑доменные зависимости Projects ↔ CRM

- **Что есть сейчас**
  - Projects:
    - Знает строковые типы сущностей CRM (`'crm_contact'`, `'crm_company'`, `'crm_deal'`).
    - Вызывает `projectService.listEntityIds` для этих типов.
  - Project‑CRM страницы используют CRM‑сущности и UI‑компоненты напрямую.
- **Риски для микрофронтендов**
  - Projects MFE окажется жёстко завязан на внутренние детали CRM MFE:
    - Это мешает независимым релизам и тестированию.
  - Любое изменение в названиях типов или структуре CRM потребует правок в Projects.
- **План рефакторинга**
  - Ввести явный контракт между Projects и CRM:
    - На уровне бэкенда:
      - API, где Projects запрашивает агрегированные данные (например, «счётчики CRM по проекту») без знания внутренних типов CRM.
    - На уровне фронтенда:
      - CRM публикует remote‑компоненты (например, `<ProjectCrmView projectId="..." />`), которые Projects просто вставляет.
  - Минимизировать использование «сырых» `crm_*` строк в Projects.

### 4.5. Auth/Permissions SDK

- **Что есть сейчас**
  - Auth guard и проверки прав разбросаны между:
    - `features/auth` (guard, redirect‑логика).
    - `entities/workspace` (права модулей, requirePermission).
    - `features/permissions` (hooks и компоненты для UI).
  - Используются напрямую из различных слоёв.
- **Проблема**
  - В микрофронтендах нужен **единый**, чётко определённый API:
    - Как запросить `effectivePermissions`.
    - Как проверять `can/canAny/canAll`.
    - Как проверять `requireModuleEnabled` и `requireOwnerOrAdmin`.
- **План рефакторинга**
  - Сформировать из существующего кода «Auth/Permissions SDK»:
    - Пакет, который экспортирует:
      - `useAuth`, `useWorkspace`, `usePermissions`.
      - `authGuard`, `requireAdmin`, `requireOwnerOrAdmin`, `requirePermission`, `requireModuleEnabled`.
    - Минимум внутренних зависимостей (никаких прямых импортов страниц/конкретных виджетов).
  - Запретить прямые импорты низкоуровневых хранилищ и утилит, кроме через SDK.

### 4.6. Общий дизайн‑кит и стили

- **Что есть сейчас**
  - Единый `shared/ui` и общие стили в `styles/main.css` + Tailwind‑подобная конфигурация.
  - `initTheme()` вызывается один раз в `main.ts`.
- **Проблема**
  - При раздельном билде и деплое MFE:
    - Нужно избегать дублирования стилей и конфликтов версий UI‑компонентов.
- **План**
  - Зафиксировать публичный API дизайн‑системы:
    - Всё, что должны импортировать домены, экспортировать только через `shared/ui/index.ts`.
  - При конфигурации модульной федерации:
    - Делать `shared/ui`, `shared/lib`, `shared/api` разделяемыми библиотеками (singleton).
  - Убедиться, что `initTheme()`:
    - Идемпотентен.
    - Вызывается только Host‑приложением.

### 4.7. Предположение об одном активном workspace

- **Что есть сейчас**
  - Большинство доменных хуков и сторов вызывают `useWorkspaceStore()` и используют `currentWorkspace` неявно.
- **Проблема**
  - В микрофронтендах допустим сценарий, когда:
    - MFE может быть замонтирован/размонтирован независимо от другого.
    - Нужно гарантировать, что к моменту инициализации MFE `currentWorkspace` уже определён и согласован.
- **План**
  - Стандартизировать контракт:
    - Host гарантирует, что:
      - При маунте любого доменного MFE `workspaceId` уже известен.
      - SDK `useWorkspace` возвращает готовое состояние (или явный loading/error).
  - По возможности передавать `workspaceId` как параметр в доменные хуки/запросы, а не всегда читать его из стора.

---

## 5. Поэтапный план перехода

Ниже — рекомендуемая последовательность шагов, которая даёт быстрый выигрыш и одновременно готовит проект к возможной микрофронтенд‑архитектуре.

### Шаг 1. Выделить Core SDK (Auth, Workspace, API, Permissions)

- Объединить:
  - Логику авторизации (`auth-store`, `authGuard`, `handleUnauthorized`).
  - Логику workspace (`workspace-store`, модули, лицензии).
  - Логику прав (`usePermissions`, `PermissionGuard`, `requirePermission` и т.д.).
  - Клиент `api`.
- В отдельный «ядровой» пакет/директорию:
  - Минимизировать зависимости от конкретных страниц и фич.
  - Оставить только общие модели и утилиты.

### Шаг 2. Убрать жёсткую связь ядра с доменными сторами

- В первую очередь:
  - Удалить прямую зависимость `workspace-store` от `habit-store`.
  - Заменить на событие/подписку или внутреннюю логику внутри Habits.
- Проверить аналогичные связи:
  - Если Shell где‑то ещё тянет доменные фичи — вынести это в SDK или события.

### Шаг 3. Разделить модульный конфиг и маршруты

- Разбить `app/modules/config.ts`:
  - Чистая мета‑информация о модуле (id, label, icon, basePath, permissions, флаг `isCore`).
  - Доменные маршруты и компоненты — перенести ближе к соответствующим доменам (и в будущем — в их микрофронтенды).
- В роутере:
  - Ввести концепцию «маунт‑роутов» для модулей.

### Шаг 4. Выделить Notes как пробный микрофронтенд

- Определить минимальный scope:
  - Страницы `/notes/*`.
  - `features/notes` + соответствующие `entities`.
- Протестировать:
  - Сборку, деплой и интеграцию Notes как отдельного бандла.
  - Работу с общим `api`, Auth/Workspace SDK, UI‑китом.

### Шаг 5. Постепенно выводить крупные домены (Habits, CRM, Projects)

- После успешного пилота:
  - Повторить паттерн для Habits и CRM (как двух основных доменов).
  - Дополнительно:
    - Для Projects заранее ослабить связь с CRM (через явные контракты/remote‑компоненты).

### Шаг 6. Решить судьбу Admin/Settings и Auth

- В зависимости от продуктовых требований:
  - Либо оставить Admin/Settings и Auth внутри Host, но с чётким SDK.
  - Либо вынести в самостоятельные MFE, если требуется отдельный жизненный цикл релизов.

---

## 6. Риски и рекомендации

- **Сложность инфраструктуры**:
  - Микрофронтенды имеют ощутимый overhead по настройке CI/CD, оркестрации версий, мониторингу.
  - Имеет смысл двигаться к ним постепенно, начиная с одного‑двух доменов.
- **Согласованность UI и прав доступа**:
  - UI‑кит и модель прав (`module:entity:action`) должны оставаться едиными:
    - Любое изменение в `permission_catalog` и `Perm`‑типа должно быть синхронизировано.
  - Рекомендуется поддерживать единый типовой пакет с типами прав, как и сделано сейчас для ролей/permissions.
- **Backward‑совместимость**:
  - На время миграции придётся поддерживать:
    - Частично монолитный роутер.
    - И новый слой, маунтящий MFE.
  - Важно прописать стратегию «переключения» домена целиком на MFE, чтобы избежать двойной логики.

Этот документ описывает целевую картину и основные технические шаги. Перед началом фактической реализации микрофронтендов рекомендуется дополнительно создать короткий RFC по выбору конкретного стека интеграции (Module Federation, single-spa, iframes и т.д.) и описать требования к деплою и мониторингу.

