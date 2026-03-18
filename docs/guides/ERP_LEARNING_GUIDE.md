# 🎓 Полный гайд по обучению и развитию ERP системы

## 📋 Содержание

1. [План развития ERP по модулям](#план-развития-erp-по-модулям)
2. [Миграции базы данных - полное руководство](#миграции-базы-данных)
3. [Алгоритм работы с моделями](#алгоритм-работы-с-моделями)
4. [Управление зависимостями](#управление-зависимостями)
5. [Таблицы связей (Junction Tables)](#таблицы-связей)
6. [Обратная совместимость](#обратная-совместимость)
7. [Транзакции - где и как применять](#транзакции)
8. [Супер-пользователь и модульная система](#супер-пользователь-и-модули)
9. [SQL мастер-класс](#sql-мастер-класс)
10. [Практические примеры](#практические-примеры)

---

## 🗺️ План развития ERP по модулям

### Этап 1: Основа (Уже реализовано ✅)

#### 1.1. Пользователи (Users)
**Порядок разработки:**
1. ✅ Создать миграцию `000002_create_users.up.sql`
2. ✅ Создать модель `internal/model/user.go`
3. ✅ Создать repository `internal/repository/user/repository.go`
4. ✅ Создать service `internal/service/auth/service.go`
5. ✅ Создать handler `internal/handler/auth/handler.go`

**Что изучили:**
- Базовая структура таблицы
- Индексы для быстрого поиска
- Хеширование паролей
- JWT токены

---

#### 1.2. Workspaces (Рабочие пространства)
**Порядок разработки:**
1. ✅ Миграция `000005_create_workspaces.up.sql`
2. ✅ Модель `internal/model/workspace.go`
3. ✅ Repository, Service, Handler

**Ключевые моменты:**
- `owner_id` → связь с users
- Триггер для `updated_at`
- Workspace как контейнер для модулей

---

#### 1.3. User_Workspaces (Связь пользователей и workspace)
**Порядок разработки:**
1. ✅ Миграция `000006_create_user_workspaces.up.sql`
2. ✅ Junction table (таблица связей)

**Что это:**
- Многие-ко-многим связь
- `UNIQUE(user_id, workspace_id)` - один пользователь не может быть дважды в одном workspace
- Роли: OWNER, MEMBER, ADMIN

---

### Этап 2: Первый модуль - Habits (Уже реализовано ✅)

**Порядок разработки:**

1. **Миграция основной таблицы:**
   ```sql
   -- 000003_create_habits.up.sql
   CREATE TABLE habits (
       id UUID PRIMARY KEY,
       title VARCHAR(255) NOT NULL,
       user_id UUID NOT NULL,
       workspace_id UUID NOT NULL,
       -- ... остальные поля
   );
   ```

2. **Миграция связанной таблицы:**
   ```sql
   -- 000004_create_habit_completions.up.sql
   CREATE TABLE habit_completions (
       id UUID PRIMARY KEY,
       habit_id UUID NOT NULL,
       user_id UUID NOT NULL,
       date DATE NOT NULL,
       -- ...
   );
   ```

3. **Миграция связей (Foreign Keys):**
   ```sql
   -- constraints/01_foreign_keys.up.sql
   ALTER TABLE habits 
   ADD CONSTRAINT fk_habits_user 
   FOREIGN KEY (user_id) REFERENCES users(id);
   ```

4. **Модели:**
   - `internal/model/habits.go` - Habit, HabitCompletion, DTOs

5. **Repository:**
   - `internal/repository/habits/repository.go`
   - Методы: List, Create, Update, Delete, Complete, Toggle

6. **Service:**
   - `internal/service/habits/service.go`
   - Бизнес-логика, валидация

7. **Handler:**
   - `internal/handler/habits/handler.go`
   - HTTP endpoints

---

### Этап 3: Второй модуль - Journal (В разработке)

**Порядок разработки:**

#### Шаг 1: Планирование

**Вопросы для себя:**
1. Какие данные хранит Journal?
   - Записи по датам
   - Текст, теги, настроение
   - Привязка к workspace

2. Какие связи нужны?
   - Journal → Workspace (1:N)
   - Journal → User (1:N)
   - Journal → Tags (N:M через junction table)

3. Какие операции нужны?
   - Создание записи
   - Получение по дате
   - Обновление
   - Удаление
   - Поиск по тегам

#### Шаг 2: Создание миграций

**Миграция 1: Основная таблица**
```sql
-- 000010_create_journals.up.sql
CREATE TABLE journals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    date DATE NOT NULL,
    title VARCHAR(255),
    content TEXT NOT NULL,
    mood VARCHAR(50), -- 'happy', 'sad', 'neutral'
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Одна запись на день для пользователя в workspace
    UNIQUE(user_id, workspace_id, date)
);

CREATE INDEX idx_journals_user_workspace ON journals(user_id, workspace_id);
CREATE INDEX idx_journals_date ON journals(date);
CREATE INDEX idx_journals_created_at ON journals(created_at DESC);
```

**Миграция 2: Таблица тегов (если нужна)**
```sql
-- 000011_create_journal_tags.up.sql
CREATE TABLE journal_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    color VARCHAR(50) DEFAULT '#3B82F6',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Миграция 3: Junction table для связи Journal ↔ Tags**
```sql
-- 000012_create_journal_tag_links.up.sql
CREATE TABLE journal_tag_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    UNIQUE(journal_id, tag_id)
);

CREATE INDEX idx_journal_tag_links_journal ON journal_tag_links(journal_id);
CREATE INDEX idx_journal_tag_links_tag ON journal_tag_links(tag_id);
```

**Миграция 4: Foreign Keys**
```sql
-- constraints/03_journal_foreign_keys.up.sql
ALTER TABLE journals 
ADD CONSTRAINT fk_journals_user 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE journals 
ADD CONSTRAINT fk_journals_workspace 
FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;

ALTER TABLE journal_tag_links 
ADD CONSTRAINT fk_journal_tag_links_journal 
FOREIGN KEY (journal_id) REFERENCES journals(id) ON DELETE CASCADE;

ALTER TABLE journal_tag_links 
ADD CONSTRAINT fk_journal_tag_links_tag 
FOREIGN KEY (tag_id) REFERENCES journal_tags(id) ON DELETE CASCADE;
```

#### Шаг 3: Модели

```go
// internal/model/journal.go
package model

type Journal struct {
    ID          string   `json:"id" db:"id"`
    UserID      string   `json:"userId" db:"user_id"`
    WorkspaceID string   `json:"workspaceId" db:"workspace_id"`
    Date        string   `json:"date" db:"date"` // "2006-01-02"
    Title       string   `json:"title,omitempty" db:"title"`
    Content     string   `json:"content" db:"content"`
    Mood        string   `json:"mood,omitempty" db:"mood"`
    Tags        []Tag    `json:"tags,omitempty"` // Загружается отдельным запросом
    CreatedAt   string   `json:"createdAt" db:"created_at"`
    UpdatedAt   string   `json:"updatedAt" db:"updated_at"`
}

type JournalTag struct {
    ID        string `json:"id" db:"id"`
    Name      string `json:"name" db:"name"`
    Color     string `json:"color" db:"color"`
    CreatedAt string `json:"createdAt" db:"created_at"`
}

type CreateJournalDto struct {
    Date    string   `json:"date" binding:"required"`
    Title   string   `json:"title,omitempty"`
    Content string   `json:"content" binding:"required"`
    Mood    string   `json:"mood,omitempty"`
    TagIDs  []string `json:"tagIds,omitempty"`
}
```

#### Шаг 4: Repository

```go
// internal/repository/journal/repository.go
package journal

import (
    "context"
    "database/sql"
    "time"
    "backend/internal/model"
    "github.com/google/uuid"
)

type Repository struct {
    db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
    return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, dto model.CreateJournalDto, userID, workspaceID uuid.UUID) (*model.Journal, error) {
    // Начинаем транзакцию для атомарности
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()
    
    // 1. Создаем запись журнала
    journalID := uuid.New()
    now := time.Now()
    
    query := `
        INSERT INTO journals (
            id, user_id, workspace_id, date, title, content, mood, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id, user_id, workspace_id, date, title, content, mood, created_at, updated_at
    `
    
    var journal model.Journal
    var createdAt, updatedAt time.Time
    
    err = tx.QueryRowContext(ctx, query,
        journalID, userID, workspaceID, dto.Date, dto.Title, dto.Content, dto.Mood, now, now,
    ).Scan(
        &journal.ID, &journal.UserID, &journal.WorkspaceID, &journal.Date,
        &journal.Title, &journal.Content, &journal.Mood, &createdAt, &updatedAt,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create journal: %w", err)
    }
    
    // 2. Связываем теги (если есть)
    if len(dto.TagIDs) > 0 {
        for _, tagIDStr := range dto.TagIDs {
            tagID, err := uuid.Parse(tagIDStr)
            if err != nil {
                continue
            }
            
            _, err = tx.ExecContext(ctx,
                "INSERT INTO journal_tag_links (id, journal_id, tag_id, created_at) VALUES ($1, $2, $3, $4)",
                uuid.New(), journalID, tagID, now,
            )
            if err != nil {
                // Логируем ошибку, но не прерываем транзакцию
                log.Printf("Failed to link tag %s: %v", tagIDStr, err)
            }
        }
    }
    
    // Коммитим транзакцию
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    journal.CreatedAt = createdAt.Format(time.RFC3339)
    journal.UpdatedAt = updatedAt.Format(time.RFC3339)
    
    return &journal, nil
}

func (r *Repository) GetByDate(ctx context.Context, userID, workspaceID uuid.UUID, date time.Time) (*model.Journal, error) {
    query := `
        SELECT id, user_id, workspace_id, date, title, content, mood, created_at, updated_at
        FROM journals
        WHERE user_id = $1 AND workspace_id = $2 AND date = $3
    `
    
    var journal model.Journal
    var createdAt, updatedAt time.Time
    
    err := r.db.QueryRowContext(ctx, query, userID, workspaceID, date).Scan(
        &journal.ID, &journal.UserID, &journal.WorkspaceID, &journal.Date,
        &journal.Title, &journal.Content, &journal.Mood, &createdAt, &updatedAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get journal: %w", err)
    }
    
    journal.CreatedAt = createdAt.Format(time.RFC3339)
    journal.UpdatedAt = updatedAt.Format(time.RFC3339)
    
    // Загружаем теги отдельным запросом
    tags, _ := r.getTagsForJournal(ctx, journal.ID)
    journal.Tags = tags
    
    return &journal, nil
}

func (r *Repository) getTagsForJournal(ctx context.Context, journalID uuid.UUID) ([]model.JournalTag, error) {
    query := `
        SELECT t.id, t.name, t.color, t.created_at
        FROM journal_tags t
        INNER JOIN journal_tag_links l ON t.id = l.tag_id
        WHERE l.journal_id = $1
        ORDER BY t.name
    `
    
    rows, err := r.db.QueryContext(ctx, query, journalID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var tags []model.JournalTag
    for rows.Next() {
        var tag model.JournalTag
        var createdAt time.Time
        err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &createdAt)
        if err != nil {
            continue
        }
        tag.CreatedAt = createdAt.Format(time.RFC3339)
        tags = append(tags, tag)
    }
    
    return tags, nil
}
```

---

### Этап 4: Последующие модули

**Шаблон для каждого нового модуля:**

1. **Планирование:**
   - Определить сущности
   - Определить связи
   - Определить операции

2. **Миграции:**
   - Основная таблица
   - Связанные таблицы
   - Junction tables (если нужны)
   - Foreign keys
   - Индексы

3. **Модели:**
   - Основная модель
   - DTOs (Create, Update)
   - Response модели

4. **Repository:**
   - CRUD операции
   - Сложные запросы
   - Транзакции (если нужны)

5. **Service:**
   - Бизнес-логика
   - Валидация
   - Обработка ошибок

6. **Handler:**
   - HTTP endpoints
   - Валидация запросов
   - Обработка ответов

---

## 🗄️ Миграции базы данных

### Что такое миграция?

**Миграция** - это версионированное изменение структуры базы данных.

**Принципы:**
- ✅ Каждая миграция имеет номер (000001, 000002, ...)
- ✅ Каждая миграция имеет `.up.sql` (применить) и `.down.sql` (откатить)
- ✅ Миграции применяются последовательно
- ✅ Миграции не изменяются после применения в production

---

### Структура миграции

#### Базовый шаблон

```sql
-- 000XXX_create_table_name.up.sql
CREATE TABLE table_name (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- поля
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Индексы
CREATE INDEX idx_table_name_field ON table_name(field);

-- Комментарии (опционально, но рекомендуется)
COMMENT ON TABLE table_name IS 'Описание таблицы';
COMMENT ON COLUMN table_name.field IS 'Описание поля';
```

```sql
-- 000XXX_create_table_name.down.sql
DROP TABLE IF EXISTS table_name;
```

---

### Типы миграций

#### 1. Создание таблицы

**Пример:**
```sql
-- 000010_create_products.up.sql
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    workspace_id UUID NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_products_workspace ON products(workspace_id);
CREATE INDEX idx_products_user ON products(user_id);
CREATE INDEX idx_products_created_at ON products(created_at DESC);
```

**Down миграция:**
```sql
-- 000010_create_products.down.sql
DROP TABLE IF EXISTS products;
```

---

#### 2. Добавление поля

**Пример:**
```sql
-- 000011_add_status_to_products.up.sql
ALTER TABLE products 
ADD COLUMN status VARCHAR(50) DEFAULT 'active';

CREATE INDEX idx_products_status ON products(status);

COMMENT ON COLUMN products.status IS 'Статус продукта: active, inactive, archived';
```

**Down миграция:**
```sql
-- 000011_add_status_to_products.down.sql
ALTER TABLE products DROP COLUMN IF EXISTS status;
```

---

#### 3. Изменение типа поля

**⚠️ ВНИМАНИЕ:** Это опасная операция! Нужна обратная совместимость.

**Безопасный способ:**
```sql
-- 000012_change_price_type.up.sql
-- Шаг 1: Добавляем новое поле
ALTER TABLE products 
ADD COLUMN price_new DECIMAL(12, 2);

-- Шаг 2: Копируем данные с преобразованием
UPDATE products 
SET price_new = CAST(price AS DECIMAL(12, 2));

-- Шаг 3: Удаляем старое поле
ALTER TABLE products DROP COLUMN price;

-- Шаг 4: Переименовываем новое поле
ALTER TABLE products RENAME COLUMN price_new TO price;

-- Шаг 5: Добавляем NOT NULL (если нужно)
ALTER TABLE products ALTER COLUMN price SET NOT NULL;
```

**Down миграция:**
```sql
-- 000012_change_price_type.down.sql
-- Обратная операция
ALTER TABLE products 
ADD COLUMN price_old VARCHAR(50);

UPDATE products 
SET price_old = CAST(price AS VARCHAR(50));

ALTER TABLE products DROP COLUMN price;
ALTER TABLE products RENAME COLUMN price_old TO price;
```

---

#### 4. Добавление Foreign Key

**Пример:**
```sql
-- 000013_add_product_category_fk.up.sql
-- Сначала добавляем поле (если его нет)
ALTER TABLE products 
ADD COLUMN category_id UUID;

-- Затем добавляем Foreign Key
ALTER TABLE products 
ADD CONSTRAINT fk_products_category 
FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL;

-- Индекс для Foreign Key (ускоряет JOIN)
CREATE INDEX idx_products_category_id ON products(category_id);
```

**Down миграция:**
```sql
-- 000013_add_product_category_fk.down.sql
ALTER TABLE products DROP CONSTRAINT IF EXISTS fk_products_category;
ALTER TABLE products DROP COLUMN IF EXISTS category_id;
```

---

#### 5. Создание Junction Table

**Пример:**
```sql
-- 000014_create_product_tags.up.sql
CREATE TABLE product_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Предотвращаем дубликаты
    UNIQUE(product_id, tag_id)
);

-- Foreign Keys
ALTER TABLE product_tags 
ADD CONSTRAINT fk_product_tags_product 
FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE;

ALTER TABLE product_tags 
ADD CONSTRAINT fk_product_tags_tag 
FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE;

-- Индексы для быстрого поиска в обе стороны
CREATE INDEX idx_product_tags_product ON product_tags(product_id);
CREATE INDEX idx_product_tags_tag ON product_tags(tag_id);
```

**Down миграция:**
```sql
-- 000014_create_product_tags.down.sql
DROP TABLE IF EXISTS product_tags;
```

---

#### 6. Добавление данных (Seed)

**Пример:**
```sql
-- 000015_seed_default_tags.up.sql
INSERT INTO tags (id, name, color, created_at) VALUES
    (gen_random_uuid(), 'Важное', '#EF4444', NOW()),
    (gen_random_uuid(), 'Работа', '#3B82F6', NOW()),
    (gen_random_uuid(), 'Личное', '#10B981', NOW())
ON CONFLICT DO NOTHING;
```

**Down миграция:**
```sql
-- 000015_seed_default_tags.down.sql
DELETE FROM tags WHERE name IN ('Важное', 'Работа', 'Личное');
```

---

### Порядок применения миграций

**Важно:** Миграции применяются в порядке номеров!

```
000001 → 000002 → 000003 → ... → 000010 → 000011
```

**Если нужно добавить миграцию между существующими:**

❌ **НЕПРАВИЛЬНО:**
```
000001, 000002, 000003, 000003.5 ← НЕ ДЕЛАЙТЕ ТАК!
```

✅ **ПРАВИЛЬНО:**
```
000001, 000002, 000003, 000004 (новая), 000005 (переименовать старую 000003)
```

**Или использовать десятичные номера:**
```
000001, 000002, 000003, 000003_1, 000003_2, 000004
```

---

### Обработка ошибок в миграциях

**Проблема:** Миграция упала посередине

**Решение:** Используйте транзакции в миграциях (PostgreSQL делает это автоматически)

```sql
-- PostgreSQL автоматически оборачивает каждую миграцию в транзакцию
-- Если что-то упало - все откатывается

BEGIN; -- автоматически
CREATE TABLE products (...);
CREATE INDEX ...;
-- Если здесь ошибка - все откатывается
COMMIT; -- автоматически
```

**Но для некоторых операций нужен явный контроль:**

```sql
-- 000016_add_column_with_default.up.sql
-- Некоторые операции не могут быть в транзакции (например, CREATE INDEX CONCURRENTLY)
-- Но для обычных операций транзакция работает

ALTER TABLE products ADD COLUMN new_field VARCHAR(100);
UPDATE products SET new_field = 'default' WHERE new_field IS NULL;
ALTER TABLE products ALTER COLUMN new_field SET NOT NULL;
```

---

### Откат миграций

**Команда:**
```bash
# Откатить последнюю миграцию
migrate -path ./migrations -database "postgres://..." down 1

# Откатить 3 последние миграции
migrate -path ./migrations -database "postgres://..." down 3

# Откатить все до версии 000010
migrate -path ./migrations -database "postgres://..." down 10
```

**⚠️ ВНИМАНИЕ:** В production откатывайте осторожно! Убедитесь, что:
1. Нет активных пользователей
2. Сделали backup БД
3. Понимаете последствия

---

### Best Practices для миграций

1. **Всегда создавайте `.down.sql`**
   - Даже если кажется, что откат не нужен
   - Может понадобиться для разработки

2. **Тестируйте миграции локально**
   ```bash
   # Применить
   migrate up
   # Откатить
   migrate down 1
   # Применить снова
   migrate up
   ```

3. **Используйте комментарии**
   ```sql
   COMMENT ON TABLE products IS 'Товары в системе';
   COMMENT ON COLUMN products.price IS 'Цена в рублях';
   ```

4. **Именуйте миграции понятно**
   ```
   ✅ 000010_create_products.up.sql
   ✅ 000011_add_status_to_products.up.sql
   ❌ 000010_migration.up.sql
   ❌ 000011_fix.up.sql
   ```

5. **Разделяйте большие миграции**
   ```
   ❌ Одна миграция создает 10 таблиц
   ✅ 10 миграций, каждая создает одну таблицу
   ```

6. **Не изменяйте примененные миграции**
   ```
   ❌ Изменить 000010 после применения в production
   ✅ Создать новую миграцию 000020 для исправления
   ```

---

## 🔄 Алгоритм работы с моделями

### Шаг 1: Создание новой модели

#### 1.1. Определить структуру данных

**Вопросы:**
- Какие поля нужны?
- Какие поля обязательные (NOT NULL)?
- Какие поля могут быть NULL?
- Какие типы данных использовать?

**Пример планирования:**
```
Модель: Task (Задача)
Поля:
  - id (UUID) - обязательное
  - title (string) - обязательное
  - description (text) - опциональное
  - status (enum: todo, in_progress, done) - обязательное, default: todo
  - priority (int: 1-5) - опциональное
  - due_date (date) - опциональное
  - user_id (UUID) - обязательное
  - workspace_id (UUID) - обязательное
  - created_at (timestamp) - обязательное
  - updated_at (timestamp) - обязательное
```

#### 1.2. Создать миграцию

```sql
-- 000020_create_tasks.up.sql
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'todo',
    priority INTEGER CHECK (priority >= 1 AND priority <= 5),
    due_date DATE,
    user_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Проверка статуса
    CHECK (status IN ('todo', 'in_progress', 'done'))
);

CREATE INDEX idx_tasks_user_workspace ON tasks(user_id, workspace_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_due_date ON tasks(due_date);
```

#### 1.3. Создать Go модель

```go
// internal/model/task.go
package model

type TaskStatus string

const (
    TaskStatusTodo       TaskStatus = "todo"
    TaskStatusInProgress TaskStatus = "in_progress"
    TaskStatusDone       TaskStatus = "done"
)

type Task struct {
    ID          string     `json:"id" db:"id"`
    Title       string     `json:"title" db:"title"`
    Description string     `json:"description,omitempty" db:"description"`
    Status      TaskStatus `json:"status" db:"status"`
    Priority    *int       `json:"priority,omitempty" db:"priority"` // Указатель для NULL
    DueDate     string     `json:"dueDate,omitempty" db:"due_date"`
    UserID      string     `json:"userId" db:"user_id"`
    WorkspaceID string     `json:"workspaceId" db:"workspace_id"`
    CreatedAt   string     `json:"createdAt" db:"created_at"`
    UpdatedAt   string     `json:"updatedAt" db:"updated_at"`
}

type CreateTaskDto struct {
    Title       string  `json:"title" binding:"required"`
    Description string  `json:"description,omitempty"`
    Status      string  `json:"status,omitempty"`
    Priority    *int    `json:"priority,omitempty"`
    DueDate     string  `json:"dueDate,omitempty"`
}

type UpdateTaskDto struct {
    Title       *string `json:"title,omitempty"`
    Description *string `json:"description,omitempty"`
    Status      *string `json:"status,omitempty"`
    Priority    *int    `json:"priority,omitempty"`
    DueDate     *string `json:"dueDate,omitempty"`
}
```

**Важные моменты:**
- Используйте `*int`, `*string` для опциональных полей (NULL в БД)
- Используйте `omitempty` в JSON тегах
- Используйте `db` теги для соответствия с колонками БД

---

### Шаг 2: Обновление существующей модели

#### 2.1. Добавление нового поля

**Алгоритм:**

1. **Создать миграцию:**
   ```sql
   -- 000021_add_assignee_to_tasks.up.sql
   ALTER TABLE tasks 
   ADD COLUMN assignee_id UUID;
   
   CREATE INDEX idx_tasks_assignee ON tasks(assignee_id);
   
   COMMENT ON COLUMN tasks.assignee_id IS 'ID пользователя, которому назначена задача';
   ```

2. **Обновить модель:**
   ```go
   type Task struct {
       // ... существующие поля
       AssigneeID *string `json:"assigneeId,omitempty" db:"assignee_id"` // Новое поле
   }
   ```

3. **Обновить DTOs:**
   ```go
   type CreateTaskDto struct {
       // ... существующие поля
       AssigneeID *string `json:"assigneeId,omitempty"` // Новое поле
   }
   ```

4. **Обновить Repository:**
   ```go
   func (r *Repository) Create(ctx context.Context, dto model.CreateTaskDto, ...) {
       query := `
           INSERT INTO tasks (..., assignee_id, ...)
           VALUES (..., $N, ...)
       `
       // Добавить assignee_id в Scan
   }
   ```

5. **Обновить Service и Handler** (если нужно)

---

#### 2.2. Удаление поля

**⚠️ ВНИМАНИЕ:** Это breaking change!

**Алгоритм:**

1. **Создать миграцию для удаления:**
   ```sql
   -- 000022_remove_priority_from_tasks.up.sql
   ALTER TABLE tasks DROP COLUMN IF EXISTS priority;
   ```

2. **Обновить модель:**
   ```go
   type Task struct {
       // ... убрать Priority
   }
   ```

3. **Обновить все места использования**

**⚠️ Для обратной совместимости:**
- Сначала пометить поле как deprecated
- Подождать версию
- Затем удалить

---

#### 2.3. Изменение типа поля

**Пример: Изменить status с VARCHAR на ENUM**

**Безопасный способ:**

1. **Создать новое поле:**
   ```sql
   -- 000023_change_task_status_type.up.sql
   -- Шаг 1: Создать новый тип
   CREATE TYPE task_status_enum AS ENUM ('todo', 'in_progress', 'done');
   
   -- Шаг 2: Добавить новое поле
   ALTER TABLE tasks 
   ADD COLUMN status_new task_status_enum;
   
   -- Шаг 3: Скопировать данные
   UPDATE tasks 
   SET status_new = CASE 
       WHEN status = 'todo' THEN 'todo'::task_status_enum
       WHEN status = 'in_progress' THEN 'in_progress'::task_status_enum
       WHEN status = 'done' THEN 'done'::task_status_enum
   END;
   
   -- Шаг 4: Удалить старое поле
   ALTER TABLE tasks DROP COLUMN status;
   
   -- Шаг 5: Переименовать новое поле
   ALTER TABLE tasks RENAME COLUMN status_new TO status;
   
   -- Шаг 6: Установить NOT NULL
   ALTER TABLE tasks ALTER COLUMN status SET NOT NULL;
   ```

2. **Обновить модель:**
   ```go
   type TaskStatus string // Остается string, но значения проверяются
   ```

---

### Шаг 3: Рефакторинг модели

#### 3.1. Разделение большой модели

**Проблема:** Модель стала слишком большой

**Решение:** Разделить на несколько таблиц

**Пример:**
```
Было: tasks (все поля в одной таблице)
Стало: 
  - tasks (основные поля)
  - task_metadata (дополнительные поля)
  - task_attachments (файлы)
```

**Миграция:**
```sql
-- 000024_split_tasks_table.up.sql
-- Создать новую таблицу для метаданных
CREATE TABLE task_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL UNIQUE,
    tags TEXT[],
    custom_fields JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Перенести данные (если есть)
INSERT INTO task_metadata (task_id, tags, custom_fields)
SELECT id, ARRAY[]::TEXT[], '{}'::JSONB
FROM tasks;

-- Foreign Key
ALTER TABLE task_metadata 
ADD CONSTRAINT fk_task_metadata_task 
FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE;
```

---

## 🔗 Управление зависимостями

### Типы зависимостей

#### 1. Foreign Key зависимости

**Пример:**
```
tasks.user_id → users.id
tasks.workspace_id → workspaces.id
```

**Правила:**
- Нельзя удалить `users` если есть `tasks` с этим `user_id`
- Можно использовать `ON DELETE CASCADE` для автоматического удаления
- Можно использовать `ON DELETE SET NULL` для обнуления

**Создание:**
```sql
ALTER TABLE tasks 
ADD CONSTRAINT fk_tasks_user 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
```

**Удаление:**
```sql
ALTER TABLE tasks DROP CONSTRAINT fk_tasks_user;
```

---

#### 2. Зависимости в коде (Go)

**Проблема:** Repository зависит от других Repository

**Пример:**
```go
// ❌ ПЛОХО: Прямая зависимость
type TaskRepository struct {
    db *sql.DB
    userRepo *user.Repository // Прямая зависимость
}

// ✅ ХОРОШО: Зависимость через интерфейс
type UserRepository interface {
    GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type TaskRepository struct {
    db       *sql.DB
    userRepo UserRepository // Интерфейс
}
```

---

#### 3. Зависимости модулей

**Проблема:** Модуль A зависит от модуля B

**Решение:** Использовать события (Event-Driven)

**Пример:**
```go
// Модуль Tasks не знает о модуле Habits
// Но когда задача завершена - нужно обновить статистику

// Создать событие
type TaskCompletedEvent struct {
    TaskID    uuid.UUID
    UserID    uuid.UUID
    WorkspaceID uuid.UUID
    CompletedAt time.Time
}

// В Task Service
func (s *Service) Complete(ctx context.Context, taskID string) error {
    // ... завершить задачу
    
    // Отправить событие
    s.eventBus.Publish("task.completed", TaskCompletedEvent{
        TaskID: taskID,
        // ...
    })
    
    return nil
}

// В Habits Service (подписчик)
func (s *Service) OnTaskCompleted(event TaskCompletedEvent) {
    // Обновить статистику привычек
}
```

---

### Удаление зависимостей

#### Шаг 1: Найти все использования

```bash
# Найти все места, где используется модель
grep -r "model.Task" backend/internal/

# Найти все Foreign Keys
grep -r "fk_tasks" backend/migrations/
```

#### Шаг 2: Удалить Foreign Keys

```sql
-- 000025_remove_task_dependencies.up.sql
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_user;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_workspace;
```

#### Шаг 3: Удалить код

```go
// Удалить из repository
// Удалить из service
// Удалить из handler
```

#### Шаг 4: Удалить таблицу

```sql
-- 000026_drop_tasks_table.up.sql
DROP TABLE IF EXISTS tasks;
```

---

## 🔀 Таблицы связей (Junction Tables)

### Когда нужна Junction Table?

**Junction Table** нужна для связи **многие-ко-многим (N:M)**

**Примеры:**
- Пользователи ↔ Workspaces (один пользователь в нескольких workspace, один workspace содержит нескольких пользователей)
- Задачи ↔ Теги (одна задача может иметь несколько тегов, один тег может быть у нескольких задач)
- Продукты ↔ Категории (один продукт в нескольких категориях, одна категория содержит несколько продуктов)

---

### Структура Junction Table

**Базовый шаблон:**
```sql
CREATE TABLE junction_table_name (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity1_id UUID NOT NULL,
    entity2_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Предотвращаем дубликаты
    UNIQUE(entity1_id, entity2_id)
);

-- Foreign Keys
ALTER TABLE junction_table_name 
ADD CONSTRAINT fk_junction_entity1 
FOREIGN KEY (entity1_id) REFERENCES entity1_table(id) ON DELETE CASCADE;

ALTER TABLE junction_table_name 
ADD CONSTRAINT fk_junction_entity2 
FOREIGN KEY (entity2_id) REFERENCES entity2_table(id) ON DELETE CASCADE;

-- Индексы для быстрого поиска в обе стороны
CREATE INDEX idx_junction_entity1 ON junction_table_name(entity1_id);
CREATE INDEX idx_junction_entity2 ON junction_table_name(entity2_id);
```

---

### Пример: Tasks ↔ Tags

#### 1. Создать таблицу тегов

```sql
-- 000030_create_tags.up.sql
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    color VARCHAR(50) DEFAULT '#3B82F6',
    workspace_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tags_workspace ON tags(workspace_id);
```

#### 2. Создать Junction Table

```sql
-- 000031_create_task_tags.up.sql
CREATE TABLE task_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    UNIQUE(task_id, tag_id)
);

ALTER TABLE task_tags 
ADD CONSTRAINT fk_task_tags_task 
FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE;

ALTER TABLE task_tags 
ADD CONSTRAINT fk_task_tags_tag 
FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE;

CREATE INDEX idx_task_tags_task ON task_tags(task_id);
CREATE INDEX idx_task_tags_tag ON task_tags(tag_id);
```

#### 3. Использование в коде

```go
// Создать задачу с тегами
func (r *Repository) CreateWithTags(ctx context.Context, dto model.CreateTaskDto, userID, workspaceID uuid.UUID, tagIDs []uuid.UUID) (*model.Task, error) {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()
    
    // 1. Создать задачу
    task, err := r.createTask(ctx, tx, dto, userID, workspaceID)
    if err != nil {
        return nil, err
    }
    
    // 2. Связать теги
    for _, tagID := range tagIDs {
        _, err = tx.ExecContext(ctx,
            "INSERT INTO task_tags (id, task_id, tag_id, created_at) VALUES ($1, $2, $3, $4)",
            uuid.New(), task.ID, tagID, time.Now(),
        )
        if err != nil {
            return nil, err
        }
    }
    
    if err := tx.Commit(); err != nil {
        return nil, err
    }
    
    return task, nil
}

// Получить задачу с тегами
func (r *Repository) GetWithTags(ctx context.Context, taskID uuid.UUID) (*model.Task, error) {
    // 1. Получить задачу
    task, err := r.Get(ctx, taskID)
    if err != nil {
        return nil, err
    }
    
    // 2. Получить теги
    query := `
        SELECT t.id, t.name, t.color, t.created_at
        FROM tags t
        INNER JOIN task_tags tt ON t.id = tt.tag_id
        WHERE tt.task_id = $1
        ORDER BY t.name
    `
    
    rows, err := r.db.QueryContext(ctx, query, taskID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var tags []model.Tag
    for rows.Next() {
        var tag model.Tag
        var createdAt time.Time
        rows.Scan(&tag.ID, &tag.Name, &tag.Color, &createdAt)
        tag.CreatedAt = createdAt.Format(time.RFC3339)
        tags = append(tags, tag)
    }
    
    task.Tags = tags
    return task, nil
}
```

---

### Дополнительные поля в Junction Table

**Иногда нужны дополнительные данные о связи:**

```sql
CREATE TABLE task_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'assignee', -- 'assignee', 'reviewer', 'watcher'
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    assigned_by UUID, -- Кто назначил
    
    UNIQUE(task_id, user_id)
);
```

---

## 🔄 Обратная совместимость

### Что это?

**Обратная совместимость** - способность новой версии работать со старыми данными и API.

---

### Стратегии

#### 1. Добавление нового поля

**✅ Безопасно (обратно совместимо):**
```sql
-- Добавить опциональное поле
ALTER TABLE tasks 
ADD COLUMN new_field VARCHAR(100); -- NULL по умолчанию
```

**В коде:**
```go
type Task struct {
    // ... существующие поля
    NewField *string `json:"newField,omitempty" db:"new_field"` // Опциональное
}
```

**Старый код продолжит работать**, потому что:
- Поле опциональное (может быть NULL)
- JSON `omitempty` не отправляет поле, если оно пустое
- Старые клиенты просто игнорируют новое поле

---

#### 2. Удаление поля

**❌ НЕ безопасно (breaking change):**

```sql
-- Удалить поле
ALTER TABLE tasks DROP COLUMN old_field;
```

**Решение: Deprecation процесс**

**Шаг 1: Пометить как deprecated**
```go
type Task struct {
    // ... другие поля
    OldField string `json:"oldField,omitempty" db:"old_field"` // DEPRECATED: будет удалено в v2.0
}
```

**Шаг 2: Подождать версию**
- Выпустить версию с deprecated полем
- Дать время клиентам обновиться

**Шаг 3: Удалить**
```sql
-- В новой версии
ALTER TABLE tasks DROP COLUMN old_field;
```

---

#### 3. Изменение типа поля

**❌ НЕ безопасно напрямую**

**✅ Безопасный способ:**

**Вариант A: Добавить новое поле, удалить старое позже**
```sql
-- Шаг 1: Добавить новое поле
ALTER TABLE tasks 
ADD COLUMN status_new VARCHAR(50);

-- Шаг 2: Копировать данные
UPDATE tasks 
SET status_new = CAST(status AS VARCHAR(50));

-- Шаг 3: В следующей версии удалить старое
-- ALTER TABLE tasks DROP COLUMN status;
-- ALTER TABLE tasks RENAME COLUMN status_new TO status;
```

**Вариант B: Поддержка обоих форматов в коде**
```go
type Task struct {
    Status    string `json:"status" db:"status"`        // Старый формат
    StatusNew string `json:"statusNew" db:"status_new"` // Новый формат
}

func (t *Task) GetStatus() string {
    if t.StatusNew != "" {
        return t.StatusNew
    }
    return t.Status // Fallback на старый
}
```

---

#### 4. Изменение API

**❌ Breaking change:**
```go
// Было
GET /api/tasks?user_id=123

// Стало
GET /api/tasks?userId=123
```

**✅ Обратно совместимо:**
```go
// Поддержать оба варианта
func (h *Handler) List(c *gin.Context) {
    userID := c.Query("user_id")
    if userID == "" {
        userID = c.Query("userId") // Новый формат
    }
    // ...
}
```

---

### Версионирование API

**Решение:** Версионировать API

```
/api/v1/tasks  - старая версия (deprecated)
/api/v2/tasks  - новая версия
```

**В коде:**
```go
v1 := router.Group("/api/v1")
{
    v1.GET("/tasks", v1Handler.List)
}

v2 := router.Group("/api/v2")
{
    v2.GET("/tasks", v2Handler.List)
}
```

---

## 💾 Транзакции

### Что такое транзакция?

**Транзакция** - атомарная операция: либо выполняется все, либо ничего.

**ACID свойства:**
- **Atomicity** (Атомарность) - все или ничего
- **Consistency** (Согласованность) - данные всегда в валидном состоянии
- **Isolation** (Изоляция) - транзакции не мешают друг другу
- **Durability** (Долговечность) - изменения сохраняются

---

### Когда использовать транзакции?

#### ✅ Использовать транзакции когда:

1. **Несколько связанных операций**
   ```go
   // Создать задачу и назначить теги
   tx, _ := db.BeginTx(ctx, nil)
   defer tx.Rollback()
   
   // 1. Создать задачу
   task, _ := createTask(tx, ...)
   
   // 2. Назначить теги
   for _, tagID := range tagIDs {
       assignTag(tx, task.ID, tagID)
   }
   
   tx.Commit() // Все или ничего
   ```

2. **Перемещение данных между таблицами**
   ```go
   // Перенести задачу из одного workspace в другой
   tx, _ := db.BeginTx(ctx, nil)
   defer tx.Rollback()
   
   // 1. Обновить workspace_id
   UPDATE tasks SET workspace_id = $1 WHERE id = $2
   
   // 2. Обновить связанные данные
   UPDATE task_tags SET ... WHERE task_id = $2
   
   tx.Commit()
   ```

3. **Удаление с каскадом (когда CASCADE недостаточно)**
   ```go
   // Удалить workspace и все связанные данные
   tx, _ := db.BeginTx(ctx, nil)
   defer tx.Rollback()
   
   // 1. Удалить все задачи
   DELETE FROM tasks WHERE workspace_id = $1
   
   // 2. Удалить все привычки
   DELETE FROM habits WHERE workspace_id = $1
   
   // 3. Удалить workspace
   DELETE FROM workspaces WHERE id = $1
   
   tx.Commit()
   ```

4. **Обновление счетчиков/статистики**
   ```go
   // Обновить счетчик выполненных задач
   tx, _ := db.BeginTx(ctx, nil)
   defer tx.Rollback()
   
   // 1. Завершить задачу
   UPDATE tasks SET status = 'done' WHERE id = $1
   
   // 2. Увеличить счетчик
   UPDATE user_stats SET completed_tasks = completed_tasks + 1 WHERE user_id = $2
   
   tx.Commit()
   ```

---

#### ❌ НЕ использовать транзакции когда:

1. **Одна простая операция**
   ```go
   // ❌ Избыточно
   tx, _ := db.BeginTx(ctx, nil)
   UPDATE tasks SET title = $1 WHERE id = $2
   tx.Commit()
   
   // ✅ Достаточно
   db.ExecContext(ctx, "UPDATE tasks SET title = $1 WHERE id = $2", title, id)
   ```

2. **Долгие операции**
   ```go
   // ❌ Транзакция блокирует другие операции
   tx, _ := db.BeginTx(ctx, nil)
   // Долгая операция (например, отправка email)
   sendEmail(...)
   tx.Commit()
   
   // ✅ Разделить на части
   // 1. Обновить БД (быстро)
   db.ExecContext(ctx, "UPDATE ...")
   // 2. Отправить email (асинхронно, вне транзакции)
   go sendEmail(...)
   ```

---

### Примеры из вашего кода

#### Пример 1: Удаление привычки (habits/repository.go)

```go
func (r *Repository) Delete(ctx context.Context, id, userID uuid.UUID) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback() // Откатить при ошибке
    
    // 1. Удалить связанные completions
    _, err = tx.ExecContext(ctx,
        "DELETE FROM habit_completions WHERE habit_id = $1 AND user_id = $2",
        id, userID,
    )
    if err != nil {
        return fmt.Errorf("failed to delete habit completions: %w", err)
    }
    
    // 2. Удалить привычку
    result, err := tx.ExecContext(ctx,
        "DELETE FROM habits WHERE id = $1 AND user_id = $2",
        id, userID,
    )
    if err != nil {
        return fmt.Errorf("failed to delete habit: %w", err)
    }
    
    // Проверить, что что-то удалено
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return sql.ErrNoRows
    }
    
    // Коммитить транзакцию
    return tx.Commit()
}
```

**Почему транзакция здесь:**
- Две операции должны выполниться вместе
- Если удаление completions успешно, а удаление habit - нет, нужно откатить все

---

#### Пример 2: Toggle выполнения (habits/repository.go)

```go
func (r *Repository) Toggle(ctx context.Context, habitID, userID uuid.UUID, date time.Time) (bool, *model.HabitCompletion, error) {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return false, nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()
    
    // Проверить существующее completion
    var existing model.HabitCompletion
    err = tx.QueryRowContext(ctx,
        "SELECT * FROM habit_completions WHERE habit_id = $1 AND user_id = $2 AND date = $3",
        habitID, userID, date,
    ).Scan(...)
    
    if err == nil {
        // Существует - удалить
        _, err = tx.ExecContext(ctx,
            "DELETE FROM habit_completions WHERE id = $1",
            existing.ID,
        )
        if err != nil {
            return false, nil, err
        }
        
        if err := tx.Commit(); err != nil {
            return false, nil, err
        }
        return false, &existing, nil
    }
    
    // Не существует - создать
    completion, err := r.Complete(ctx, habitID, userID, date, "", 0, nil)
    if err != nil {
        return false, nil, err
    }
    
    if err := tx.Commit(); err != nil {
        return false, nil, err
    }
    return true, completion, nil
}
```

**Почему транзакция здесь:**
- Предотвращает race condition (два запроса одновременно проверяют и создают)
- Гарантирует атомарность: либо удаление, либо создание

---

### Уровни изоляции

**PostgreSQL поддерживает уровни изоляции:**

```go
// Read Uncommitted (не поддерживается в PostgreSQL)
// Read Committed (по умолчанию)
tx, _ := db.BeginTx(ctx, &sql.TxOptions{
    Isolation: sql.LevelDefault, // Read Committed
})

// Repeatable Read
tx, _ := db.BeginTx(ctx, &sql.TxOptions{
    Isolation: sql.LevelRepeatableRead,
})

// Serializable
tx, _ := db.BeginTx(ctx, &sql.TxOptions{
    Isolation: sql.LevelSerializable,
})
```

**В большинстве случаев достаточно уровня по умолчанию (Read Committed).**

---

### Обработка ошибок в транзакциях

**Правильный паттерн:**
```go
func (r *Repository) ComplexOperation(ctx context.Context, ...) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback() // Всегда откатывать при выходе
    
    // Операция 1
    if err := operation1(tx, ...); err != nil {
        return fmt.Errorf("operation1 failed: %w", err)
    }
    
    // Операция 2
    if err := operation2(tx, ...); err != nil {
        return fmt.Errorf("operation2 failed: %w", err)
    }
    
    // Все успешно - коммитить
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    return nil
}
```

**Важно:**
- `defer tx.Rollback()` вызывается даже при успешном Commit (но Commit уже выполнен, так что Rollback ничего не делает)
- Если произошла ошибка до Commit, Rollback откатит все изменения

---

## 👑 Супер-пользователь и модули

### Концепция

**Супер-пользователь (Super Admin)** - пользователь с правами на:
- Создание новых модулей
- Включение/отключение модулей для workspace
- Управление системой

---

### Реализация

#### Шаг 1: Расширить модель User

```sql
-- 000040_add_super_admin_to_users.up.sql
ALTER TABLE users 
ADD COLUMN is_super_admin BOOLEAN DEFAULT FALSE;

CREATE INDEX idx_users_super_admin ON users(is_super_admin) WHERE is_super_admin = TRUE;

COMMENT ON COLUMN users.is_super_admin IS 'Супер-администратор системы';
```

```go
// internal/model/user.go
type User struct {
    // ... существующие поля
    IsSuperAdmin bool `json:"isSuperAdmin,omitempty" db:"is_super_admin"`
}
```

---

#### Шаг 2: Создать таблицу модулей

```sql
-- 000041_create_modules.up.sql
CREATE TABLE modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE, -- 'habits', 'journal', 'tasks'
    title VARCHAR(255) NOT NULL, -- 'Привычки', 'Журнал', 'Задачи'
    description TEXT,
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    enabled BOOLEAN DEFAULT TRUE, -- Включен ли модуль в системе
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_modules_name ON modules(name);
CREATE INDEX idx_modules_enabled ON modules(enabled);

COMMENT ON TABLE modules IS 'Модули ERP системы';
COMMENT ON COLUMN modules.name IS 'Уникальное имя модуля (используется в коде)';
COMMENT ON COLUMN modules.enabled IS 'Включен ли модуль в системе (может быть отключен супер-админом)';
```

---

#### Шаг 3: Создать таблицу workspace_modules

```sql
-- 000042_create_workspace_modules.up.sql
CREATE TABLE workspace_modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    module_id UUID NOT NULL,
    enabled BOOLEAN DEFAULT TRUE, -- Включен ли модуль для этого workspace
    config JSONB, -- Конфигурация модуля для workspace
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    UNIQUE(workspace_id, module_id)
);

ALTER TABLE workspace_modules 
ADD CONSTRAINT fk_workspace_modules_workspace 
FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;

ALTER TABLE workspace_modules 
ADD CONSTRAINT fk_workspace_modules_module 
FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE CASCADE;

CREATE INDEX idx_workspace_modules_workspace ON workspace_modules(workspace_id);
CREATE INDEX idx_workspace_modules_module ON workspace_modules(module_id);
CREATE INDEX idx_workspace_modules_enabled ON workspace_modules(workspace_id, enabled) WHERE enabled = TRUE;

COMMENT ON TABLE workspace_modules IS 'Связь workspace с модулями';
COMMENT ON COLUMN workspace_modules.config IS 'JSON конфигурация модуля для workspace';
```

---

#### Шаг 4: Seed данные (создать модули)

```sql
-- 000043_seed_modules.up.sql
INSERT INTO modules (id, name, title, description, version, enabled) VALUES
    (gen_random_uuid(), 'habits', 'Привычки', 'Модуль отслеживания привычек', '1.0.0', TRUE),
    (gen_random_uuid(), 'journal', 'Журнал', 'Модуль ведения дневника', '1.0.0', TRUE),
    (gen_random_uuid(), 'tasks', 'Задачи', 'Модуль управления задачами', '1.0.0', FALSE) -- Выключен по умолчанию
ON CONFLICT (name) DO NOTHING;
```

---

#### Шаг 5: Модели

```go
// internal/model/module.go
package model

type Module struct {
    ID          string `json:"id" db:"id"`
    Name        string `json:"name" db:"name"`
    Title       string `json:"title" db:"title"`
    Description string `json:"description,omitempty" db:"description"`
    Version     string `json:"version" db:"version"`
    Enabled     bool   `json:"enabled" db:"enabled"`
    CreatedAt   string `json:"createdAt" db:"created_at"`
    UpdatedAt   string `json:"updatedAt" db:"updated_at"`
}

type WorkspaceModule struct {
    ID          string                 `json:"id" db:"id"`
    WorkspaceID string                 `json:"workspaceId" db:"workspace_id"`
    ModuleID    string                 `json:"moduleId" db:"module_id"`
    Enabled     bool                   `json:"enabled" db:"enabled"`
    Config      map[string]interface{} `json:"config,omitempty" db:"config"`
    CreatedAt   string                 `json:"createdAt" db:"created_at"`
    UpdatedAt   string                 `json:"updatedAt" db:"updated_at"`
}
```

---

#### Шаг 6: Middleware для проверки модулей

```go
// internal/middleware/module.go
package middleware

import (
    "backend/internal/model"
    "github.com/gin-gonic/gin"
)

func RequireModule(moduleName string) gin.HandlerFunc {
    return func(c *gin.Context) {
        workspaceID, _ := GetWorkspaceIDFromGin(c)
        if workspaceID == "" {
            c.JSON(400, gin.H{"error": "Workspace required"})
            c.Abort()
            return
        }
        
        // Проверить, включен ли модуль для workspace
        // Это нужно реализовать в service
        // moduleService := ...
        // enabled, err := moduleService.IsModuleEnabled(workspaceID, moduleName)
        
        c.Next()
    }
}
```

---

#### Шаг 7: Использование в роутах

```go
// internal/router/router.go
func (r *Router) RegisterRoutes(container *di.Container) {
    api := r.engine.Group("/api")
    api.Use(middleware.Auth())
    
    // Habits модуль
    habits := api.Group("/habits")
    habits.Use(middleware.RequireWorkspace())
    habits.Use(middleware.RequireModule("habits")) // Проверка модуля
    {
        container.HabitsHandler.RegisterRoutes(habits)
    }
    
    // Journal модуль
    journal := api.Group("/journal")
    journal.Use(middleware.RequireWorkspace())
    journal.Use(middleware.RequireModule("journal"))
    {
        container.JournalHandler.RegisterRoutes(journal)
    }
}
```

---

### API для супер-админа

```go
// internal/handler/admin/handler.go
package admin

type Handler struct {
    moduleService *module.Service
    responder     *response.Responder
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
    admin := r.Group("/admin")
    admin.Use(middleware.RequireSuperAdmin()) // Проверка супер-админа
    {
        admin.GET("/modules", h.ListModules)
        admin.POST("/modules", h.CreateModule)
        admin.PUT("/modules/:id", h.UpdateModule)
        admin.DELETE("/modules/:id", h.DeleteModule)
        admin.POST("/modules/:id/enable", h.EnableModule)
        admin.POST("/modules/:id/disable", h.DisableModule)
    }
}

func (h *Handler) ListModules(c *gin.Context) {
    modules, err := h.moduleService.List(c.Request.Context())
    if err != nil {
        h.responder.InternalServerError(c, "Failed to list modules")
        return
    }
    h.responder.SuccessWithData(c, gin.H{"modules": modules})
}

func (h *Handler) CreateModule(c *gin.Context) {
    var req model.CreateModuleDto
    if err := c.ShouldBindJSON(&req); err != nil {
        h.responder.BadRequest(c, "Invalid request")
        return
    }
    
    module, err := h.moduleService.Create(c.Request.Context(), req)
    if err != nil {
        h.responder.InternalServerError(c, "Failed to create module")
        return
    }
    
    h.responder.Created(c, "Module created", module)
}
```

---

### Структура модуля по умолчанию

**Каждый модуль должен иметь:**

```
module_name/
├── migrations/
│   ├── 000XXX_create_module_table.up.sql
│   └── 000XXX_create_module_table.down.sql
├── model/
│   └── module_name.go
├── repository/
│   └── repository.go
├── service/
│   └── service.go
└── handler/
    ├── handler.go
    └── routes.go
```

**Пример для модуля Tasks:**

```
tasks/
├── migrations/
│   ├── 000020_create_tasks.up.sql
│   ├── 000020_create_tasks.down.sql
│   ├── 000021_create_task_tags.up.sql
│   └── 000021_create_task_tags.down.sql
├── model/
│   └── task.go
├── repository/
│   └── repository.go
├── service/
│   └── service.go
└── handler/
    ├── handler.go
    └── routes.go
```

---

## 📊 SQL Мастер-класс

### Основы SQL для ERP

#### 1. SELECT с фильтрами

**Базовый запрос:**
```sql
SELECT id, title, status
FROM tasks
WHERE workspace_id = $1
ORDER BY created_at DESC;
```

**С фильтрами:**
```sql
SELECT id, title, status, priority
FROM tasks
WHERE workspace_id = $1
  AND user_id = $2
  AND status = 'todo'
  AND priority >= 3
  AND due_date >= CURRENT_DATE
ORDER BY priority DESC, created_at DESC;
```

**С поиском:**
```sql
SELECT id, title
FROM tasks
WHERE workspace_id = $1
  AND (title ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')
ORDER BY created_at DESC;
```

**С пагинацией:**
```sql
SELECT id, title, status
FROM tasks
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
-- LIMIT = количество записей на странице
-- OFFSET = пропустить N записей (для пагинации)
```

---

#### 2. JOIN для связанных данных

**INNER JOIN (только совпадающие):**
```sql
SELECT 
    t.id,
    t.title,
    u.name AS assignee_name
FROM tasks t
INNER JOIN users u ON t.assignee_id = u.id
WHERE t.workspace_id = $1;
```

**LEFT JOIN (все задачи, даже без assignee):**
```sql
SELECT 
    t.id,
    t.title,
    u.name AS assignee_name
FROM tasks t
LEFT JOIN users u ON t.assignee_id = u.id
WHERE t.workspace_id = $1;
```

**Множественные JOIN:**
```sql
SELECT 
    t.id,
    t.title,
    u.name AS assignee_name,
    w.name AS workspace_name
FROM tasks t
LEFT JOIN users u ON t.assignee_id = u.id
INNER JOIN workspaces w ON t.workspace_id = w.id
WHERE t.workspace_id = $1;
```

**JOIN с Junction Table:**
```sql
SELECT 
    t.id,
    t.title,
    tag.name AS tag_name,
    tag.color AS tag_color
FROM tasks t
INNER JOIN task_tags tt ON t.id = tt.task_id
INNER JOIN tags tag ON tt.tag_id = tag.id
WHERE t.workspace_id = $1
ORDER BY t.created_at DESC, tag.name;
```

---

#### 3. Агрегатные функции

**COUNT:**
```sql
SELECT COUNT(*) AS total_tasks
FROM tasks
WHERE workspace_id = $1;
```

**COUNT с GROUP BY:**
```sql
SELECT status, COUNT(*) AS count
FROM tasks
WHERE workspace_id = $1
GROUP BY status;
-- Результат:
-- status      | count
-- ------------|------
-- todo        | 10
-- in_progress | 5
-- done        | 20
```

**SUM, AVG, MIN, MAX:**
```sql
SELECT 
    COUNT(*) AS total,
    AVG(priority) AS avg_priority,
    MIN(created_at) AS first_task,
    MAX(created_at) AS last_task
FROM tasks
WHERE workspace_id = $1;
```

---

#### 4. Подзапросы (Subqueries)

**Подзапрос в WHERE:**
```sql
SELECT id, title
FROM tasks
WHERE workspace_id = $1
  AND assignee_id IN (
      SELECT id FROM users 
      WHERE workspace_id = $1 AND role = 'member'
  );
```

**Подзапрос в SELECT:**
```sql
SELECT 
    t.id,
    t.title,
    (SELECT COUNT(*) FROM task_comments WHERE task_id = t.id) AS comments_count
FROM tasks t
WHERE t.workspace_id = $1;
```

**EXISTS:**
```sql
SELECT id, title
FROM tasks t
WHERE workspace_id = $1
  AND EXISTS (
      SELECT 1 FROM task_tags tt 
      WHERE tt.task_id = t.id AND tt.tag_id = $2
  );
```

---

#### 5. Оконные функции (Window Functions)

**ROW_NUMBER:**
```sql
SELECT 
    id,
    title,
    ROW_NUMBER() OVER (PARTITION BY workspace_id ORDER BY created_at DESC) AS row_num
FROM tasks
WHERE workspace_id = $1;
```

**RANK:**
```sql
SELECT 
    id,
    title,
    priority,
    RANK() OVER (PARTITION BY workspace_id ORDER BY priority DESC) AS priority_rank
FROM tasks
WHERE workspace_id = $1;
```

**SUM OVER:**
```sql
SELECT 
    date,
    COUNT(*) AS tasks_count,
    SUM(COUNT(*)) OVER (ORDER BY date) AS cumulative_count
FROM tasks
WHERE workspace_id = $1
GROUP BY date
ORDER BY date;
```

---

#### 6. CTE (Common Table Expressions)

**Простой CTE:**
```sql
WITH active_tasks AS (
    SELECT id, title, status
    FROM tasks
    WHERE workspace_id = $1 AND status != 'done'
)
SELECT * FROM active_tasks
ORDER BY created_at DESC;
```

**Рекурсивный CTE (для иерархий):**
```sql
WITH RECURSIVE task_tree AS (
    -- Базовый случай: корневые задачи
    SELECT id, title, parent_id, 1 AS level
    FROM tasks
    WHERE workspace_id = $1 AND parent_id IS NULL
    
    UNION ALL
    
    -- Рекурсивный случай: дочерние задачи
    SELECT t.id, t.title, t.parent_id, tt.level + 1
    FROM tasks t
    INNER JOIN task_tree tt ON t.parent_id = tt.id
    WHERE t.workspace_id = $1
)
SELECT * FROM task_tree
ORDER BY level, created_at;
```

---

#### 7. Оптимизация запросов

**Использование индексов:**
```sql
-- Создать индекс для часто используемых фильтров
CREATE INDEX idx_tasks_workspace_status ON tasks(workspace_id, status);

-- Составной индекс для сложных запросов
CREATE INDEX idx_tasks_workspace_user_status ON tasks(workspace_id, user_id, status);
```

**EXPLAIN для анализа:**
```sql
EXPLAIN ANALYZE
SELECT id, title
FROM tasks
WHERE workspace_id = $1 AND status = 'todo'
ORDER BY created_at DESC;
```

**Покрывающий индекс (Covering Index):**
```sql
-- Индекс содержит все нужные поля - не нужно обращаться к таблице
CREATE INDEX idx_tasks_covering ON tasks(workspace_id, status) 
INCLUDE (id, title, created_at);
```

---

## 🎯 Практические примеры

### Пример 1: Полный цикл создания модуля Tasks

#### Шаг 1: Планирование

**Определяем:**
- Таблица `tasks` с полями: id, title, description, status, priority, due_date, user_id, workspace_id
- Связь с тегами через junction table `task_tags`
- Связь с пользователями через `assignee_id`

#### Шаг 2: Миграции

**000020_create_tasks.up.sql:**
```sql
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'todo',
    priority INTEGER CHECK (priority >= 1 AND priority <= 5),
    due_date DATE,
    assignee_id UUID,
    user_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CHECK (status IN ('todo', 'in_progress', 'done'))
);

CREATE INDEX idx_tasks_workspace ON tasks(workspace_id);
CREATE INDEX idx_tasks_user ON tasks(user_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_assignee ON tasks(assignee_id);
```

**000020_create_tasks.down.sql:**
```sql
DROP TABLE IF EXISTS tasks;
```

**000021_create_task_tags.up.sql:**
```sql
CREATE TABLE task_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    UNIQUE(task_id, tag_id)
);

ALTER TABLE task_tags 
ADD CONSTRAINT fk_task_tags_task 
FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE;

ALTER TABLE task_tags 
ADD CONSTRAINT fk_task_tags_tag 
FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE;

CREATE INDEX idx_task_tags_task ON task_tags(task_id);
CREATE INDEX idx_task_tags_tag ON task_tags(tag_id);
```

**constraints/03_tasks_foreign_keys.up.sql:**
```sql
ALTER TABLE tasks 
ADD CONSTRAINT fk_tasks_user 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE tasks 
ADD CONSTRAINT fk_tasks_workspace 
FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;

ALTER TABLE tasks 
ADD CONSTRAINT fk_tasks_assignee 
FOREIGN KEY (assignee_id) REFERENCES users(id) ON DELETE SET NULL;
```

#### Шаг 3: Модель

```go
// internal/model/task.go
package model

type TaskStatus string

const (
    TaskStatusTodo       TaskStatus = "todo"
    TaskStatusInProgress TaskStatus = "in_progress"
    TaskStatusDone      TaskStatus = "done"
)

type Task struct {
    ID          string     `json:"id" db:"id"`
    Title       string     `json:"title" db:"title"`
    Description string     `json:"description,omitempty" db:"description"`
    Status      TaskStatus `json:"status" db:"status"`
    Priority    *int       `json:"priority,omitempty" db:"priority"`
    DueDate     string     `json:"dueDate,omitempty" db:"due_date"`
    AssigneeID  *string    `json:"assigneeId,omitempty" db:"assignee_id"`
    UserID      string     `json:"userId" db:"user_id"`
    WorkspaceID string     `json:"workspaceId" db:"workspace_id"`
    Tags        []Tag      `json:"tags,omitempty"`
    CreatedAt   string     `json:"createdAt" db:"created_at"`
    UpdatedAt   string     `json:"updatedAt" db:"updated_at"`
}

type CreateTaskDto struct {
    Title       string   `json:"title" binding:"required"`
    Description string   `json:"description,omitempty"`
    Status      string   `json:"status,omitempty"`
    Priority    *int     `json:"priority,omitempty"`
    DueDate     string   `json:"dueDate,omitempty"`
    AssigneeID  *string  `json:"assigneeId,omitempty"`
    TagIDs      []string `json:"tagIds,omitempty"`
}

type UpdateTaskDto struct {
    Title       *string `json:"title,omitempty"`
    Description *string `json:"description,omitempty"`
    Status      *string `json:"status,omitempty"`
    Priority    *int    `json:"priority,omitempty"`
    DueDate     *string `json:"dueDate,omitempty"`
    AssigneeID  *string `json:"assigneeId,omitempty"`
}
```

#### Шаг 4: Repository

```go
// internal/repository/tasks/repository.go
package tasks

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    "backend/internal/model"
    "github.com/google/uuid"
)

type Repository struct {
    db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
    return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, dto model.CreateTaskDto, userID, workspaceID uuid.UUID) (*model.Task, error) {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()
    
    taskID := uuid.New()
    now := time.Now()
    
    // Определить статус
    status := dto.Status
    if status == "" {
        status = string(model.TaskStatusTodo)
    }
    
    // Создать задачу
    query := `
        INSERT INTO tasks (
            id, title, description, status, priority, due_date, 
            assignee_id, user_id, workspace_id, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        RETURNING id, title, description, status, priority, due_date, 
                  assignee_id, user_id, workspace_id, created_at, updated_at
    `
    
    var task model.Task
    var createdAt, updatedAt time.Time
    var dueDatePtr sql.NullTime
    var priorityPtr sql.NullInt64
    var assigneeIDPtr sql.NullString
    
    var dueDateValue interface{}
    if dto.DueDate != "" {
        dueDate, err := time.Parse("2006-01-02", dto.DueDate)
        if err == nil {
            dueDateValue = dueDate
        }
    }
    
    err = tx.QueryRowContext(ctx, query,
        taskID, dto.Title, dto.Description, status, dto.Priority, dueDateValue,
        dto.AssigneeID, userID, workspaceID, now, now,
    ).Scan(
        &task.ID, &task.Title, &task.Description, &task.Status,
        &priorityPtr, &dueDatePtr, &assigneeIDPtr,
        &task.UserID, &task.WorkspaceID, &createdAt, &updatedAt,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create task: %w", err)
    }
    
    // Обработать NULL значения
    if priorityPtr.Valid {
        p := int(priorityPtr.Int64)
        task.Priority = &p
    }
    if dueDatePtr.Valid {
        task.DueDate = dueDatePtr.Time.Format("2006-01-02")
    }
    if assigneeIDPtr.Valid {
        task.AssigneeID = &assigneeIDPtr.String
    }
    
    // Связать теги
    if len(dto.TagIDs) > 0 {
        for _, tagIDStr := range dto.TagIDs {
            tagID, err := uuid.Parse(tagIDStr)
            if err != nil {
                continue
            }
            
            _, err = tx.ExecContext(ctx,
                "INSERT INTO task_tags (id, task_id, tag_id, created_at) VALUES ($1, $2, $3, $4)",
                uuid.New(), taskID, tagID, now,
            )
            if err != nil {
                return nil, fmt.Errorf("failed to link tag: %w", err)
            }
        }
    }
    
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    task.CreatedAt = createdAt.Format(time.RFC3339)
    task.UpdatedAt = updatedAt.Format(time.RFC3339)
    
    return &task, nil
}

func (r *Repository) List(ctx context.Context, userID, workspaceID uuid.UUID, filters map[string]interface{}) ([]model.Task, error) {
    query := `
        SELECT id, title, description, status, priority, due_date, 
               assignee_id, user_id, workspace_id, created_at, updated_at
        FROM tasks
        WHERE user_id = $1 AND workspace_id = $2
    `
    
    args := []interface{}{userID, workspaceID}
    argIndex := 3
    
    // Динамические фильтры
    if status, ok := filters["status"].(string); ok && status != "" {
        query += fmt.Sprintf(" AND status = $%d", argIndex)
        args = append(args, status)
        argIndex++
    }
    
    if assigneeID, ok := filters["assignee_id"].(string); ok && assigneeID != "" {
        query += fmt.Sprintf(" AND assignee_id = $%d", argIndex)
        args = append(args, assigneeID)
        argIndex++
    }
    
    query += " ORDER BY created_at DESC"
    
    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query tasks: %w", err)
    }
    defer rows.Close()
    
    var tasks []model.Task
    for rows.Next() {
        var task model.Task
        var createdAt, updatedAt time.Time
        var dueDatePtr sql.NullTime
        var priorityPtr sql.NullInt64
        var assigneeIDPtr sql.NullString
        
        err := rows.Scan(
            &task.ID, &task.Title, &task.Description, &task.Status,
            &priorityPtr, &dueDatePtr, &assigneeIDPtr,
            &task.UserID, &task.WorkspaceID, &createdAt, &updatedAt,
        )
        if err != nil {
            continue
        }
        
        if priorityPtr.Valid {
            p := int(priorityPtr.Int64)
            task.Priority = &p
        }
        if dueDatePtr.Valid {
            task.DueDate = dueDatePtr.Time.Format("2006-01-02")
        }
        if assigneeIDPtr.Valid {
            task.AssigneeID = &assigneeIDPtr.String
        }
        
        task.CreatedAt = createdAt.Format(time.RFC3339)
        task.UpdatedAt = updatedAt.Format(time.RFC3339)
        
        tasks = append(tasks, task)
    }
    
    return tasks, nil
}
```

---

### Пример 2: Рефакторинг существующего модуля

**Задача:** Добавить поле `category` в таблицу `habits` с обратной совместимостью

#### Шаг 1: Создать миграцию

```sql
-- 000050_add_category_to_habits.up.sql
ALTER TABLE habits 
ADD COLUMN category VARCHAR(100);

CREATE INDEX idx_habits_category ON habits(category);

COMMENT ON COLUMN habits.category IS 'Категория привычки';
```

#### Шаг 2: Обновить модель

```go
// internal/model/habits.go
type Habit struct {
    // ... существующие поля
    Category string `json:"category,omitempty" db:"category"` // Новое поле
}
```

#### Шаг 3: Обновить Repository

```go
func (r *Repository) Create(ctx context.Context, dto model.CreateHabitDto, ...) {
    query := `
        INSERT INTO habits (..., category, ...)
        VALUES (..., $N, ...)
    `
    // Добавить category в Scan
}

func (r *Repository) List(ctx context.Context, ...) {
    query := `
        SELECT ..., category, ...
        FROM habits
    `
    // Добавить category в Scan
}
```

#### Шаг 4: Обновить DTOs

```go
type CreateHabitDto struct {
    // ... существующие поля
    Category string `json:"category,omitempty"` // Новое поле
}
```

**Результат:** Старый код продолжит работать, новое поле опциональное.

---

### Пример 3: Сложная транзакция

**Задача:** Перенести задачу из одного workspace в другой со всеми связями

```go
func (r *Repository) MoveToWorkspace(ctx context.Context, taskID uuid.UUID, newWorkspaceID uuid.UUID, userID uuid.UUID) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()
    
    // 1. Проверить, что задача существует и принадлежит пользователю
    var currentWorkspaceID uuid.UUID
    err = tx.QueryRowContext(ctx,
        "SELECT workspace_id FROM tasks WHERE id = $1 AND user_id = $2",
        taskID, userID,
    ).Scan(&currentWorkspaceID)
    if err != nil {
        return fmt.Errorf("task not found: %w", err)
    }
    
    if currentWorkspaceID == newWorkspaceID {
        return nil // Уже в нужном workspace
    }
    
    // 2. Обновить workspace_id задачи
    _, err = tx.ExecContext(ctx,
        "UPDATE tasks SET workspace_id = $1, updated_at = NOW() WHERE id = $2",
        newWorkspaceID, taskID,
    )
    if err != nil {
        return fmt.Errorf("failed to update task: %w", err)
    }
    
    // 3. Обновить теги (если теги привязаны к workspace)
    // Удалить теги, которые не существуют в новом workspace
    _, err = tx.ExecContext(ctx, `
        DELETE FROM task_tags tt
        USING tags t
        WHERE tt.tag_id = t.id 
          AND tt.task_id = $1
          AND t.workspace_id != $2
    `, taskID, newWorkspaceID)
    if err != nil {
        return fmt.Errorf("failed to update tags: %w", err)
    }
    
    // 4. Создать запись в истории (если есть таблица task_history)
    _, err = tx.ExecContext(ctx, `
        INSERT INTO task_history (id, task_id, user_id, action, changes, created_at)
        VALUES ($1, $2, $3, 'MOVED', $4, NOW())
    `, uuid.New(), taskID, userID, fmt.Sprintf(`{"old_workspace_id": "%s", "new_workspace_id": "%s"}`, currentWorkspaceID, newWorkspaceID))
    if err != nil {
        // Логируем, но не прерываем транзакцию
        log.Printf("Failed to log history: %v", err)
    }
    
    // 5. Коммитить транзакцию
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    return nil
}
```

---

## 📚 Чек-лист для разработки модуля

### Перед началом

- [ ] Определить сущности и их поля
- [ ] Определить связи с другими сущностями
- [ ] Определить необходимые операции (CRUD)
- [ ] Определить индексы для оптимизации

### Миграции

- [ ] Создать `.up.sql` миграцию
- [ ] Создать `.down.sql` миграцию
- [ ] Добавить Foreign Keys в `constraints/`
- [ ] Добавить индексы
- [ ] Добавить комментарии (COMMENT)
- [ ] Протестировать миграцию локально

### Модели

- [ ] Создать основную модель
- [ ] Создать DTOs (Create, Update)
- [ ] Добавить валидацию (binding tags)
- [ ] Обработать NULL значения (указатели)

### Repository

- [ ] Реализовать Create
- [ ] Реализовать Get/List
- [ ] Реализовать Update
- [ ] Реализовать Delete
- [ ] Добавить транзакции где нужно
- [ ] Обработать ошибки

### Service

- [ ] Реализовать бизнес-логику
- [ ] Добавить валидацию
- [ ] Обработать ошибки
- [ ] Проверить права доступа

### Handler

- [ ] Реализовать HTTP endpoints
- [ ] Добавить валидацию запросов
- [ ] Обработать ответы
- [ ] Добавить middleware (auth, workspace)

### Тестирование

- [ ] Протестировать создание
- [ ] Протестировать получение
- [ ] Протестировать обновление
- [ ] Протестировать удаление
- [ ] Протестировать связи
- [ ] Протестировать транзакции

---

## 🎓 Итоговые рекомендации

### Для изучения

1. **Начните с простого:** Создайте простой модуль (например, Notes) для практики
2. **Изучайте существующий код:** Смотрите на Habits модуль как на пример
3. **Практикуйтесь:** Создавайте модули по шаблону
4. **Тестируйте миграции:** Всегда тестируйте up и down миграции

### Для разработки

1. **Следуйте шаблону:** Используйте структуру Habits как шаблон
2. **Документируйте:** Комментируйте сложные запросы и логику
3. **Обрабатывайте ошибки:** Всегда обрабатывайте ошибки правильно
4. **Используйте транзакции:** Для связанных операций

### Для масштабирования

1. **Планируйте заранее:** Продумывайте структуру перед созданием
2. **Избегайте зависимостей:** Модули должны быть независимы
3. **Используйте события:** Для связи между модулями
4. **Оптимизируйте запросы:** Используйте индексы и EXPLAIN

---

## 📖 Дополнительные ресурсы

### Документация

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Go database/sql](https://pkg.go.dev/database/sql)
- [Gin Framework](https://gin-gonic.com/docs/)

### Полезные команды

```bash
# Применить миграции
migrate -path ./migrations -database "postgres://user:pass@localhost/dbname?sslmode=disable" up

# Откатить миграции
migrate -path ./migrations -database "postgres://..." down 1

# Проверить статус миграций
migrate -path ./migrations -database "postgres://..." version

# Создать новую миграцию
migrate create -ext sql -dir ./migrations -seq create_new_table
```

---

**Документ создан:** 2026-01-23  
**Версия:** 1.0  
**Автор:** AI Assistant

**Следующие шаги:**
1. Изучите этот документ полностью
2. Практикуйтесь на простом модуле (Notes)
3. Рефакторьте существующий код по шаблону
4. Создавайте новые модули самостоятельно

**Удачи в разработке! 🚀**
