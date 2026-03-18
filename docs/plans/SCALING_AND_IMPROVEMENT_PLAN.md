# Анализ текущих таблиц и план масштабирования

## 0. Принцип разбиения: 1 сервис = 1 БД = 1 бизнес-сущность

(По С. Шевелёву.)

- **Один сервис** — одна зона ответственности: одна бизнес-сущность или один домен. Вешать кучу таблиц с разной логикой в один сервис — плохая практика.
- **Одна БД у сервиса** — только те таблицы, которые относятся к этой сущности. Никаких общих таблиц «на все сервисы» в одной БД с доменной логикой.
- **Никакой сервис ни от какого не зависит** — общение только по API (или через оркестратор). Легко дорабатывать и выкидывать сервисы.

**Текущие модули в проекте (как они соотносятся с принципом):**

| Сервис/модуль | Таблицы (сейчас в одной БД) | Что хранит и что делает |
|----------------|-----------------------------|--------------------------|
| **Core** | users, workspaces, user_workspaces, modules, workspace_modules, user_module_licenses, user_preferences | Пользователи, воркспейсы, доступ, справочник модулей, лицензии. Один сервис хранит «владельца» и контекст; остальные по API запрашивают проверку доступа. |
| **Habits** | habits, habit_completions, habit_history, habit_versions | Данные о привычках в разрезе user + workspace. Хранит user_id, workspace_id (сейчас есть FK на Core; при split — убрать FK, оставить UUID). |
| **CRM** | crm_contacts, crm_companies, crm_pipelines, crm_stages, crm_deals, crm_activities, crm_contact_phones, crm_contact_emails, crm_company_contacts, crm_activity_* | Контакты, компании, воронки, сделки, лента активностей. Нет FK на Core — только workspace_id, owner_id как UUID. Готов к выносу в отдельную БД. |
| **Notes** | notes | Заметки по workspace. Сейчас FK на workspaces, users; при split — заменить на UUID, проверка через Core. |
| **Journal** | journal_entries | Записи дневника. Аналогично Notes. |
| **Shared/инфра** | request_logs, currencies, counterparties, activities (000009) | Инфраструктура и общие справочники. При split решается: остаются в Core или выносятся в отдельный сервис. |
| **Оркестратор/BFF** | — | Не хранит данных. Дёргает Core (проверка доступа, модули), затем CRM / Habits / Notes по workspace_id, собирает ответ. |

Итого: в модулях не должно быть таблиц и логики «про пользователей/воркспейсы» — только ссылки по id (user_id, workspace_id) и запросы к Core при необходимости.

---

## 1. Текущее состояние схемы БД

### 1.1. Ядро (Core) — владелец и контекст

| Таблица | Назначение | FK на другие таблицы |
|---------|------------|----------------------|
| `users` | Пользователи | — |
| `workspaces` | Рабочие пространства | owner_id → users |
| `user_workspaces` | Доступ user ↔ workspace (роль) | user_id → users, workspace_id → workspaces |
| `modules` | Справочник модулей (habits, crm, notes…) | — |
| `workspace_modules` | Какие модули включены в workspace | workspace_id → workspaces, module_id → modules |
| `user_module_licenses` | Лицензии пользователя на модуль | user_id → users, module_id → modules, workspace_id → workspaces (опц.) |
| `user_preferences` | Текущий workspace пользователя | user_id → users, current_workspace_id → workspaces |

**Вывод:** Ядро уже выделено: один «сервис» хранит пользователей, workspace и доступ. Остальные сервисы могут хранить только UUID и ходить в ядро за проверкой.

---

### 1.2. Модуль CRM

