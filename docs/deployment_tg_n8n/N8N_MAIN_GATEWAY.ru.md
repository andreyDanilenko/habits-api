# n8n Main Gateway (clean v2)

Один универсальный workflow для входящих событий от приложения:

`POST /webhook/gateway/events` -> `Switch(event_type)` -> нужный провайдер (Telegram/Google/...)

## Зачем

- Не плодить отдельные webhook под каждый кейс.
- Единый контракт между backend и n8n.
- Проще добавлять новые интеграции: добавляешь новую ветку в `Switch`.

## Импорт workflow

- Файл: `deployment/n8n/workflows/main-gateway.json`
- После импорта включи workflow (`Active`) для production URL.

## ENV

Добавь в `deployment/.env`:

```env
N8N_GATEWAY_SECRET=replace_with_gateway_secret
TELEGRAM_USER_BOT_TOKEN=replace_with_user_bot_token
TELEGRAM_USER_CHAT_ID=replace_with_default_user_chat_id
```

## Контракт события от backend

```json
{
  "secret": "replace_with_gateway_secret",
  "event_type": "task.created",
  "user_id": "uuid",
  "chat_id": "optional_telegram_chat_id",
  "payload": {
    "title": "Новая задача",
    "assignee": "Andrei",
    "link": "https://app/tasks/123"
  }
}
```

Где:
- `event_type` обязателен (`task.created`, `user.mention`, и т.д.)
- `chat_id` опционален, если пусто — берется `TELEGRAM_USER_CHAT_ID`

## Локальный тест

Test mode:

```bash
curl -X POST "http://localhost:5678/webhook-test/gateway/events" \
  -H "Content-Type: application/json" \
  -d '{
    "secret":"replace_with_gateway_secret",
    "event_type":"task.created",
    "payload":{"title":"Тест clean v2","assignee":"Andrei","link":"http://localhost:3000"}
  }'
```

Production mode (workflow active):

```bash
curl -X POST "http://localhost:5678/webhook/gateway/events" \
  -H "Content-Type: application/json" \
  -d '{
    "secret":"replace_with_gateway_secret",
    "event_type":"task.created",
    "payload":{"title":"Тест clean v2","assignee":"Andrei","link":"http://localhost:3000"}
  }'
```

## План перехода (без простоя)

1. Импортировать `Main Gateway`, протестировать в `webhook-test`.
2. Перевести backend на новый URL `/gateway/events`.
3. Убедиться, что события приходят стабильно.
4. Только потом удалять старые workflow и legacy таблицы.
