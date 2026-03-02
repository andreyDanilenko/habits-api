# Тестовые данные CRM для workspace

Скрипт создаёт в указанном workspace тестовые данные для проверки CRM: воронку продаж, компании, контакты, сделки и одну запись в ленте активностей.

## Требования

- В БД уже должны быть применены миграции CRM (000016, 000017).
- Указанный `USER_ID` должен существовать в таблице `users`.
- Указанный `WORKSPACE_ID` должен существовать в таблице `workspaces`, и пользователь должен иметь доступ (запись в `user_workspaces`).

## Как запустить

Из корня backend (или репозитория, где лежит `go.mod`):

```bash
WORKSPACE_ID=<uuid-workspace> USER_ID=<uuid-user> go run ./cmd/seed_crm
```

### Пример

```bash
# Подставьте свои UUID workspace и пользователя (владельца записей)
WORKSPACE_ID=550e8400-e29b-41d4-a716-446655440000 USER_ID=6ba7b810-9dad-11d1-80b4-00c04fd430c8 go run ./cmd/seed_crm
```

Подключение к БД берётся из конфига приложения (переменные окружения или `.env`: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`).

## Что создаётся

| Сущность | Количество и поля |
|----------|-------------------|
| **Воронки** | 4: «Продажи» (по умолчанию), «Подписки», «Партнёрские», «B2B» — у каждой свои этапы и цвета |
| **Компании** | 30: name, inn, kpp, ogrn, phone, email, website, legal_address, actual_address (JSONB), tags |
| **Контакты** | 100: first_name, last_name, middle_name, position, birthday, tags, custom_fields (JSONB); mobile+work телефоны, work+personal email; 70 привязаны к компаниям |
| **Сделки** | 20: распределены по всем 4 воронкам; name, contact, company, budget, currency (RUB/USD/EUR), expected_close_date, actual_close_date, status (open/won/lost), lost_reason, description, source, probability, tags |
| **Активность** | Записи в feed по первым 5 сделкам, 10 контактам и 5 компаниям |

После выполнения в логах: `CRM seed done for workspace ..., user .... Created: 4 pipelines, 30 companies, 100 contacts, 20 deals (all fields, all pipelines), activity feed (sample).`