| Таблица | workspace_id | owner_id / created_by | FK на Core |
|---------|--------------|------------------------|------------|
| crm_contacts | UUID NOT NULL | UUID, без FK | **Нет** |
| crm_companies | UUID NOT NULL | UUID, без FK | **Нет** |
| crm_pipelines | UUID NOT NULL | created_by UUID, без FK | **Нет** |
| crm_stages | — | — | pipeline_id → crm_pipelines (внутри CRM) |
| crm_deals | UUID NOT NULL | UUID, без FK | pipeline_id → crm_pipelines, stage_id → crm_stages |
| crm_activities | UUID NOT NULL | created_by UUID, без FK | **Нет** |
| crm_contact_phones, crm_contact_emails | — | — | contact_id → crm_contacts |
| crm_company_contacts | — | — | company_id, contact_id (внутри CRM) |

**Вывод:** **Хороший задел под микросервис.** В CRM нет ни одного FK на `users` или `workspaces`. Изоляция по `workspace_id` делается в коде (WHERE workspace_id = $1). При переносе CRM в отдельную БД достаточно вынести эти таблицы; проверку «есть ли у user доступ к workspace» делать через вызов Core API.

---

### 1.3. Модуль Habits

| Таблица | FK |
|---------|-----|
| habits | user_id → users, workspace_id → workspaces (в constraints/01) |
| habit_completions | habit_id → habits, user_id → users |
| habit_history, habit_versions | habit_id → habits, user_id → users |

**Вывод:** Сейчас Habits жёстко привязан к Core через FK. При выделении в отдельный сервис нужно будет убрать FK на `users` и `workspaces`, оставить только колонки `user_id`, `workspace_id` (UUID) и проверять доступ через API Core.

---

### 1.4. Общие / shared (notes, journal, activities, currencies, counterparties)

- `notes`, `journal_entries`, `activities` (000009) — FK на workspaces и users.
- `currencies`, `counterparties` — FK на workspaces.

При разделении на сервисы: либо остаются в Core/Shared БД, либо переносятся с заменой FK на хранение только UUID и проверку через API.

---

## 2. Оценка задела для микросервисов и standalone

Основано на актуальном списке таблиц (миграции 000001–000017, constraints): **28 таблиц** — request_logs, users, workspaces, user_workspaces, user_preferences, modules, workspace_modules, user_module_licenses, habits, habit_completions, habit_history, habit_versions, activities, currencies, counterparties, notes, journal_entries, crm_contacts, crm_contact_phones, crm_contact_emails, crm_companies, crm_company_contacts, crm_pipelines, crm_stages, crm_deals, crm_activities, crm_activity_files, crm_activity_reminders.

| Критерий | Статус | Комментарий |
|----------|--------|-------------|
| CRM не зависит от Core по FK | ✅ | 12 таблиц crm_* — нет FK на users/workspaces; можно вынести в отдельную БД без смены схемы |
| Изоляция по workspace_id | ✅ | Есть в crm_*, habits, habit_completions, habit_versions, notes, journal_entries, currencies, counterparties, activities, workspace_modules; дочерние изолируются через родителя |
| Один сервис — один контекст (Core = владелец) | ✅ | user_workspaces + workspace_modules задают контекст; Core — 7 таблиц (users, workspaces, …) |
| **Habits/Core — жёсткая связь по FK** | ⚠️ | habits → users, habits → workspaces; habit_completions → users (constraints/01). При выделении Habits убрать FK, оставить UUID |
| Notes, Journal, Shared — FK на Core | ⚠️ | notes, journal_entries, currencies, counterparties, activities — REFERENCES workspaces(id) и/или users(id); при выносе заменить на UUID |
| Projects (контекст связки модулей) | ✅ | Реализовано: таблицы projects, project_entities (миграция 000018), API CRUD и привязки сущностей (вариант B — без project_id в модулях) |
| Режим «модуль без workspace» (standalone) | ❌ | У модульных таблиц workspace_id NOT NULL; tenant_type/tenant_id нет — для standalone нужна доработка схемы |
| Связи между модулями (CRM↔Tasks) | — | Модуля Tasks нет; при появлении — link-таблицы без FK |
| RLS (Row Level Security) | ❌ | Не включено; изоляция только в коде приложения |

