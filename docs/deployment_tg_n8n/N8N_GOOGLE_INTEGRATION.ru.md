# n8n: безопасность + первая интеграция с Google

Этот документ фиксирует текущую безопасность internal endpoint и дает первую рабочую интеграцию:

`Google Sheets -> n8n -> Nest internal notifications -> WebSocket -> Frontend`

---

## 1) Что уже защищено

- `nest_satellite` не публикуется наружу (в compose нет `ports`, только `expose`).
- Nginx не проксирует `POST /internal/notifications/send` во внешний интернет.
- Endpoint защищен заголовком `x-internal-api-key`.
- Ключ берется из `INTERNAL_NOTIFICATIONS_API_KEY`.

Итог: внешний интернет не может напрямую спамить этим route.

---

## 2) Переменные окружения

В `deployment/.env`:

```env
INTERNAL_NOTIFICATIONS_API_KEY=replace_with_strong_internal_key
N8N_INTERNAL_NOTIFICATIONS_URL=http://nest_satellite:3001/internal/notifications/send
```

В `deployment/docker-compose.yml` эти переменные прокинуты в `n8n`.

---

## 3) Первая интеграция с Google (Google Sheets)

Сценарий: при ручном запуске (первый тест) n8n:
1. добавляет строку в Google Sheet (аудит),
2. отправляет уведомление в `nest-satellite` через internal endpoint.

### Шаг 1. Создай Google OAuth (для n8n)

1. Открой [Google Cloud Console](https://console.cloud.google.com/).
2. Создай проект (или выбери существующий).
3. Включи API:
   - `Google Sheets API`
4. Создай OAuth Client ID (тип `Web application`).
5. Добавь Redirect URI из n8n:
   - `https://n8n.lifedream.tech/rest/oauth2-credential/callback`
   - для локалки: `http://localhost:5678/rest/oauth2-credential/callback`
6. Сохрани `Client ID` и `Client Secret`.

### Шаг 2. Создай credentials в n8n

1. `n8n -> Credentials -> New -> Google Sheets OAuth2 API`.
2. Вставь `Client ID`/`Client Secret`.
3. Авторизуйся Google-аккаунтом.

### Шаг 3. Собери workflow в n8n

Ноды:

1. `Manual Trigger`
2. `Set` (или `Edit Fields`) с полями:
   - `userId` (string)
   - `event` = `new_task`
   - `title` = `Тест из Google интеграции`
   - `link` = `/tasks/123`
3. `Google Sheets` (Append Row):
   - документ/лист: тестовый
   - колонки: timestamp, userId, event, title, link
4. `HTTP Request`:
   - Method: `POST`
   - URL: `{{$env.N8N_INTERNAL_NOTIFICATIONS_URL}}`
   - Header:
     - `x-internal-api-key: {{$env.INTERNAL_NOTIFICATIONS_API_KEY}}`
   - Send Body as JSON:
```json
{
  "userId": "={{$json.userId}}",
  "event": "={{$json.event}}",
  "payload": {
    "title": "={{$json.title}}",
    "link": "={{$json.link}}"
  }
}
```

### Шаг 4. Проверка

1. Открой фронтенд под пользователем `userId`.
2. Запусти workflow вручную (`Execute workflow`).
3. Ожидаемо:
   - в Google Sheets появилась новая строка,
   - на фронт пришло realtime уведомление.

---

## 4) Типичные ошибки

- `401 Invalid internal api key`:
  - ключ в `n8n` и `nest-satellite` не совпадает.
- `ECONNREFUSED` из n8n:
  - неверный URL (проверь `N8N_INTERNAL_NOTIFICATIONS_URL`).
- Не приходит WS событие:
  - frontend не подключен к namespace `/notifications`,
  - пользователь не авторизован (JWT cookie).
