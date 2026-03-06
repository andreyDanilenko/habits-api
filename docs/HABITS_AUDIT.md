# Аудит модуля Habits

Подробный анализ архитектуры, сервиса и репозиториев модуля привычек с рекомендациями по улучшению.

---

## 1. Обзор архитектуры

### 1.1 Текущая структура слоёв

```
Handler (HTTP) → Service (бизнес-логика) → Repository (доступ к данным)
                     ↓
              WorkspaceRepository (для проверки Owner)
```

**Плюсы:**
- Чёткое разделение: handler парсит запросы, service содержит бизнес-правила, repository — SQL.
- Service не знает про HTTP/JSON.
- Зависимости инжектятся (repo, wsRepo).

**Минусы:**
- Repository — «толстый»: в одном `Repository` объединены habits, completions, versions, activity, stats. Нарушается Single Responsibility.
- Нет интерфейсов для репозиториев — сложнее тестировать (нужны моки).

---

## 2. Анализ Service (`internal/service/habits/service.go`)

### 2.1 Что сделано хорошо

| Аспект | Описание |
|--------|----------|
| **Валидация workspace** | `ErrWorkspaceNeeded` при пустом workspaceID |
| **Права доступа** | Get — любой участник workspace; Update — только владелец; Delete — владелец или Owner workspace |
| **Активность** | Запись в `activities` при Create, Update, Delete, Complete |
| **Идемпотентность** | Get перед мутирующими операциями — проверка существования |

### 2.2 Проблемные места

#### 2.2.1 Игнорирование ошибок при записи активности

```go
_ = s.repo.CreateActivity(ctx, uid, wid, hid, habitsRepo.ActivityTypeHabitCreated, ...)
```

**Проблема:** Если запись активности упадёт (БД недоступна, constraint), пользователь всё равно получит успешный ответ. Активность потеряется.

**Рекомендация:**
- **Вариант A (простой):** Логировать ошибку, но не возвращать её пользователю (активность — не критичная часть).
- **Вариант B (надёжный):** Вынести запись активности в фоновую очередь (см. раздел 6).

#### 2.2.2 Дублирование парсинга UUID

В каждом методе:
```go
hid, _ := uuid.Parse(habitID)
uid, _ := uuid.Parse(userID)
wid, _ := uuid.Parse(workspaceID)
```

**Проблема:** `uuid.Parse` может вернуть ошибку, но она игнорируется (`_`). Невалидный UUID приведёт к панике или некорректному поведению.

**Рекомендация:** Парсить один раз в handler, передавать в service уже `uuid.UUID`. Либо в service возвращать `ErrBadRequest` при ошибке парсинга.

#### 2.2.3 Несогласованность ошибок

- `Update` при попытке редактировать чужую привычку возвращает `ErrHabitNotFound` (маскирует 403).
- `Delete` возвращает `ErrCannotDelete` — корректно.

**Рекомендация:** Ввести `ErrForbidden` или `ErrNotHabitOwner` для Update, чтобы handler мог вернуть 403.

---

## 3. Анализ Repository

### 3.1 Repository (`repository.go`) — «God Object»

Один репозиторий отвечает за:
- CRUD привычек
- Версионирование (через `versions`)
- Completions (через `completions`)
- Activity
- Stats
- Calendar

**Рекомендация:** Разбить на:
- `HabitRepository` — habits + versions
- `CompletionRepository` — уже вынесен, но вызывается из основного repo
- `ActivityRepository` — activity
- `StatsCalculator` — уже отдельно (stateless)

### 3.2 GetCalendar — N+1 и нагрузка

```go
for !current.After(normalizedEnd) {
    dayHabits, err := r.GetHabitsForDate(ctx, workspaceID, current)
    // ...
    current = current.AddDate(0, 0, 1)
}
```

**Проблема:** Для календаря на месяц (31 день) вызывается `GetHabitsForDate` 31 раз. Каждый вызов — отдельный запрос к БД (habit_versions или habits).

**Рекомендация:**
- Добавить `GetHabitsForDateRange(ctx, workspaceID, start, end)` — один запрос с `BETWEEN` и группировкой по дате в приложении.
- Либо материализовать календарь в кэш (Redis) с TTL 5–15 минут.