**Итог:** Задел для микросервисов **нормальный**: CRM готов к выносу; **Projects реализован** (Core: projects, project_entities, API). Habits имеет **жёсткую привязку по FK** к Core (constraints/01). Notes, Journal, Shared — FK на Core. Без нового функционала пока не закрыты: standalone (tenant_type/tenant_id), RLS; Habits/Notes/Shared — снятие FK при выделении сервисов.

---

### 2.1. Как формируются связи таблиц, контекст и приватность

Ниже — правила, по которым в системе задаются связи между таблицами, контекст доступа и приватность (кто что видит и кто участник сущности).

---

**1. Контекст доступа (где лежат данные и кто к ним имеет право)**

- **Единица контекста** — **workspace** (таблица `workspaces`). Все данные, привязанные к «команде/аккаунту», привязаны к workspace.
- **Кто входит в контекст:** таблица **`user_workspaces`** (user_id, workspace_id, role). Строка = пользователь имеет доступ к этому workspace с данной ролью. Владелец workspace дополнительно задаётся `workspaces.owner_id`.
- **Правило доступа:** пользователь видит данные workspace только если есть запись в `user_workspaces` для (user_id, workspace_id). Проверка в коде: перед выборкой по любому модулю — убедиться, что текущий user_id имеет доступ к выбранному workspace_id (через user_workspaces или owner_id).

**2. Связь таблиц с контекстом**

- **Таблицы Core (контекст и участники):**
  - `workspaces` — владелец `owner_id` → users.
  - `user_workspaces` — явная связь пользователь ↔ контекст (user_id, workspace_id).
  - `projects` — принадлежат контексту: `workspace_id` → workspaces.
  - `project_entities` — привязка сущностей модулей к проекту (project_id, entity_type, entity_id); project_id → projects (в том же Core).

- **Таблицы модулей (CRM, Notes, Journal, будущий Tasks и т.д.):**
  - У сущностей, которые должны быть видны «в рамках workspace», в таблице есть **workspace_id** (ссылка на контекст). Примеры: crm_contacts, crm_deals, crm_companies, notes, journal_entries. Дочерние таблицы (crm_contact_phones, crm_stages и т.п.) привязаны к родителю; контекст наследуется от родителя (родитель уже с workspace_id).
  - Опционально **owner_id** / **created_by** (кто создал/владелец записи) — для участников и аудита; проверка доступа всё равно по workspace (пользователь должен быть в user_workspaces для этого workspace).

**3. Приватность и участники**

- **Кто видит сущность:** не по отдельной таблице «участников» на каждую сущность, а по контексту. Если сущность имеет workspace_id, то её видят все пользователи, у которых есть запись в user_workspaces для этого workspace_id. То есть приватность на уровне «контекста» (workspace), а не на уровне каждой сделки/задачи (если не вводить отдельную модель «участники сделки»).
- **Кто «участник» контекста:** задаётся только таблицей **user_workspaces**. Роль (role) может различать владельца, редактора, зрителя — логика в коде при проверке прав на действие.
- **Проекты:** доступ к проекту = доступ к workspace, в котором лежит проект (projects.workspace_id). Отдельной таблицы «пользователь ↔ проект» нет: список проектов для пользователя = проекты всех workspace, к которым у него есть доступ.

**4. Привязка сущностей модуля к проекту**

- Модули **не хранят** project_id (чтобы не зависеть от Core и не дублировать контекст).
- Факт «сделка/задача в проекте» хранится в Core в таблице **project_entities**: (project_id, entity_type, entity_id). entity_type — тип сущности модуля (например `crm_deal`, `task`), entity_id — id записи в своей таблице модуля.
- Чтобы показать «сделки по проекту»: взять из project_entities список entity_id по project_id и entity_type, затем в модуле выбрать записи по этим id (и по workspace_id для безопасности). Чтобы показать «в каких проектах эта сделка»: выбрать project_id из project_entities по entity_type и entity_id (и убедиться, что проект в workspace, к которому есть доступ).

