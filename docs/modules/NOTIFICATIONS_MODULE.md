# Универсальный модуль уведомлений

## Назначение

Хранение и отображение уведомлений для каждого пользователя отдельно. Поддержка разных каналов: `activity`, `chat`, `system` и т.д.

## API

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/me/notifications` | Список уведомлений (query: `channel`, `unreadOnly`, `limit`, `offset`) |
| POST | `/api/v1/me/notifications` | Создать/обновить (идемпотентно по `event_key`) |
| PATCH | `/api/v1/me/notifications/:id/read` | Отметить прочитанным |
| POST | `/api/v1/me/notifications/mark-all-read` | Отметить все прочитанными (query: `channel`) |

## Схема

- `channel` — канал: `activity`, `chat`, `system`
- `event_key` — уникальный ключ события для идемпотентности (например `deal.updated:uuid`, `chat:room:msg:id`)
- `read_at` — время прочтения (NULL = непрочитано)

## Расширение для чатов

Для чатов можно использовать:
- `channel: "chat"`
- `event_type: "chat.message"`
- `event_key: "chat:{roomId}:{messageId}"`
- `payload` — JSON с данными сообщения

Аналогично для других модулей.
