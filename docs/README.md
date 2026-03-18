# Документация Backend

Навигация по документации проекта.

## Структура папок

| Папка | Назначение | Содержимое |
|-------|------------|------------|
| **spec/** | Спецификации | API, состояние ERP, системные спеки |
| **guides/** | Гайды | Архитектура, логирование, миграции, обучение |
| **reference/** | Справочники | Шпаргалки, методы API, SQL/NoSQL |
| **schema/** | Схемы БД | SCHEMA.md, ALL_MIGRATIONS_UP.sql, NEW_MIGRATE.sql |
| **modules/** | Документация модулей | MODULES, CRM, Habits, Invite, Notifications |
| **system/** | Системная документация | Роли, права, безопасность |
| **checks/** | Чек-листы приемки | CRM, роли, pipeline |
| **backlog/** | Бэклог | Auth, идеальная безопасность |
| **plans/** | Планы развития | Масштабирование, MVP, микросервисы |

## Быстрый доступ

### Спецификации
- [API ERP V1](SPEC/API_ERP_V1.md) — актуальный статус API
- [TOTAL_STATE_ERP v4](SPEC/TOTAL_STATE_ERP.v4.md) — полное состояние системы
- [FRONT_API_V1](SPEC/FRONT_API_V1.md) — фронтенд API

### Гайды
- [ARCHITECTURE](guides/ARCHITECTURE.md) — архитектура backend
- [SQL_AND_MIGRATIONS_GUIDE](guides/SQL_AND_MIGRATIONS_GUIDE.md) — миграции и соглашения
- [ERP_LEARNING_GUIDE](guides/ERP_LEARNING_GUIDE.md) — обучение разработке модулей
- [LOGGING](guides/LOGGING.md) — система логирования
- **[Developer Guide](guides/developer-guide/README.md)** — полный гайд: структура, БД, модули, план развития, интеграции

### Модули и лицензии
- [MODULES](modules/MODULES.md) — модули, лицензии, включение/отключение
- [MODULES_LICENSING_GUIDE](modules/MODULES_LICENSING_GUIDE.md) — полный гайд: таблицы, связи, логика, API
- [MODULES_PAID_VS_FREE_GUIDE](modules/MODULES_PAID_VS_FREE_GUIDE.md) — **платный/бесплатный/триал: как переключать**
- [MODULES_LICENSE_QUICK_GUIDE](modules/MODULES_LICENSE_QUICK_GUIDE.md) — быстрые команды (curl, SQL)

### Схема БД
- [SCHEMA](schema/SCHEMA.md) — визуализация таблиц

## Swagger / OpenAPI

- `swagger.json`, `swagger.yaml` — спецификация API (генерируется swaggo)
- `docs.go` — встроенная документация для Go