**5. Краткая схема связей**

| Что | Как задаётся |
|-----|----------------|
| Контекст доступа | workspace (таблица workspaces) |
| Пользователь ↔ контекст | user_workspaces (user_id, workspace_id, role) |
| Проект ↔ контекст | projects.workspace_id |
| Данные модуля ↔ контекст | workspace_id в таблицах модуля (crm_*, notes, journal_entries, …) |
| Сущность модуля ↔ проект | project_entities (project_id, entity_type, entity_id); в модуле project_id не хранится |
| Приватность (кто видит) | Доступ к workspace через user_workspaces; все данные с этим workspace_id видны участникам этого workspace |

---

## 3. Жёсткая развязка: хранить ли ID сущностей Core в модулях?

**Принцип жёсткой развязки:** в таблицах модуля (CRM, Tasks и т.д.) **не должны храниться идентификаторы сущностей, принадлежащих Core** (users, workspaces, **projects**). Иначе модуль логически зависит от Core: значение `project_id` в сделке имеет смысл только если в Core есть проект с таким id. При переносе CRM в отдельную БД эта колонка будет хранить «чужой» UUID без возможности проверки.

**Исключение, которое уже есть:** `workspace_id` и `owner_id` в модулях — это тоже ID из Core. Их часто оставляют как «минимальную привязку к контексту»: модуль должен знать, в каком workspace данные лежат и кто владелец. Без FK это мягкая связь: при split модуль просто хранит UUID, проверку делает BFF/Core. Для **projects** можно поступить так же (хранить project_id без FK) или **вообще не хранить** — см. варианты ниже.

---

### Два варианта привязки сущностей к проектам

| | Вариант A: project_id в таблицах модуля | Вариант B: только таблица связей в Core |
|--|-----------------------------------------|------------------------------------------|
| **Где хранится факт «сделка в проекте»** | В модуле: `crm_deals.project_id` | В Core: `project_entities(project_id, entity_type, entity_id)` |
| **Хранит ли модуль ID из Core** | Да (project_id) | **Нет** — модуль не хранит ни одного ID Core |
| **Жёсткая развязка** | Мягкая: модуль хранит «контекст» из Core | **Соблюдается**: модули не знают про projects |
| **Запрос «сделки по проекту»** | CRM: `WHERE project_id = $1` | Core отдаёт список entity_id по project_id; CRM: `WHERE id = ANY($1)` |
| **Кто создаёт привязку** | Модуль при создании/обновлении сделки (получает project_id из запроса, проверка в BFF/Core) | BFF/Core: после создания сделки в CRM записывает строку в project_entities |

**Рекомендация при требовании жёсткой развязки:** **Вариант B** — в модулях не хранить `project_id`; привязку вести только в Core в таблице связей. Тогда ни один модуль не хранит ID сущностей Core (кроме уже принятого workspace_id/owner_id при необходимости).

---

## 4. План улучшений и масштабирования

### Фаза 1: Укрепление контекста (без смены БД)

Ниже — **подробная постановка задачи** с учётом двух вариантов (A и B). Реализовать нужно один из них.

---

#### Задача 1.1. Таблица projects (в Core)

**Цель:** общий контекст для связки модулей внутри workspace.

**Миграция (новая, в Core):**

- Таблица `projects`:
  - `id` UUID PRIMARY KEY DEFAULT gen_random_uuid()
  - `workspace_id` UUID NOT NULL
  - `name` VARCHAR(255) NOT NULL
  - `description` TEXT
  - `created_at` TIMESTAMPTZ NOT NULL DEFAULT NOW()
  - `updated_at` TIMESTAMPTZ NOT NULL DEFAULT NOW()
- Индекс: `(workspace_id)`.
- Опционально: комментарий, что таблица принадлежит ядру; при split projects остаётся в БД Core.

