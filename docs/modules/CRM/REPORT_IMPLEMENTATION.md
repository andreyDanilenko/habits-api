# Отчёт о состоянии реализации CRM относительно плана

**Дата:** 2026-03-01  

## 1. Соответствие архитектуре (PROJECT.md)

| Принцип | Статус | Комментарий |
|--------|--------|-------------|
| **Изоляция по workspace** | ✅ | Все запросы фильтруются по `workspace_id`; доступ проверяется через `HasAccess` перед операциями. |
| **Без FK между модулями** | ✅ | Внутри CRM используются FK (contacts ↔ companies, deals ↔ pipelines/stages). С ядром (users, workspaces) связи по UUID без FK в БД — по плану. |
| **RLS (Row Level Security)** | ⏳ | Не реализовано. В плане — `SET app.current_workspace_id` и RLS на таблицах. Можно добавить на следующем этапе. |
| **Soft Links / task_links** | — | Относится к модулю Tasks; для ядра CRM не требуется. |
| **Сквозные сущности (Core)** | ✅ | CRM опирается на workspace и user (owner_id, created_by); прямых FK на core-таблицы нет. |
| **Связь через project_id** | ⏳ | В SPEC_BACK_1 и текущей схеме БД поля `project_id` в сделках/контактах нет. При появлении модуля Projects можно добавить. |

**Вывод:** Текущая реализация соответствует заложенной модульности и изоляции по workspace; RLS и project_id — кандидаты на следующий этап.

---

## 2. Соответствие SPEC_BACK_1 (первый этап — ядро CRM)

### 2.1. Модели и БД

| Сущность | Таблицы | Статус |
|----------|---------|--------|
| Contact | `crm_contacts`, `crm_contact_phones`, `crm_contact_emails` | ✅ |
| Company | `crm_companies`, `crm_company_contacts` | ✅ |
| Pipeline / Stage | `crm_pipelines`, `crm_stages` | ✅ |
| Deal | `crm_deals` | ✅ |

Миграция: `000016_crm_tables.up.sql` — созданы таблицы и индексы по спецификации. Soft delete через `deleted_at` везде, где указано.

### 2.2. API и маршруты

Базовый путь в спецификации: `/api/v1/workspaces/{workspaceId}/crm`.  
Фронтенд уже использует пути **без** префикса `/crm`: `/api/v1/workspaces/{workspaceId}/contacts|companies|deals|pipelines`. Реализация приведена к путям фронтенда.

| Группа | Эндпоинты | Реализовано |
|--------|-----------|-------------|
| **Контакты** | GET/POST /contacts, GET/PUT/DELETE /contacts/:id | ✅ |
| **Компании** | GET/POST /companies, GET/PUT/DELETE /companies/:id | ✅ |
| **Связь компания–контакт** | POST/DELETE /companies/:companyId/contacts/:contactId | ✅ |
| **Воронки** | GET/POST /pipelines | ✅ |
| **Сделки** | GET/POST /deals, GET/PUT/DELETE /deals/:id | ✅ |

Формат ответов приведён к ожиданиям фронтенда:

- Списки: `{ contacts | companies | deals: [], total: number }`, обёрнутые в `{ status, data }`.
- Pipelines: `{ pipelines: [] }`.
- PUT используется для обновления (как на фронте); частичное обновление через merge с существующей сущностью.

### 2.3. Бизнес-логика (SPEC_BACK_1, раздел 4)

| Правило | Статус |
|---------|--------|
| При создании контакта с `companyId` — добавление в `crm_company_contacts` | ✅ |
| При обновлении контакта — пересоздание связи компания–контакт при смене `companyId` | ✅ |
| Удаление контакта при наличии сделок — 409 Conflict | ✅ |
| Удаление компании при наличии контактов/сделок — 409 Conflict | ✅ |
| Создание сделки: проверка контакт/компания, статус `open`, валюта по умолчанию RUB | ✅ |
| Смена этапа сделки (логика won/lost, actual_close_date) | ⏳ Частично (обновление полей через общий PUT; отдельного POST /deals/:id/stage нет) |

### 2.4. Коды ответов и ошибки

Используются: 200, 201, 204, 400, 401, 403, 404, 409, 500. Формат ошибки — через существующий `Responder` (message, details при необходимости).

---

## 3. Соответствие фронтенду

