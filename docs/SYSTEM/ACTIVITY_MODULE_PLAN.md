# План модуля Activity: Habits и асинхронные брокеры

## 1. Текущее состояние

### 1.1. Два источника активности

| Источник | Таблица | Сущности | Типы событий |
|----------|--------|----------|--------------|
| **Habits** | `activities` | habit, completion, workspace | HABIT_CREATED, HABIT_UPDATED, HABIT_DELETED, HABIT_COMPLETED |
| **CRM** | `crm_activities` | contact, company, deal | note, call, email, task, deal_stage_changed, … |

### 1.2. API

- **CRM**: `GET/POST /workspaces/:id/crm/activities` — привязано к entity (contact, company, deal)
- **Habits**: таблица `activities` есть, но API для чтения/записи активности habits не реализован

### 1.3. Фронтенд

- `RecentActivityWidget` — использует mock-данные, не подключён к API
- CRM Activity — `ActivityFeed`, `ActivityComposer` — работают с `crm_activities`

---

## 2. Доработка Activity для Habits

### 2.1. Задачи

1. **API для habits-активности**
   - `GET /workspaces/:id/activities` — лента активности по workspace (habits + опционально CRM)
   - Параметры: `entityType` (habit|completion|all), `limit`, `offset`, `dateFrom`, `dateTo`
   - Использовать таблицу `activities` (migration 000009)

2. **Запись активности при действиях**
   - При создании/обновлении/удалении привычки — `CreateSystemActivity` в `activities`
   - При завершении привычки (HabitCompletion) — `HABIT_COMPLETED`
   - Сейчас: нужно проверить, вызывается ли это в habits service

3. **RecentActivityWidget**
   - Подключить к `GET /workspaces/:id/activities`
   - Объединить habits + CRM в одну ленту (опционально) или показывать только habits

### 2.2. Модель `activities`

```sql
-- Из 000009_create_activities.up.sql
CREATE TABLE activities (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL,  -- HABIT_CREATED, HABIT_UPDATED, HABIT_DELETED, HABIT_COMPLETED
    entity_type VARCHAR(50) NOT NULL,  -- habit, completion, workspace
    entity_id VARCHAR(255),
    title VARCHAR(500) NOT NULL,
    emoji VARCHAR(10),
    created_at TIMESTAMP NOT NULL
);
```

---

## 3. Асинхронные брокеры: подготовка к масштабированию

### 3.1. Зачем

- Запись активности — не критична для ответа пользователю
- При высокой нагрузке синхронная запись в БД увеличивает latency
- Асинхронная публикация в очередь → воркер пишет в БД — разгружает API

### 3.2. Варианты

| Брокер | Плюсы | Минусы |
|--------|-------|--------|
| **RabbitMQ** | Надёжность, гибкая маршрутизация | Нужен отдельный сервис |
| **Redis Streams** | Простота, уже может быть в стеке | Меньше гарантий доставки |
| **AWS SQS** | Managed, масштабируемость | Привязка к AWS |
| **NATS** | Лёгкий, быстрый | Меньше экосистемы |

### 3.3. Архитектура (подготовка)

```
[Handler] → [ActivityPublisher interface]
                 ↓
         [SyncPublisher]  — текущее поведение: пишем в БД сразу
         [RabbitPublisher] — публикуем в очередь, воркер пишет в БД
```

**Интерфейс:**

```go
type ActivityPublisher interface {
    PublishHabitActivity(ctx context.Context, wsID, userID string, evt HabitActivityEvent) error
    PublishCrmActivity(ctx context.Context, wsID string, evt CrmActivityEvent) error
}
```

**Реализации:**

- `SyncActivityPublisher` — вызывает repo напрямую (текущее поведение)
- `RabbitActivityPublisher` — публикует в exchange `activity.events`, routing key `habit.*` / `crm.*`

**Воркер (отдельный процесс):**

- Подписывается на очередь
- Получает сообщения → вызывает repo для записи в БД
- При ошибке — retry / dead-letter queue

### 3.4. Этапы внедрения

1. **Этап 1**: Ввести интерфейс `ActivityPublisher`, реализация `SyncActivityPublisher`
2. **Этап 2**: Реализовать API для habits-активности, подключить RecentActivityWidget
3. **Этап 3**: Добавить `RabbitActivityPublisher` и воркер (опционально, при росте нагрузки)

---

## 4. Middleware vs Dependency Injection для Activity

### 4.1. Вариант A: Middleware

**Идея:** Middleware перехватывает ответы хендлеров и по коду ответа/телу запроса решает, нужно ли публиковать событие активности.

**Минусы:**

- Сложно извлекать бизнес-контекст (какая привычка создана, какой контакт обновлён)
- Дублирование логики: middleware должен знать маппинг `(path, method) → тип события`
- Трудно тестировать

### 4.2. Вариант B: Dependency Injection в хендлеры

**Идея:** Хендлер получает `ActivityPublisher` через DI. После успешной операции явно вызывает `publisher.PublishHabitActivity(...)`.

**Плюсы:**

- Явный контроль: хендлер знает, какое событие публиковать
- Легко тестировать (mock publisher)
- Гибкость: можно передавать метаданные (entity_id, title, emoji)

**Рекомендация:** **Dependency Injection в хендлеры.**

Middleware подходит для сквозных задач (логирование, метрики, auth). Публикация активности — бизнес-логика, её лучше держать в хендлерах/сервисах.

### 4.3. Гибрид (опционально)

Сервисный слой (HabitsService, CrmService) вызывает `ActivityPublisher`. Хендлеры не знают об активности — вся логика в сервисе. Это тоже вариант DI, но на уровне сервиса, а не хендлера.

---

## 5. Связь с другими документами

- [ROLES_BACKEND_ARCHITECTURE.md](./ROLES_BACKEND_ARCHITECTURE.md) — права на `crm:activity:*`
- [SCALING_AND_IMPROVEMENT_PLAN.md](../plans/SCALING_AND_IMPROVEMENT_PLAN.md) — общий план масштабирования
