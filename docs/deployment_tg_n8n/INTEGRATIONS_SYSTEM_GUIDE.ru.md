# Система интеграций: текущее состояние и план

Этот документ фиксирует:
- что уже реализовано,
- что нужно доделать,
- как запускать локально и в проде,
- как избежать конфликтов,
- как расширять до GitHub и команд из Telegram.

## 1) Что уже сделано

### Backend
- Добавлен internal endpoint в `nest-satellite`:
  - `POST /internal/notifications/send`
  - защита по `x-internal-api-key`
- В `backend` добавлены endpoint'ы Telegram bind:
  - `POST /api/v1/integrations/telegram/link` (выдает deep-link `/start <token>`)
  - `POST /api/v1/integrations/telegram/confirm` (привязка chat_id к user)
- Добавлены миграции:
  - `000031_telegram_integrations.*` (telegram link tokens + user links)
  - `000032_user_integrations.*` (универсальная таблица связей)

### n8n
- `telegram-user-bind-start.json`:
  - принимает Telegram update `/start <token>`
  - вызывает backend confirm endpoint
- `main-gateway.json`:
  - единая точка входа событий приложения
  - `POST /webhook/gateway/events`
  - маршрутизация по `event_type` (через If-ветки)
cd /Users/andrei/Documents/myProject/deployment
./scripts/set-telegram-webhook.sh 'https://poison-express-past-female.trycloudflare.com/webhook/telegram-connect'
Команды для туннеля и `setWebhook` — только в `deployment/docs/DEV_TUNNEL.ru.md` и `deployment/docs/TELEGRAM_BIND_LOCAL_AND_PROD.ru.md` (**без токенов в репозитории**). Токен брать из `deployment/.env` (`TELEGRAM_USER_BOT_TOKEN`).

### Frontend
- В Workspace Settings добавлена кнопка
  - `Подключить Telegram`
  - вызывает `POST /api/v1/integrations/telegram/link`
  - открывает deep-link в Telegram

## 2) Целевая архитектура (v2)

```
Frontend -> Backend (/integrations/telegram/link) -> Telegram /start
Telegram -> n8n (/webhook/telegram-connect) -> Backend (/integrations/telegram/confirm)

Backend business event -> n8n (/webhook/gateway/events) -> Telegram/GitHub/...
```

Принцип: backend знает только "событие", n8n знает "как обработать и куда отправить".

## 3) Алгоритм запуска локально

1. Запустить backend (`localhost:8080`)
2. Запустить n8n (`localhost:5678`)
3. Импортировать в n8n:
   - `main-gateway.json`
   - `telegram-user-bind-start.json`
4. Для локального приема Telegram webhook поднять tunnel:
   - `cloudflared tunnel --url http://localhost:5678`
5. Поставить webhook Telegram на:
   - `https://<tunnel-host>/webhook/telegram-connect`
6. Проверить:
   - кнопка "Подключить Telegram" в UI
   - `/start <token>` в боте
   - execution в `telegram-user-bind-start`

## 4) Алгоритм запуска в продакшене

1. n8n доступен по домену (например `https://n8n.example.com`)
2. Workflow в n8n переведены в `Active`
3. Telegram webhook указывает на прод URL:
   - `https://n8n.example.com/webhook/telegram-connect`
4. Backend шлет события в:
   - `https://n8n.example.com/webhook/gateway/events`
5. Все секреты (gateway/internal key) заданы в `.env`

## 5) Что обязательно держать в env

### Основные
- `INTERNAL_NOTIFICATIONS_API_KEY`
- `N8N_GATEWAY_SECRET`
- `TELEGRAM_LINK_CONFIRM_URL` — **полный URL Nest**: `.../internal/integrations/telegram/confirm` (n8n не ходит в Go напрямую)
- `HABITS_API_BASE_URL` — для **nest_satellite**: базовый URL Go (`http://habits_api:8080` в Docker)

### Telegram (разделение админ/пользователь)
- Legacy admin:
  - `TELEGRAM_BOT_TOKEN`
  - `TELEGRAM_CHAT_ID`
- User bot:
  - `TELEGRAM_USER_BOT_TOKEN`
  - `TELEGRAM_USER_BOT_USERNAME`
  - `TELEGRAM_USER_CHAT_ID`

## 6) Как избежать конфликтов

- Не смешивать test/prod URL:
  - test: `/webhook-test/...`
  - prod: `/webhook/...` (требует Active)
- Не смешивать старые и новые workflow:
  - оставить только `main-gateway` + `telegram-user-bind-start`
- Не дублировать переменные под разными именами без назначения
- Не использовать localhost для внешних webhook (Telegram/GitHub)

## 7) Что нужно доделать до "каждый пользователь привязывает свой Telegram"

1. Endpoint статуса в backend:
   - `GET /api/v1/integrations/telegram/status`
2. UI-статус:
   - "Telegram подключен/не подключен"
3. Endpoint отключения:
   - `POST /api/v1/integrations/telegram/disconnect`
4. Перевести bind-хранение полностью на `user_integrations`
   - миграция данных из `telegram_user_links` (или оставить как provider-specific)

## 8) Как добавить GitHub к задачам (следующий этап)

### Минимальный сценарий
1. В `main-gateway` добавить ветку `event_type = task.created`
2. Нода GitHub:
   - create issue или create comment
3. В payload от backend передавать:
   - `repo`, `title`, `body`, `labels`, `assignee`

### Обратная синхронизация GitHub -> ERP
1. GitHub webhook -> n8n
2. n8n -> Backend endpoint `/integrations/github/events`
3. Backend обновляет задачу/сделку

## 9) Команды из Telegram: создать задачу/сдвинуть сделку

Да, это реально. Рекомендуемый подход:

1. Telegram update -> n8n webhook
2. n8n парсит команду (`/task`, `/deal`)
3. n8n вызывает backend API
4. Backend применяет бизнес-правила и права доступа
5. n8n отправляет ответ в Telegram

### Как понять "кто пользователь" в Telegram

При `/start <token>` сохраняем связь:
- `telegram_user_id/chat_id` <-> `user_id`

Потом любая команда из этого chat_id выполняется от этого `user_id`.

Важно:
- без этой привязки команды выполнять нельзя
- все изменения должны идти через backend (не напрямую в БД из n8n)

## 10) Практический roadmap

1. Закрепить 2 workflow (`main-gateway`, `telegram-user-bind-start`)
2. Сделать status/disconnect в backend + UI
3. Перевести business-события приложения на `main-gateway`
4. Добавить 1 Telegram command (`/task create ...`)
5. Добавить 1 GitHub-сценарий (`task.created -> GitHub issue`)
