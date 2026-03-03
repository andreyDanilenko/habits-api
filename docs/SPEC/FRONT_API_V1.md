## АКТУАЛЬНЫЙ СТАТУС ИСПОЛЬЗОВАНИЯ API НА ФРОНТЕНДЕ

### На основе `frontend/src/shared/api/endpoints.ts`

| Дата | Март 2026 | Версия | 1.0 |
|------|-----------|--------|-----|

---

## 1. Общий анализ

Фронтенд использует единый объект `API_ENDPOINTS` с функциями-генераторами URL на базе `apiV1 = '/api/v1'`.
Все пути соответствуют backend-спецификации `API_ERP_V1.md` (разделы Auth, Workspace, Master, Projects, Habits, Journal, Admin, CRM).

### 1.1. Распределение по модулям (frontend endpoints)

Здесь считаются **фронтовые функции** (builder’ы URL), а не фактические HTTP-роуты бэкенда:

| Модуль          | Кол-во фронтовых функций | Статус покрытия |
|-----------------|--------------------------|-----------------|
| Auth            | 5                        | ✅ Полностью     |
| Workspace       | 15                       | ✅ Полностью     |
| Master Data     | 4                        | ✅ Полностью     |
| Notes/Journal   | 4                        | ✅ Полностью     |
| Admin           | 4                        | ✅ Полностью     |
| CRM             | 19                       | ✅ Основное + pipelines/stages/links |
| Projects        | 5                        | ✅ Полностью     |
| Habits/Journal  | 7                        | ✅ Полностью     |

**ИТОГО:** фронтенд имеет доступ ко всем основным backend-эндпоинтам; опциональные расширения (batch-операции в CRM и пр.) пока сознательно не заведены.

---

## 2. Детальный анализ по модулям (frontend)

### 2.1. Auth

**Источник:** `API_ENDPOINTS.AUTH`

- `LOGIN`   → `POST /api/v1/auth/login`
- `REGISTER`→ `POST /api/v1/auth/register`
- `LOGOUT`  → `POST /api/v1/auth/logout`
- `REFRESH` → `POST /api/v1/auth/refresh`
- `ME`      → `GET  /api/v1/auth/me`

**Статус:** ✅ Полное соответствие backend-спеке.

---

### 2.2. Workspace, Notes, Journal, Master Data

**Источник:** `API_ENDPOINTS.WORKSPACE`

**Workspace:**

- `BASE`             → `/api/v1/workspaces`
- `CURRENT`          → `/api/v1/workspaces/current`
- `MY_LICENSES`      → `/api/v1/workspaces/me/module-licenses`
- `MEMBERS(id)`      → `/api/v1/workspaces/:workspaceId/members`
- `SWITCH(id)`       → `/api/v1/workspaces/:workspaceId/switch`

**Модули workspace:**

- `MODULES(id)`      → `/api/v1/workspaces/:workspaceId/modules`
- `MODULE(id, code)` → `/api/v1/workspaces/:workspaceId/modules/:moduleCode`

**Notes:**

- `NOTES(id)`        → `/api/v1/workspaces/:workspaceId/notes`
- `NOTE(id, noteId)` → `/api/v1/workspaces/:workspaceId/notes/:noteId`

**Journal:**

- `JOURNAL(id)`              → `/api/v1/workspaces/:workspaceId/journal`
- `JOURNAL_ENTRY(id, entry)` → `/api/v1/workspaces/:workspaceId/journal/:entryId`

**Master Data:**

- `CURRENCIES(id)`       → `/api/v1/workspaces/:workspaceId/currencies`
- `CURRENCY(id, curId)`  → `/api/v1/workspaces/:workspaceId/currencies/:id`
- `COUNTERPARTIES(id)`   → `/api/v1/workspaces/:workspaceId/counterparties`
- `COUNTERPARTY(id, cp)` → `/api/v1/workspaces/:workspaceId/counterparties/:id`

**Статус:** ✅ Полное покрытие backend-эндпоинтов.

---

### 2.3. Admin

**Источник:** `API_ENDPOINTS.ADMIN`

- `WORKSPACES`         → `GET    /api/v1/admin/workspaces`
- `USERS`              → `GET    /api/v1/admin/users`
- `USER(id)`           → `GET/DELETE /api/v1/admin/users/:id` (по HTTP-методу)
- `USER_LICENSES(id)`  → `POST   /api/v1/admin/users/:userId/licenses`

**Статус:** ✅ Полностью соответствует backend Admin API.

---

### 2.4. CRM

**Источник:** `API_ENDPOINTS.CRM`

#### Контакты

- `CONTACTS(workspaceId)` → `/api/v1/workspaces/:workspaceId/contacts`
- `CONTACT(workspaceId, id)` → `/api/v1/workspaces/:workspaceId/contacts/:id`

Покрытие: список, деталка, создание, обновление, удаление (через методы HTTP).

#### Компании

- `COMPANIES(workspaceId)` → `/api/v1/workspaces/:workspaceId/companies`
- `COMPANY(workspaceId, id)` → `/api/v1/workspaces/:workspaceId/companies/:id`
- `COMPANY_ATTACH_CONTACT(workspaceId, companyId, contactId)` →
  `/api/v1/workspaces/:workspaceId/companies/:companyId/contacts/:contactId` (POST)
- `COMPANY_DETACH_CONTACT(workspaceId, companyId, contactId)` →
  `/api/v1/workspaces/:workspaceId/companies/:companyId/contacts/:contactId` (DELETE)

Покрытие:
- список и карточка компании,
- создание/редактирование/удаление,
- загрузка привязанных контактов и сделок компании,
- UI для привязки существующего контакта к компании через модалку выбора.

#### Сделки