| Аспект | Деталь |
|--------|--------|
| **URL** | Совпадают: `.../workspaces/:id/contacts`, `.../companies`, `.../deals`, `.../pipelines`, `.../activities`. |
| **Обёртка ответа** | Бэкенд отдаёт `{ status: "success", data: payload }`; фронт берёт `response.data.data`. |
| **Списки** | `data.contacts` / `data.companies` / `data.deals` и `data.total`. |
| **Pipelines** | `data.pipelines` — массив с `stages`. |
| **Метод обновления** | PUT (как во фронтовых сервисах). |
| **Моки** | В `client.ts` убрано принудительное использование моков для CRM; при `VITE_USE_MOCK_API !== 'true'` запросы идут на бэкенд. |

Типы сущностей (Contact, Company, Deal, Pipeline, Stage) и полей совпадают с фронтовыми типами и спецификацией.

---

## 3.1. Соответствие SPEC_BACK_2 (этап 2 — модуль активности)

### Модели и БД

| Сущность | Таблицы | Статус |
|----------|---------|--------|
| Activity | `crm_activities` | ✅ |
| Activity Files | `crm_activity_files` | ✅ (таблица; загрузка файлов через multipart — опционально позже) |
| Activity Reminders | `crm_activity_reminders` | ✅ (таблица; напоминания в заметках — опционально позже) |

Миграция: `000017_crm_activities.up.sql` — созданы таблицы и индексы по SPEC_BACK_2.

### API активностей

Путь по факту: `/api/v1/workspaces/{workspaceId}/activities` (без префикса `/crm`, как на фронте).

| Эндпоинт | Описание | Статус |
|----------|----------|--------|
| GET /activities?entityType=&entityId=&page=&limit=&types=&dateFrom=&dateTo=&importantOnly=&search | Список активностей сущности | ✅ |
| POST /activities | Создание заметки (type=note) или звонка (type=call) | ✅ |
| GET /activities/:id | Получение одной активности | ✅ |
| PUT /activities/:id | Обновление заметки (только type=note) | ✅ |
| DELETE /activities/:id | Удаление заметки (только type=note) | ✅ |
| POST /activities/:id/important | Пометка важности (тело опционально) | ✅ |
| POST /activities/:id/files | Загрузка файлов (multipart) | ⏳ Не реализовано |

### Автоматические события (на уровне приложения)

| Событие | Когда | Статус |
|---------|--------|--------|
| contact_created | После создания контакта | ✅ |
| company_created | После создания компании | ✅ |
| deal_created | После создания сделки (с названием этапа) | ✅ |
| contact_updated | После обновления контакта | ✅ |
| company_updated | После обновления компании | ✅ |
| deal_stage_changed | При смене этапа сделки (metadata: fromStage, toStage, dealValue) | ✅ |

Права: просмотр/создание/редактирование/удаление привязаны к доступу к workspace (HasAccess). Отдельные роли CRM_VIEW/CRM_EDIT в текущей реализации не введены.

---

## 4. Что сделано в коде

- **Миграции:** `000016_crm_tables.up.sql` (ядро CRM), `000017_crm_activities.up.sql` (лента активностей).
- **Модели:** `internal/model/crm.go` (Contact, Company, Deal, Pipeline, Stage, CrmActivity и вложенные типы).
- **Репозиторий:** `internal/repository/crm/` (repository.go, contact.go, company.go, pipeline.go, deal.go, activity.go).
- **Сервис:** `internal/service/crm/service.go` — CRUD по сущностям, проверки при удалении, активности (список, создание заметки/звонка, обновление/удаление заметки, важность), создание системных событий при создании/обновлении контакта, компании, сделки и смене этапа.
- **Обработчик:** `internal/handler/crm/handler.go` — эндпоинты контактов, компаний, воронок, сделок и активностей; проверка доступа к workspace; частичное обновление через merge.
- **DI:** в `internal/di/container.go` CRM service получает user repository для подстановки имени автора в активностях.

---

## 5. Резюме

- **План (PROJECT.md):** архитектура соблюдена: изоляция по workspace, без жёстких FK к другим модулям; RLS и project_id — на следующий этап.
- **SPEC_BACK_1:** ядро CRM реализовано: контакты, компании, воронки, сделки, связь компания–контакт, мягкое удаление, конфликты при удалении. Отдельный эндпоинт смены этапа (POST `/deals/:id/stage`) и автоматика won/lost по этапу при желании добавляются поверх текущего PUT.
- **SPEC_BACK_2:** модуль активности реализован: лента по entityType/entityId, создание заметок и звонков, обновление/удаление только заметок, пометка важности; автоматические события при создании/обновлении контакта, компании, сделки и смене этапа. Загрузка файлов (POST .../files) и напоминания — опционально на потом.
- **Фронтенд:** пути, форматы ответов (`data`, `total`, `page`, `limit` для активностей) и типы данных согласованы; моки для CRM (в т.ч. activities) отключены при работе с бэкендом.

Реализация готова к тестированию при применённых миграциях 000016 и 000017.