### 3.3 GetHabitsForDate — два источника правды

Логика:
1. Сначала запрос к `habit_versions`
2. Если пусто и дата >= сегодня — fallback на `habits`

**Проблема:** Дублирование условий расписания в двух запросах. При изменении логики легко забыть обновить оба места.

**Рекомендация:** Вынести условие расписания в общую функцию/константу или использовать VIEW в БД.

### 3.4 Create — версионирование при сбое

```go
if err := r.versions.Create(...); err != nil {
    log.Printf("Error creating habit version: %v", err)
}
return &habit, nil  // привычка создана, версия — нет
```

**Проблема:** Привычка создана, но версия не записана. Для «сегодня» сработает fallback на `habits`, но для прошлых дат календаря привычка может не отображаться.

**Рекомендация:** Либо делать в одной транзакции (INSERT habit + INSERT version), либо откатывать создание привычки при ошибке версии.

### 3.5 Update — сложная логика версий

При `shouldVersion`:
1. Закрывается предыдущая версия (`valid_to = today`)
2. Если версий не было — создаётся backfill из `oldHabit`
3. Создаётся новая версия с `valid_from = tomorrow`

**Проблема:** Много ветвлений, легко допустить ошибку. `oldHabit` может быть nil при race (привычка удалена между Get и Update).

**Рекомендация:** Обернуть в транзакцию; добавить тесты на граничные случаи (обновление в первый день, смена расписания и т.д.).

---

## 4. CompletionRepository

### 4.1 Toggle — race condition

```go
// Путь: completion не найден
completion, err := r.Create(ctx, habitID, userID, normalizedDate, "", nil, nil)
if err != nil {
    return false, nil, err
}
tx.Commit()  // tx использовался только для SELECT
```

**Проблема:** `Create` использует `r.db`, а не `tx`. Между `SELECT` (no rows) и `Create` другой запрос может вставить completion. Получим `UNIQUE (habit_id, date, user_id)` violation.

**Рекомендация:**
- Вариант A: Делать INSERT внутри той же транзакции (передавать `tx` в Create или дублировать логику).
- Вариант B: Использовать `INSERT ... ON CONFLICT (habit_id, date, user_id) DO NOTHING` и проверять `RowsAffected`.
- Вариант C: Ловить `pq.Error` с кодом unique_violation и трактовать как «уже есть» — вернуть `(false, existing, nil)`.

### 4.2 GetAllByWorkspaceAndDateRange — фильтр по user_id

```go
WHERE user_id = $1 AND workspace_id = $2 AND date BETWEEN $3 AND $4
```

**Проблема:** Возвращаются только completions текущего пользователя. Но в workspace могут быть привычки других участников, и они могут выполнять чужие привычки. Календарь использует `GetCompletionMap`, который фильтрует по `h.user_id = c.user_id` (владелец привычки). А `GetAllByWorkspaceAndDateRange` — по `user_id` запроса. Это разные сценарии: «мои выполнения» vs «все выполнения в workspace». Нужно уточнить контракт API.

---

## 5. Activity

### 5.1 Синхронная запись

Каждый Create/Update/Delete/Complete синхронно пишет в `activities`. Это увеличивает latency ответа.

### 5.2 ListActivities — COUNT(*) при пагинации

```go
SELECT COUNT(*) FROM activities WHERE workspace_id = $1
```

**Проблема:** На больших объёмах COUNT(*) по всей таблице может быть медленным.

**Рекомендация:**
- Для «бесконечного скролла» можно обойтись без total (проверять `len(list) < limit`).
- Для постраничной навигации — рассмотреть приблизительный count или материализованное представление.

---

## 6. Очереди и асинхронность

### 6.1 Когда имеет смысл очередь

| Операция | Критичность | Кандидат на очередь? |
|----------|-------------|------------------------|
| CreateActivity | Низкая | Да |
| CreateCompletionActivity | Низкая | Да |
| Create habit version | Средняя | Опционально (если версии тяжёлые) |
| Completion Create | Высокая | Нет |
| Habit Create/Update/Delete | Высокая | Нет |

### 6.2 Архитектура с очередью (RabbitMQ / Redis Streams / pg_notify)

