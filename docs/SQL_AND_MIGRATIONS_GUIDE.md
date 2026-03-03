# Гайд: как в проекте формируются таблицы и миграции

Описываются соглашения по SQL, именованию таблиц/колонок и структуре миграций в этом репозитории.

---

## 1. Инструмент миграций

- **golang-migrate** (пакет `github.com/golang-migrate/migrate/v4`).
- Каталог миграций: `backend/migrations/`.
- При старте приложения вызывается `database.RunMigrations()` — поднимаются все миграции из `file://./migrations` до последней версии.
- Таблица `schema_migrations` хранит номер текущей версии; откат — через `migrate.Down()` или вручную (см. [DATABASE.md](DATABASE.md)).

---

## 2. Именование файлов миграций

- Формат: `NNNNNN_краткое_описание.up.sql` и `NNNNNN_краткое_описание.down.sql`.
- `NNNNNN` — шестизначный порядковый номер (000001, 000002, …). Порядок применения — по возрастанию номера.
- Примеры из проекта:
  - `000001_create_request_logs.up.sql`
  - `000002_create_users.up.sql`
  - `000016_crm_tables.up.sql`
  - `000017_crm_activities.up.sql`
- Отдельная папка `constraints/` для FK и триггеров (например `01_foreign_keys.up.sql`), если их выносят после создания таблиц.

**Правило:** один логический шаг схемы — одна пара up/down. Не смешивать в одной миграции несвязанные изменения (например, новую таблицу модуля и правку Core).

---

## 3. Именование таблиц

- **snake_case**, множественное число или смысловое имя сущности.
- **Модульные таблицы** — префикс по домену, чтобы при выносе в отдельную БД было понятно, что куда относится:
  - Core: `users`, `workspaces`, `user_workspaces`, `modules`, `workspace_modules`, `user_module_licenses`, `user_preferences`.
  - Habits: `habits`, `habit_completions`, `habit_history`, `habit_versions`.
  - CRM: `crm_contacts`, `crm_companies`, `crm_pipelines`, `crm_stages`, `crm_deals`, `crm_activities`, `crm_contact_phones`, `crm_contact_emails`, `crm_company_contacts`, `crm_activity_files`, `crm_activity_reminders`.
  - Notes: `notes`. Journal: `journal_entries`.
- **Связующие таблицы** (many-to-many внутри одного модуля): имя по двум сущностям, например `crm_company_contacts` (company ↔ contact).
- Избегать общих имён без префикса для доменной логики (чтобы не путать с другими модулями при разделении БД).

---

## 4. Колонки: общие соглашения

- **Первичный ключ:** почти везде `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`. Исключение — чисто технические таблицы вроде `request_logs` с `id SERIAL`.
- **Временные метки:** `created_at`, `updated_at` — тип `TIMESTAMP` или `TIMESTAMPTZ`, по умолчанию `NOW()` где нужно. В CRM используется `TIMESTAMPTZ`.
- **Мягкое удаление:** колонка `deleted_at TIMESTAMP` (или TIMESTAMPTZ). NULL = запись активна; не NULL = удалена. В выборках фильтр `WHERE deleted_at IS NULL`.
- **Изоляция по workspace:** у таблиц, принадлежащих модулям и общим сущностям, колонка `workspace_id UUID NOT NULL`. Индекс по `workspace_id` (часто составной с `deleted_at` или другими полями для типичных запросов).
- **Владелец/автор:** `owner_id`, `created_by`, `updated_by` — UUID, без FK на `users` в таблицах модулей (см. [SCALING_AND_IMPROVEMENT_PLAN.md](SCALING_AND_IMPROVEMENT_PLAN.md)). В Core (например `workspaces`) допустим FK на `users`.

---

## 5. Внешние ключи (FK)

