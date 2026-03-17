# Clean Baseline: миграции для чистой базы с нуля

Декомпозиция по **сущностям** для запуска продукта на пустой БД.
Без заплаток, ALTER для существующих данных и backfill-скриптов.

---

## ⚠️ Синхронизация с текущими миграциями

**Эта папка должна соответствовать** финальному состоянию схемы после применения всех миграций из `../` (000001–000030 + constraints).

**При изменении текущих миграций** — обновляйте соответствующий файл здесь. См. `../README.md` (таблица соответствия).

## Порядок применения (001 → 027)

| # | Файл | Сущность |
|---|------|----------|
| 001 | infra_request_logs | Инфраструктура: логи запросов |
| 002 | auth_users | Пользователи |
| 003 | core_workspaces | Воркспейсы |
| 004 | core_user_workspaces | Связь пользователь–воркспейс |
| 005 | auth_user_preferences | Настройки пользователя |
| 006 | modules_system | Модули, workspace_modules, лицензии |
| 007 | habits | Привычки (полная схема) |
| 008 | habit_completions | Выполнения привычек |
| 009 | habit_history | История привычек |
| 010 | habit_versions | Версии привычек |
| 011 | activities | Глобальная лента активности |
| 012 | shared_currencies | Валюты (shared) |
| 013 | shared_counterparties | Контрагенты (shared) |
| 014 | shared_notes | Заметки |
| 015 | journal | Дневник |
| 016 | crm_contacts | CRM: контакты |
| 017 | crm_companies | CRM: компании |
| 018 | crm_pipelines | CRM: воронки и этапы |
| 019 | crm_deals | CRM: сделки |
| 020 | crm_activities | CRM: активности |
| 021 | projects | Проекты |
| 022 | permissions | Права и роли |
| 023 | auth_registration | Регистрация (токены) |
| 024 | auth_invitations | Приглашения |
| 025 | notifications | Уведомления |
| 026 | tasks | Задачи |
| 027 | task_comments | Комментарии к задачам |

## Примечания

- **Без backfill**: нет `INSERT ... FROM workspaces` — триггеры добавляют core-модули и роли при создании workspace.
- **Финальная схема**: habits с schedule, tasks с полными типами, registration_tokens с invite_token.

## Запуск

```bash
# PostgreSQL
for f in migrations/clean_baseline/0*.sql; do psql -d your_db -f "$f"; done
```

Или через migrate-инструмент после интеграции.