```
Service.Complete() →
  1. repo.Complete()           // синхронно, в транзакции
  2. Publish("activity.completion", payload)  // в очередь
  return completion

Worker:
  Consume("activity.completion") →
    CreateCompletionActivity(payload)
```

**Плюсы:**
- Ответ пользователю не ждёт запись в activities.
- При падении worker сообщения не теряются (persistent queue).
- Можно батчить записи активности.

**Минусы:**
- Усложнение инфраструктуры.
- Нужна доставка «at least once» и идемпотентность обработки.

### 6.3 Простая альтернатива — goroutine

```go
go func() {
    _ = s.repo.CreateCompletionActivity(ctx, uid, wid, hid, title, emoji)
}()
```

**Плюсы:** Просто, без новых компонентов.  
**Минусы:** При падении процесса активность теряется; при перезапуске во время выполнения — тоже.

---

## 7. Оптимизации

### 7.1 Индексы

Проверить наличие индексов:

- `habits(workspace_id)`
- `habit_versions(workspace_id, valid_from, valid_to)`
- `habit_completions(habit_id, user_id, date)` — есть
- `activities(workspace_id, created_at DESC)` — для пагинации

### 7.2 Кэширование

| Данные | Кандидат | TTL |
|--------|----------|-----|
| Список привычек на сегодня | Redis | 1–5 мин |
| Календарь на месяц | Redis | 5–15 мин |
| Stats по привычке | Redis | 5 мин |
| Activities (последние 20) | Redis | 1 мин |

**Инвалидация:** При Create/Update/Delete/Complete — сбрасывать ключи по `workspace_id` и `habit_id`.

### 7.3 Батчинг GetHabitsForDate

Вместо 31 запроса в GetCalendar — один запрос с диапазоном дат и разбором результата в коде.

---

## 8. Чистота кода и поддерживаемость

### 8.1 Магические строки

- `"recurring"`, `"one_time"` — вынести в константы.
- Эмодзи `"➕"`, `"✏️"`, `"🗑️"`, `"✅"` — в конфиг или константы.

### 8.2 Жёстко заданные тексты

```go
"Создана привычка \""+habit.Title+"\""
```

**Рекомендация:** Вынести в `i18n` или хотя бы в константы, если планируется мультиязычность.

### 8.3 Форматирование в repository

В `repository.go` строка 86 — лишний отступ (табуляция):

```go
		todayStart := NormalizeDate(time.Now().UTC())
```

Исправить на один уровень с `habits, err := ...`.

---

## 9. Безопасность

### 9.1 SQL-инъекции

Используются параметризованные запросы (`$1`, `$2`) — хорошо.

### 9.2 Масштабирование по workspace

Проверка `workspace_id` выполняется на уровне запросов. Middleware проверяет доступ к workspace до вызова handler — ок.

---

## 10. Резюме рекомендаций

### Высокий приоритет

1. **Toggle race condition** — исправить логику Create в Toggle (транзакция или ON CONFLICT).
2. **Create habit + version** — делать в одной транзакции или откатывать при ошибке версии.
3. **GetCalendar N+1** — один запрос для диапазона дат вместо цикла.

### Средний приоритет

4. Разделить Repository на несколько сущностей.
5. Ввести интерфейсы для тестирования.
6. Обработка ошибок записи активности (логирование или очередь).
7. Разные типы ошибок для Update (403 vs 404).

### Низкий приоритет

8. Кэширование (Redis) для календаря и списков.
9. Очередь для записи активности.
10. Константы для строк и эмодзи.

---

## 11. Диаграмма зависимостей (как есть)

```
Handler
   └── Service
         ├── HabitRepository (habits + versions + completions + activity + stats)
         └── WorkspaceRepository (IsOwner)
```

## 12. Диаграмма зависимостей (рекомендуемая)

```
Handler
   └── Service
         ├── HabitRepository      (habits, versions)
         ├── CompletionRepository (completions)
         ├── ActivityRepository   (activities)
         ├── StatsCalculator      (stateless)
         └── WorkspaceRepository  (IsOwner)

Опционально:
   └── ActivityQueue (publish) → Worker → ActivityRepository
```

---

*Документ подготовлен для обучения и улучшения архитектуры модуля Habits.*
