# Отчёт о состоянии проекта (после правок документации)

**Дата:** 2026-03-01  
**Контекст:** Обновлены план масштабирования, гайд по SQL/миграциям и принципы разбиения на сервисы. Таблица «Оценка задела» в SCALING_AND_IMPROVEMENT_PLAN приведена в соответствие с актуальным списком таблиц (28 таблиц, миграции 000001–000017). Ниже — сводное состояние проекта.

---

## 1. Документация (обновлённая и новая)

| Документ | Назначение |
|----------|------------|
| **SCALING_AND_IMPROVEMENT_PLAN.md** | Принцип 1 сервис / 1 БД / 1 бизнес-сущность; текущие модули (Core, Habits, CRM, Notes, Journal, Shared); оценка задела под микросервисы; жёсткая развязка (projects через таблицу связей в Core, без project_id в модулях); фазы улучшений; **план по масштабированию и добавлению модулей** (вынос сервисов, добавление нового модуля). |
| **SQL_AND_MIGRATIONS_GUIDE.md** | Гайд по формированию таблиц и миграций: именование файлов и таблиц, соглашения по колонкам (id, workspace_id, deleted_at), правила FK (внутри модуля — да, на Core — нет), индексы, типы данных, чеклист для новой миграции. |
| **PROJECT_STATE_REPORT.md** | Этот отчёт — сводное состояние проекта после правок. |
| **MODULES.md** | Регистрация модулей (backend: modules, workspace_modules; frontend: config, enabledModules), включение/отключение в workspace, лицензии. |
| **CRM/REPORT_IMPLEMENTATION.md** | Детальный отчёт по реализации CRM (SPEC_BACK_1, SPEC_BACK_2), API, соответствие фронту. |
| **CRM/PROJECT.md**, **CONTEXT.md** | Архитектура CRM, контекст (workspace, projects, tenant), связи между модулями. |

---

## 2. Текущие модули и таблицы

| Модуль | Таблицы | Миграции | FK на Core |
|--------|---------|----------|------------|
| **Core** | users, workspaces, user_workspaces, modules, workspace_modules, user_module_licenses, user_preferences, **projects**, **project_entities** | 000002, 000005, 000006, 000007, 000012, 000013, **000018** | — (внутри Core есть FK между своими таблицами; projects без FK на workspaces для задела под вынос) |
| **Habits** | habits, habit_completions, habit_history, habit_versions | 000003, 000004, 000008, 000010, 000011; constraints/01 | Да (habits → users, workspaces; completions → users). При выносе — убрать. |
| **CRM** | crm_contacts, crm_companies, crm_pipelines, crm_stages, crm_deals, crm_activities + дочерние | 000016, 000017 | **Нет** — готов к выносу в отдельную БД. |
| **Notes** | notes | 000014 | Да (workspace, user). При выносе — заменить на UUID. |
| **Journal** | journal_entries | 000015 | Да (workspace, user). При выносе — заменить на UUID. |
| **Shared** | currencies, counterparties; activities (000009) | 000014, 000009 | FK на workspaces/users. |
| **Инфра** | request_logs | 000001 | — |

Справочник модулей в БД: `modules.code` — `habits`, `crm`, `notes` (и др. по миграциям). Включение в workspace — через `workspace_modules`.

---

## 3. Соответствие принципу «1 сервис — 1 БД — 1 бизнес-сущность»

- **Сейчас:** все таблицы в одной БД. Логическое разбиение по префиксам и доменам (crm_*, habits, notes, …) и документировано в SCALING_AND_IMPROVEMENT_PLAN.
- **CRM:** без FK на Core — задел под вынос в отдельную БД без изменения схемы.
- **Habits, Notes, Journal:** при выносе потребуется миграция с удалением FK на users/workspaces и переход на хранение только UUID; проверка доступа через Core API или BFF.
- **Projects:** реализованы в коде: миграция 000018 (projects, project_entities), API CRUD проектов и привязок сущностей (attach/detach, список entity_id по проекту, список project_id по сущности). Вариант B — модули не хранят project_id.

---

## 4. Миграции и SQL

- Используется **golang-migrate**; миграции в `backend/migrations/` с нумерацией NNNNNN_description.up/down.sql.
- Соглашения зафиксированы в **SQL_AND_MIGRATIONS_GUIDE.md**: UUID для id, workspace_id в модулях, без FK из модулей на Core, индексы по workspace_id и типичным фильтрам, мягкое удаление через deleted_at где нужно.
- Ограничения FK вынесены в `constraints/01_foreign_keys.up.sql` (в т.ч. для habits → users, workspaces).

---

## 5. Реализованный функционал (кратко)

- **Auth / Core:** регистрация, вход, JWT, доступ к workspace (user_workspaces), включение/отключение модулей (workspace_modules), лицензии (user_module_licenses).
- **CRM:** контакты, компании, воронки, сделки, лента активностей (заметки, звонки, системные события). API без префикса /crm в пути (…/contacts, …/deals, …/activities). См. CRM/REPORT_IMPLEMENTATION.md.
- **Habits:** привычки, завершения, история, версии — полный цикл по своей спецификации.
- **Notes, Journal:** таблицы и API по своим миграциям и хендлерам.

---

## 6. Что сделано в рамках правок (документация)

1. **SCALING_AND_IMPROVEMENT_PLAN.md**  
   - Принцип разбиения с примерами заменён на **текущие модули** (Core, Habits, CRM, Notes, Journal, Shared, оркестратор).  
   - Добавлен **план по масштабированию** (как выносить сервисы в отдельные БД, как добавить новый модуль — backend и frontend).  
   - Сохранены фазы улучшений, варианты A/B по projects, чеклист и итог.

2. **SQL_AND_MIGRATIONS_GUIDE.md**  
   - Описано, как в проекте формируются таблицы и миграции: именование файлов и таблиц, колонки, FK, индексы, типы, up/down, чеклист для новой миграции модуля.

3. **PROJECT_STATE_REPORT.md**  
   - Новый отчёт о состоянии проекта: актуальная документация, список модулей и таблиц, соответствие принципу 1 сервис / 1 БД, миграции/SQL, краткий функционал.

---

## 7. Рекомендуемые следующие шаги

- Фаза 1 (projects + project_entities) **реализована в коде**. Дальше: интеграция с UI (выбор проекта для сделки через BFF), при необходимости RLS.
- При добавлении нового модуля (например Tasks) следовать **SQL_AND_MIGRATIONS_GUIDE** и разделу «Добавление нового модуля» в **SCALING_AND_IMPROVEMENT_PLAN** (без FK на Core, регистрация в `modules`, роуты, фронт-конфиг).
- При принятии решения о выделении первого микросервиса — начать с **CRM** (уже без FK на Core).