**Проверка доступа:** создание/редактирование/удаление проекта только при наличии доступа к workspace (как для остальных сущностей Core). Не добавлять FK на `workspaces` при желании сохранить возможность выноса projects в отдельный сервис позже; иначе можно оставить `REFERENCES workspaces(id)` в рамках одной БД Core.

---

#### Задача 1.2. Привязка сущностей модулей к проектам

Выбрать один из вариантов.

**Вариант A (project_id в модуле — мягкая связь)**

- В **CRM** (отдельная миграция модуля CRM):
  - Добавить в `crm_deals`: `project_id UUID NULL`.
  - Индекс: `(workspace_id, project_id)` (или частичный по project_id WHERE project_id IS NOT NULL).
- **Не** добавлять FK на `projects` (таблица в другой БД при split).
- Логика приложения:
  - При создании/обновлении сделки с `project_id`: BFF или CRM-сервис проверяет через Core API, что проект существует и `project.workspace_id = текущий workspace`; затем сохраняет deal с этим project_id.
  - Список сделок по проекту: CRM отдаёт по фильтру `WHERE workspace_id = $1 AND project_id = $2`.

**Минус для развязки:** модуль хранит ID сущности Core (project_id).

---

**Вариант B (только таблица связей в Core — жёсткая развязка)**

- В модулях **не** добавлять колонку `project_id`. Модули не хранят ID проектов.
- В **Core** (миграция Core) добавить таблицу привязки сущностей к проектам, например:
  - `project_entities`:
    - `id` UUID PRIMARY KEY DEFAULT gen_random_uuid()
    - `project_id` UUID NOT NULL (в рамках одной БД можно REFERENCES projects(id))
    - `entity_type` VARCHAR(50) NOT NULL — например `'crm_deal'`, `'task'`
    - `entity_id` UUID NOT NULL — id сделки/задачи в своей БД модуля
    - `created_at` TIMESTAMPTZ NOT NULL DEFAULT NOW()
    - UNIQUE(project_id, entity_type, entity_id)
  - Индексы: `(project_id, entity_type)`, `(entity_type, entity_id)` (для запросов «в каких проектах эта сделка»).
- Логика приложения:
  - Привязка сделки к проекту: после создания/обновления сделки в CRM BFF или Core по API записывает/удаляет строку в `project_entities` (project_id, 'crm_deal', deal_id). CRM не получает и не хранит project_id.
  - Список сделок по проекту: BFF запрашивает у Core список (entity_type='crm_deal', entity_id) по project_id; затем запрашивает у CRM сделки по списку id (GET /deals?ids=... или фильтр по массиву id). Либо Core отдаёт только список entity_id, CRM отдаёт сделки по этим id.
  - Удаление проекта: в Core удаляются строки из project_entities по project_id; сущности в модулях не меняются (сделки остаются, просто без проекта).

**Плюс:** модули не хранят ни одного ID из Core (кроме уже принятых workspace_id/owner_id). Полная развязка по проектам.

---

#### Задача 1.3. API и права

- **Core (или BFF):**
  - CRUD по проектам: создание/чтение/обновление/удаление с проверкой доступа к workspace.
  - Для варианта B: эндпоинты вида «добавить сущность в проект», «убрать из проекта», «список entity_id по project_id и entity_type».
- Модули (CRM и позже Tasks) не экспортируют project_id в ответах, если выбран вариант B; при варианте A — возвращают project_id как обычно.

---

#### Задача 1.4. RLS (опционально)

- Включить RLS на таблицах с `workspace_id` (в т.ч. в модулях, если они пока в одной БД).
- Установка контекста: в начале запроса (middleware/транзакция) `SET app.current_workspace_id = :workspace_id`.
- Политики: `USING (workspace_id = current_setting('app.current_workspace_id')::UUID)` и аналог для INSERT/UPDATE.
- Это не заменяет проверки в коде, но страхует от забытого WHERE при прямых запросах к БД.

---

#### Сводка по Фазе 1