- **Внутри одного модуля** — FK разрешены (например `crm_stages.pipeline_id → crm_pipelines.id`, `crm_deals.stage_id → crm_stages.id`, `crm_contact_phones.contact_id → crm_contacts.id`). Так сохраняется целостность внутри своей БД.
- **Из модуля в Core (users, workspaces)** — в новых миграциях **не создавать** FK. Хранить только UUID (`workspace_id`, `owner_id`, `created_by` и т.д.). Проверка существования user/workspace — в приложении (Core API или общий слой доступа). Это нужно для последующего выноса модуля в отдельную БД.
- **Между разными модулями** — FK не создавать. Связи через таблицы связей с `entity_type` и `entity_id` (UUID) или через оркестратор.

---

## 6. Индексы

- На все колонки, по которым часто идёт фильтрация или join: `workspace_id`, `(workspace_id, deleted_at)`, внешние ключи внутри модуля, поля для сортировки (например `created_at`).
- Частичные индексы где уместно: например `WHERE deleted_at IS NULL`, `WHERE workspace_id = ... AND deleted_at IS NULL`.
- Именование: `idx_<таблица>_<колонки>`, например `idx_crm_deals_workspace`, `idx_crm_contacts_workspace`.

---

## 7. Типы данных (PostgreSQL)

- **UUID** — для id, ссылок на другие сущности (в т.ч. workspace_id, user_id в модулях).
- **TEXT / VARCHAR(n)** — по смыслу; для длинных текстов (описание, комментарий) — TEXT.
- **Массивы** — `TEXT[]` для тегов (например `tags TEXT[] DEFAULT '{}'`).
- **JSON** — `JSONB` для гибких структур: `metadata`, `custom_fields`, `legal_address`, `settings`.
- **Дата/время** — `DATE` для дат без времени; `TIMESTAMP`/`TIMESTAMPTZ` для полных меток. В CRM предпочтительно TIMESTAMPTZ.
- **Деньги** — `DECIMAL(15,2)` для сумм (например бюджет сделки).

---

## 8. Содержимое up- и down-миграций

**Up:**

- `CREATE TABLE ...` с нужными колонками, индексами и ограничениями (CHECK, UNIQUE) внутри модуля.
- При необходимости `INSERT` сидов (например записей в `modules`). Сиды лучше выносить в конец миграции или в отдельный шаг.
- Комментарии: `COMMENT ON TABLE ...`, `COMMENT ON COLUMN ...` — по желанию, для важных полей полезно.

**Down:**

- Обратный порядок: сначала удалить зависимые объекты (дочерние таблицы, индексы), затем основную таблицу.
- Для таблиц с FK: `DROP TABLE IF EXISTS child_table; DROP TABLE IF EXISTS parent_table;`
- В проекте примеры: `000016_crm_tables.down.sql` — каскадное удаление в обратном порядке создания.

---

## 9. Чеклист для новой миграции модуля

1. Создать пару файлов `NNNNNN_short_name.up.sql` и `.down.sql` с очередным номером.
2. В up: создать таблицы с префиксом домена (например `crm_*`, `tasks_*`). Обязательно `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`, при необходимости `workspace_id UUID NOT NULL`, `created_at`/`updated_at`, `deleted_at` для мягкого удаления.
3. Не добавлять FK на `users` и `workspaces`. Ссылки на пользователя/воркспейс — только колонками UUID.
4. Добавить индексы по `workspace_id` и часто используемым полям.
5. В down: удалить таблицы в порядке, обратном созданию (с учётом зависимостей внутри модуля).
6. Если модуль новый — добавить запись в `modules` (в этой же миграции или в отдельной) с уникальным `code` и при необходимости сид для `workspace_modules` (как в 000012).

---

## 10. Ссылки

- [DATABASE.md](DATABASE.md) — подключение к PostgreSQL, просмотр миграций, сброс schema_migrations.
- [MODULES.md](MODULES.md) — как модуль регистрируется в системе (backend + frontend), включение в workspace.
- [SCALING_AND_IMPROVEMENT_PLAN.md](SCALING_AND_IMPROVEMENT_PLAN.md) — принцип 1 сервис / 1 БД, какие таблицы к какому модулю относятся, план выноса и добавления модулей.