- `DEALS(workspaceId)` → `/api/v1/workspaces/:workspaceId/deals`
- `DEAL(workspaceId, id)` → `/api/v1/workspaces/:workspaceId/deals/:id`

Покрытие: список, деталка, создание, обновление, удаление.

#### Воронки и этапы (Pipelines & Stages)

- Pipelines:
  - `PIPELINES(workspaceId)` → `/api/v1/workspaces/:workspaceId/pipelines`
  - `PIPELINE(workspaceId, pipelineId)` → `/api/v1/workspaces/:workspaceId/pipelines/:pipelineId`

- Stages:
  - `PIPELINE_STAGES(workspaceId, pipelineId)` →
    `/api/v1/workspaces/:workspaceId/pipelines/:pipelineId/stages`
  - `PIPELINE_STAGE(workspaceId, pipelineId, stageId)` →
    `/api/v1/workspaces/:workspaceId/pipelines/:pipelineId/stages/:stageId`
  - `PIPELINE_STAGES_REORDER(workspaceId, pipelineId)` →
    `/api/v1/workspaces/:workspaceId/pipelines/:pipelineId/stages/reorder`

Покрытие: полный CRUD по воронкам и этапам, включая `reorder`, 1:1 с backend `API_ERP_V1.md`.

#### Активности (CRM Activity Feed)

- `ACTIVITIES(workspaceId)` → `/api/v1/workspaces/:workspaceId/activities`
- `ACTIVITY(workspaceId, id)` → `/api/v1/workspaces/:workspaceId/activities/:id`
- `ACTIVITY_IMPORTANT(workspaceId, id)` →
  `/api/v1/workspaces/:workspaceId/activities/:id/important`

Покрытие:
- список/фильтрация активностей (GET),
- создание заметок/звонков (POST),
- обновление/удаление (PUT/DELETE),
- переключение важности (POST `/important`).

#### CRM: чего ещё нет на фронтенде

- Массовые операции (batch):
  - `POST /contacts/batch/delete`
  - `POST /contacts/batch/update`
  - `POST /deals/batch/move`
- Реализация вкладки "Задачи" в карточке контакта (ждёт модуль Tasks).
- Будущие фичи (авто-активности, расширенная валидация и т.п.) — пока не реализованы на бэке, поэтому endpoints во фронт сознательно не добавлены.

---

### 2.5. Projects

**Источник:** `API_ENDPOINTS.PROJECTS`

- `LIST(workspaceId)` → `/api/v1/workspaces/:workspaceId/projects`
- `DETAIL(workspaceId, projectId)` → `/api/v1/workspaces/:workspaceId/projects/:projectId`
- `ENTITIES(workspaceId, projectId)` →
  `/api/v1/workspaces/:workspaceId/projects/:projectId/entities`
- `DETACH_ENTITY(workspaceId, projectId, entityType, entityId)` →
  `/api/v1/workspaces/:workspaceId/projects/:projectId/entities/:entityType/:entityId`
- `BY_ENTITY(workspaceId, entityType, entityId)` →
  `/api/v1/workspaces/:workspaceId/entities/:entityType/:entityId/projects`

**Статус:** ✅ Полное покрытие Projects API.

---

### 2.6. Habits & Journal

**Источник:** `API_ENDPOINTS.HABITS`

- `BASE(workspaceId)`        → `/api/v1/workspaces/:workspaceId/habits`
- `DETAIL(workspaceId, id)`  → `/api/v1/workspaces/:workspaceId/habits/:habitsId`
- `COMPLETE(workspaceId, id)`→ `/api/v1/workspaces/:workspaceId/habits/:habitsId/complete`
- `TOGGLE(workspaceId, id)`  → `/api/v1/workspaces/:workspaceId/habits/:habitsId/toggle`
- `STATS(workspaceId, id)`   → `/api/v1/workspaces/:workspaceId/habits/:habitsId/stats`
- `COMPLETIONS(workspaceId)` → `/api/v1/workspaces/:workspaceId/habits/completions`
- `CALENDAR(workspaceId)`    → `/api/v1/workspaces/:workspaceId/habits/calendar`

**Статус:** ✅ Полное покрытие backend Habits/Journal API.

---

## 3. Сводка по покрытию API на фронтенде

| Компонент   | Backend-статус (см. `API_ERP_V1.md`) | Frontend-покрытие | Комментарий |
|------------|----------------------------------------|-------------------|-------------|
| Auth       | ✅ 5/5                                | ✅ Полностью      | Все эндпоинты заведены |
| Workspace  | ✅ 12/12                              | ✅ Полностью      | Включая модули, участников, switch |
| Master     | ✅ 10/10                              | ✅ Полностью      | Валюты и контрагенты |
| Notes      | ✅ 5/5                                | ✅ Полностью      | CRUD заметок |
| Journal    | ✅ 5/5                                | ✅ Полностью      | Через WORKSPACE/JOURNAL* |
| Habits     | ✅ 12/12                              | ✅ Полностью      | Habits + календарь + completions |
| Projects   | ✅ 8/8                                | ✅ Полностью      | CRUD + связи сущностей |
| Admin      | ✅ 4/4                                | ✅ Полностью      | Workspaces/users/licenses |
| CRM        | ✅ 23/23                              | ✅ Основное       | Полный CRUD контактов/компаний/сделок, pipelines/stages, активности; batch-операции ещё не заведены |

---

## 4. Вывод

Фронтенд полностью покрывает все основные backend-эндпоинты, описанные в `API_ERP_V1.md`, включая обновлённый CRM (pipelines/stages).

Оставшиеся пробелы касаются только **опциональных** возможностей (массовые операции в CRM и будущие фичи), которые пока не реализованы на бэке и не требуются для базового продакшн-сценария.