| Шаг | Описание | Вариант A | Вариант B |
|-----|----------|-----------|-----------|
| 1.1 | Таблица `projects` в Core | Да | Да |
| 1.2 | Привязка к проектам | `project_id` в crm_deals | Таблица `project_entities` в Core, без project_id в модулях |
| 1.3 | API проектов и привязок | CRUD проектов + передача project_id в CRM | CRUD проектов + API привязок (add/remove/list по project_id, entity_type) |
| 1.4 | RLS | По желанию | По желанию |

**Про противоречие с жёсткой развязкой:** вариант с хранением `project_id` в модулях (A) **противоречит** правилу «в модулях не хранить ID сущностей Core». Вариант B ему **соответствует**. Для строгой развязки таблиц выбирать **Вариант B**.

---

### Фаза 2: Готовность к выделению сервисов

5. **Закрепить правило: модули не создают FK на Core.**  
   Для новых модулей (например, Tasks): только `workspace_id`, `owner_id`, `assignee_id` и т.п. как UUID; проверка пользователя и workspace — через вызов Core (или JWT/контекст от BFF).

6. **Habits при выделении:**  
   Новая миграция в будущем: удалить FK `habits → users`, `habits → workspaces`, `habit_completions → users`. Оставить колонки `user_id`, `workspace_id`. Сервис Habits при необходимости проверяет доступ через HTTP к Core.

7. **BFF / оркестратор.**  
   Один слой API (или Gateway), который по запросу с `workspace_id`/user из токена:
   - дергает Core (проверка доступа, список модулей, projects);
   - дергает CRM, Habits, будущий Tasks по необходимости;
   - собирает ответ. Модули друг друга не вызывают, только BFF вызывает модули.

---

### Фаза 3: Standalone и гибкий контекст (по необходимости)

8. **Поддержка «модуль без workspace».**  
   Вариант A: в таблицах модуля добавить `tenant_type VARCHAR(20)`, `tenant_id UUID` вместо единственного `workspace_id`.  
   - ERP: `tenant_type = 'workspace'`, `tenant_id = workspace_id`.  
   - Standalone: `tenant_type = 'user'`, `tenant_id = user_id`.  
   Вариант B: оставить `workspace_id` nullable; для standalone везде `workspace_id IS NULL` и фильтр по `owner_id = user_id` (или аналог).

9. **Projects в standalone.**  
   Таблица projects с полем-владельцем контекста: например `tenant_type` + `tenant_id` (workspace или user). Тогда одни и те же сущности проектов работают и в ERP, и в отдельном приложении.

10. **Связи между модулями (CRM ↔ Tasks и т.д.).**  
    Хранить в таблицах связей (например `task_entity_links` в сервисе Tasks или общая `entity_links` в отдельном маленьком сервисе): только `entity_type` + `entity_id` (UUID), без FK на другие БД.

---

## 5. План по масштабированию и добавлению модулей

### 5.1. Масштабирование (выделение сервисов в отдельные БД)

1. **Выбор первого кандидата на вынос**  
   Лучше начать с модуля без FK на Core — у нас это **CRM**. Перенести все таблицы `crm_*` в отдельную БД; приложение CRM (или общий API) подключается к двум БД: Core и CRM. Все запросы к сделкам/контактам идут в CRM-БД; проверка доступа (user, workspace) — запрос к Core API или к Core БД в монолите.

2. **Настройка доступа**  
   Сервис CRM получает в каждом запросе `workspace_id` и `user_id` (из JWT или из BFF). Валидация «пользователь имеет доступ к workspace» выполняется в BFF или в Core перед вызовом CRM. В CRM остаётся только фильтр `WHERE workspace_id = $1`.

3. **Вынос Habits (или Notes)**  
   Перед переносом БД: миграция в текущей БД — удалить FK с `habits` и `habit_completions` на `users` и `workspaces`. Оставить колонки `user_id`, `workspace_id` как UUID. Затем перенести таблицы habits, habit_completions, habit_history, habit_versions в новую БД. Логика доступа — как у CRM.

