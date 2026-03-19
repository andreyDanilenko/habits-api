# Гайд: Масштабирование SQL-миграций и управление связями

**Цель:** Понять, как безопасно развивать схему БД, управлять FK, чистить данные и готовиться к микросервисам.

---

## 1. Два типа миграций в проекте

| Тип | Папка | Назначение |
|-----|-------|------------|
| **Инкрементальные** | `migrations/000001_*.sql` … `000032_*.sql` | Разработка: добавляют изменения поверх существующей БД |
| **Clean baseline** | `migrations/clean_baseline/001_*.sql` … `027_*.sql` | Продакшен: создание БД с нуля по сущностям |

**Важно:** При любых изменениях в инкрементальных миграциях нужно обновлять соответствующий файл в `clean_baseline/`. См. `migrations/README.md` (таблица соответствия).

---

## 2. Как избежать «мусора» миграций

### 2.1. Проблема

Со временем накапливаются десятки миграций (000001–000050). Сложно:

- Понимать историю
- Делать squash (объединение)
- Создавать чистую БД с нуля

### 2.2. Решение: baseline как источник правды

```
┌─────────────────────────────────────────────────────────────────┐
│  РАЗРАБОТКА (инкрементальные)                                    │
│  000001, 000002, ... 000032  — применяются по очереди            │
│  constraints/ — FK, индексы                                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼  при каждом релизе
┌─────────────────────────────────────────────────────────────────┐
│  CLEAN BASELINE (продакшен с нуля)                               │
│  001–027 по сущностям — финальная схема без истории              │
└─────────────────────────────────────────────────────────────────┘
```

### 2.3. Правила

1. **Clean baseline** — всегда актуален и отражает финальную схему.
2. **Инкрементальные** — только для уже существующих БД (разработка, staging).
3. **Squash** — периодически (раз в квартал или перед крупным релизом) объединять старые миграции в новый baseline и обнулять счётчик.

### 2.4. Squash-цикл

```
Было: 000001..000050
Сделали: baseline = финальная схема
Новый цикл: 000001_squashed = baseline, 000002 = новая фича
```

---

## 3. Управление FK (внешние ключи)

### 3.1. Добавление FK

```sql
-- 1. Проверить сироты (если FK NOT NULL)
SELECT COUNT(*) FROM habit_completions hc
WHERE NOT EXISTS (SELECT 1 FROM habits h WHERE h.id = hc.habit_id);

-- 2. Почистить или исправить
DELETE FROM habit_completions WHERE habit_id NOT IN (SELECT id FROM habits);

-- 3. Добавить FK
ALTER TABLE habit_completions
  ADD CONSTRAINT fk_completions_habit
  FOREIGN KEY (habit_id) REFERENCES habits(id) ON DELETE CASCADE;
```

### 3.2. Удаление FK

```sql
ALTER TABLE habit_completions DROP CONSTRAINT IF EXISTS fk_completions_habit;
```

### 3.3. Безопасный порядок при массовых изменениях

1. Удалить все FK.
2. Менять схему (колонки, таблицы).
3. Почистить данные.
4. Добавить FK заново.

### 3.4. Правила в проекте (см. SQL_AND_MIGRATIONS_GUIDE.md)

| Связь | FK | Причина |
|-------|-----|---------|
| Внутри модуля (crm_stages → crm_pipelines) | Да | Целостность внутри своей БД |
| Модуль → Core (users, workspaces) | Нет | Задел под выделение в микросервис |
| Между разными модулями | Нет | Связь через project_entities или оркестратор |

---

## 4. Очистка сирот (без FK)

Если FK нет, сироты можно искать и удалять вручную.

### 4.1. Шаблон запроса

```sql
-- Сироты в habit_completions (habit_id не существует)
DELETE FROM habit_completions hc
WHERE NOT EXISTS (SELECT 1 FROM habits h WHERE h.id = hc.habit_id);

-- Сироты в habits (user_id не существует)
DELETE FROM habits h
WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.id = h.user_id);
```

### 4.2. Порядок очистки (каскад)

Сначала дети, потом родители:

1. `habit_completions` (ссылается на habits)
2. `habit_history`
3. `habits`
4. `user_workspaces`

