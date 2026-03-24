## N8N Message Contract (clean v2)

Единая точка входа в n8n:

- URL: `POST /webhook/gateway/events`
- Secret: `N8N_GATEWAY_SECRET`

### Request JSON

```json
{
  "secret": "replace_with_gateway_secret",
  "event_type": "task.created",
  "user_id": "uuid",
  "chat_id": "optional",
  "payload": {
    "title": "Новая задача",
    "assignee": "Andrei",
    "link": "https://app/tasks/123"
  }
}
```

### event_type (начальный набор)

- `task.created`
- `user.mention`

### Правила

- `secret` обязателен; неверный secret -> `401`.
- `event_type` обязателен; пустой -> `401`.
- `payload` — свободный JSON (расширяемый контракт).

### Цель

- Один webhook для всех интеграций.
- Разруливание сценариев через `Switch` внутри n8n.

