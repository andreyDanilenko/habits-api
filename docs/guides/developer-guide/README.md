# Гайд разработчика Backend

Подробный гайд для разработчика, который учится и много кода написал с помощью AI. Помогает понять структуру backend, связи между таблицами, взаимодействие модулей и план развития.

---

## Содержание

| # | Файл | Описание |
|---|------|----------|
| 1 | [01_OVERVIEW_AND_STRUCTURE.md](./01_OVERVIEW_AND_STRUCTURE.md) | Как строилась структура с самого начала: точка входа, слои, DI, поток запроса |
| 2 | [02_DATABASE_AND_TABLES.md](./02_DATABASE_AND_TABLES.md) | Схема БД, связи между таблицами, миграции, FK, триггеры |
| 3 | [03_MODULES_INTERACTION.md](./03_MODULES_INTERACTION.md) | Взаимодействие: users, workspaces, licenses, counterparties, currencies, permissions |
| 4 | [04_DEVELOPMENT_PLAN.md](./04_DEVELOPMENT_PLAN.md) | Потенциальный план развития на основе документации |
| 5 | [05_INTEGRATIONS.md](./05_INTEGRATIONS.md) | Возможности интеграций, внешние сервисы, точки расширения |

---

## Рекомендуемый порядок чтения

1. **01** — общая картина и архитектура
2. **02** — как устроена БД
3. **03** — как модули связаны между собой
4. **04** — куда двигаться дальше
5. **05** — как расширять систему интеграциями

---

## Связанная документация

- [ARCHITECTURE.md](../ARCHITECTURE.md) — архитектура backend
- [docs/README.md](../../README.md) — навигация по всей документации
- [SCHEMA.md](../../schema/SCHEMA.md) — визуализация схемы БД
- [MODULES_LICENSING_GUIDE.md](../../modules/MODULES_LICENSING_GUIDE.md) — модули и лицензии
