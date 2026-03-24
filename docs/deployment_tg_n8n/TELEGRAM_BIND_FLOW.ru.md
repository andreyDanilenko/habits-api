# Telegram auto-connect flow (deep-link `/start`)

Цель: пользователь в ERP нажимает "Подключить Telegram", проходит в бота и автоматически привязывает свой `chat_id` к аккаунту.

> Рекомендуется использовать **отдельного бота для product-уведомлений** (не личного/админского), чтобы не смешивать каналы, права и аудит.

## Архитектура

1. ERP генерирует одноразовый токен привязки (TTL 10-15 минут).
2. ERP показывает deep-link:
   - `https://t.me/<TELEGRAM_BOT_USERNAME>?start=<token>`
3. Telegram отправляет update в n8n webhook.
4. n8n вызывает **Nest** (`POST .../internal/integrations/telegram/confirm`), **без прямого адреса Go**.
5. Nest проксирует тело и заголовок в **habits-api (Go)** `POST /api/v1/integrations/telegram/confirm`.
6. Go сохраняет связь `user_id <-> telegram_chat_id`.
7. n8n отправляет пользователю "Подключено".

## Что уже добавлено

- Готовый workflow:
  - `deployment/n8n/workflows/telegram-user-bind-start.json`
- Env в `deployment/docker-compose.yml` для `n8n`:
  - `TELEGRAM_LINK_CONFIRM_URL` → URL **Nest** (см. ниже), не Go
  - `TELEGRAM_USER_BOT_USERNAME` / токены бота
  - `INTERNAL_NOTIFICATIONS_API_KEY` (Nest и Go используют тот же ключ для internal-вызовов)
- Env для **nest_satellite**:
  - `HABITS_API_BASE_URL` — базовый URL Go (в Docker: `http://habits_api:8080`)

## Требуемые env

В `deployment/.env`:

```env
TELEGRAM_USER_BOT_USERNAME=lifedream_notify_bot
# Полный URL до Nest (n8n → Nest → Go):
TELEGRAM_LINK_CONFIRM_URL=http://nest_satellite:3001/internal/integrations/telegram/confirm
INTERNAL_NOTIFICATIONS_API_KEY=replace_with_strong_internal_key
TELEGRAM_USER_BOT_TOKEN=...
```

Для **nest_satellite** (тот же compose / сервер):

```env
HABITS_API_BASE_URL=http://habits_api:8080
```

## Контракт: публичный вызов n8n → Nest

`POST /internal/integrations/telegram/confirm`

Headers / body — те же, что раньше отправлялись в Go (Nest проверяет ключ и проксирует).

## Контракт в Go (за Nest, без изменений)

`POST /api/v1/integrations/telegram/confirm`

Headers:
- `x-internal-api-key: <INTERNAL_NOTIFICATIONS_API_KEY>`

Body:

```json
{
  "token": "one-time-token-from-start",
  "chatId": "123456789",
  "telegramUserId": "123456789",
  "telegramUsername": "john_doe"
}
```

Логика backend:

1. Проверить `x-internal-api-key`.
2. Проверить токен (подпись + TTL + one-time).
3. Найти `user_id` из токена.
4. Upsert в таблицу интеграции Telegram.
5. Пометить токен использованным.
6. Вернуть `200`.

## Рекомендуемая схема таблицы

```sql
create table if not exists user_telegram_links (
  user_id uuid primary key,
  telegram_chat_id text not null,
  telegram_user_id text not null,
  telegram_username text,
  is_enabled boolean not null default true,
  connected_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
```

## Как включить workflow

1. `n8n -> Workflows -> Import from file`
2. Выбрать `deployment/n8n/workflows/telegram-user-bind-start.json`
3. Activate workflow

## Важно: webhook Telegram -> n8n

После активации нужно зарегистрировать webhook у Telegram:

```bash
curl -X POST "https://api.telegram.org/bot<TELEGRAM_BOT_TOKEN>/setWebhook" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://n8n.lifedream.tech/webhook/telegram-connect"}'
```

Проверка:

```bash
curl "https://api.telegram.org/bot<TELEGRAM_BOT_TOKEN>/getWebhookInfo"
```

## Что отдавать на фронт (кнопка "Подключить Telegram")

Backend endpoint для frontend:
- `POST /api/v1/integrations/telegram/link`

Ответ:

```json
{
  "url": "https://t.me/lifedream_notify_bot?start=<token>"
}
```

Где `<token>` — одноразовый подписанный токен на пользователя.