### 4.3. Скрипт для генерации DELETE

```sql
-- Шаблон для каждой пары (child_table, parent_table)
SELECT format(
  'DELETE FROM %I WHERE %I NOT IN (SELECT id FROM %I);',
  'habit_completions', 'habit_id', 'habits'
);
```

---

## 5. Декомпозиция: таблицы без связей + constraints в конце

### 5.1. Идея

- **Таблицы** — только `CREATE TABLE` без FK, UNIQUE, индексов.
- **Связи** — один файл с `ALTER TABLE`, который всегда применяется последним.

### 5.2. Преимущества

| Аспект | Результат |
|--------|-----------|
| Clean baseline | Всегда чистые таблицы, без зависимостей |
| Новые таблицы | Добавляешь только `CREATE TABLE` с колонками |
| Новые связи | Добавляешь только `ALTER` в конец constraints-файла |
| Порядок | Не важен порядок создания таблиц (нет FK при создании) |

### 5.3. Пример

**CREATE TABLE (без FK):**

```sql
CREATE TABLE user_workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    role VARCHAR(50) DEFAULT 'MEMBER',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Constraints (отдельно):**

```sql
ALTER TABLE user_workspaces
  ADD CONSTRAINT fk_user_workspaces_user
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_workspaces
  ADD CONSTRAINT fk_user_workspaces_workspace
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;

ALTER TABLE user_workspaces
  ADD CONSTRAINT uq_user_workspaces_user_workspace
  UNIQUE (user_id, workspace_id);

CREATE INDEX idx_user_workspaces_user_id ON user_workspaces(user_id);
CREATE INDEX idx_user_workspaces_workspace_id ON user_workspaces(workspace_id);
```

---

## 6. Микросервисы и разделение по приложениям

### 6.1. Вариант A: одна БД, разные схемы

```
schema habits_app:  users, habits, habit_completions  (без workspace)
schema crm_app:     users, workspaces, crm_*           (с workspace)
schema shared:      users (если общий)
```

- Один PostgreSQL, разные `schema`.
- Удобно для постепенного разделения.

### 6.2. Вариант B: отдельные БД

```
habits_db:  users, habits, habit_completions
crm_db:     users, workspaces, crm_contacts, crm_deals
```

- Полная изоляция.
- Сложнее: синхронизация пользователей, транзакции между БД.

### 6.3. Вариант C: опциональный workspace (гибрид)

Добавить `workspace_id` как nullable:

```sql
CREATE TABLE habits (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    workspace_id UUID,  -- NULL = личные привычки (отдельное приложение)
    ...
);
```

- `workspace_id IS NOT NULL` — ERP с workspace.
- `workspace_id IS NULL` — отдельное приложение привычек.

### 6.4. Рекомендация

| Цель | Подход |
|------|--------|
| Один продукт, разные модули | Текущая модель с workspace |
| Отдельное приложение привычек | Вариант C (nullable workspace_id) или отдельная БД |
| Полный микросервисный стек | Отдельные БД + общий auth (OAuth/JWT) |

---

## 7. Чеклист для масштабирования

| Действие | Рекомендация |
|----------|--------------|
| Новая таблица | CREATE TABLE без FK → добавить FK в `constraints/` |
| Удаление таблицы | DROP TABLE + удалить FK в constraints |
| Изменение колонки | ALTER TABLE в отдельной миграции |
| Squash миграций | Раз в N релизов: новый baseline, обнуление счётчика |
| Чистка сирот | Скрипт с `NOT EXISTS` перед добавлением FK |
| Разделение сервисов | Схемы или отдельные БД + общий auth |

---

## 8. Связанные документы

- [SQL_AND_MIGRATIONS_GUIDE.md](./SQL_AND_MIGRATIONS_GUIDE.md) — правила миграций
- [02_DATABASE_AND_TABLES.md](./developer-guide/02_DATABASE_AND_TABLES.md) — схема БД и связи
- [SCALING_AND_IMPROVEMENT_PLAN.md](../plans/SCALING_AND_IMPROVEMENT_PLAN.md) — план выноса модулей