4. **Оркестратор (BFF)**  
   Отдельный слой API: принимает запрос с токеном, извлекает user_id и текущий workspace_id, вызывает Core (проверка доступа, список модулей), затем при необходимости вызывает сервисы CRM, Habits и т.д., собирает ответ. Модули между собой не дергают.

### 5.2. Добавление нового модуля (например Tasks)

**Backend:**

1. Запись в справочник: миграция с `INSERT INTO modules (code, name, description, is_core) VALUES ('tasks', 'Задачи', '...', false)`.
2. Миграции таблиц модуля — только свои сущности, у каждой таблицы `workspace_id UUID NOT NULL`, без FK на `users`/`workspaces`. При необходимости `owner_id`, `assignee_id` как UUID. См. [SQL_AND_MIGRATIONS_GUIDE.md](../guides/SQL_AND_MIGRATIONS_GUIDE.md).
3. Модели, репозиторий, сервис, хендлер в пакете по домену (например `internal/handler/tasks`, `internal/repository/tasks`). Проверка доступа к workspace через общий middleware (HasAccess по Core).
4. Регистрация роутов под ` /api/v1/workspaces/:workspaceId/tasks/...`.
5. По желанию: включение модуля по умолчанию для новых workspace (триггер или сид в `workspace_modules`) или только по запросу.

**Frontend:**

1. В `app/modules/config.ts` добавить объект в массив `modules` с `id: 'tasks'` (= `code` из БД), `basePath`, `routes`.
2. Страницы и API-клиент для эндпоинтов модуля. Guard по `enabledModules` уже проверяет доступ к роуту по коду модуля.

**Связи с другими модулями:** не хранить в таблицах Tasks FK на CRM. Если нужна привязка «задача к сделке» — таблица связей с `entity_type` и `entity_id` (UUID), без FK на другую БД.

---

## 6. Краткий чеклист по таблицам

| Действие | Когда |
|----------|--------|
| Ввести таблицу `projects` в Core | Фаза 1 |
| Привязка к проектам: либо `project_id` в crm_deals (вариант A), либо таблица `project_entities` в Core (вариант B, жёсткая развязка) | Фаза 1 |
| Не добавлять FK из модулей на users/workspaces/**projects** в новых миграциях; при варианте B — не хранить project_id в модулях | Сейчас / Фаза 1 |
| Включить RLS на таблицах с workspace_id | Фаза 1 (опционально) |
| При выделении CRM — перенести только crm_* таблицы, доступ проверять через Core API | При split |
| При выделении Habits — убрать FK на users/workspaces, оставить UUID | При split |
| Добавить tenant_type/tenant_id или nullable workspace_id для standalone | Фаза 3 при необходимости |
| Связи между модулями — только link-таблицы с entity_type + entity_id | При появлении второго модуля, который ссылается на первый |

---

## 7. Итог

- **Задел нормальный:** CRM уже изолирован от Core по FK, изоляция по workspace есть, Core явно хранит «владельца» и состав модулей. Это соответствует принципу «1 сервис хранит инфу о владельце, остальные ходят за ней».
- **Жёсткая развязка:** чтобы в модулях не хранились ID сущностей Core, привязку к проектам делать через таблицу связей в Core (вариант B), а не через колонку project_id в crm_deals (вариант A).
- **Для масштабирования:** добавить projects и выбранный механизм привязки (A или B); не плодить FK из модулей в Core; при разделении БД убирать FK у Habits и оставлять только UUID; ввести BFF/оркестратор; связи между модулями — только через таблицы связей без FK.
- **Для standalone-модулей:** позже ввести единый контекст (tenant_type/tenant_id или nullable workspace_id) и проекты, привязанные к этому контексту.
- **Документация:** гайд по таблицам и миграциям — [SQL_AND_MIGRATIONS_GUIDE.md](../guides/SQL_AND_MIGRATIONS_GUIDE.md); сводное состояние проекта после правок — [PROJECT_STATE_REPORT.md](./PROJECT_STATE_REPORT.md).
