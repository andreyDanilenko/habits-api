# 🚀 Быстрая шпаргалка по разработке ERP модулей

## 📋 Чек-лист создания модуля

```
1. Миграции
   □ Создать .up.sql
   □ Создать .down.sql
   □ Добавить Foreign Keys
   □ Добавить индексы
   □ Протестировать

2. Модели
   □ Основная модель
   □ CreateDto
   □ UpdateDto

3. Repository
   □ Create
   □ Get/List
   □ Update
   □ Delete

4. Service
   □ Бизнес-логика
   □ Валидация

5. Handler
   □ Endpoints
   □ Валидация

6. Тестирование
   □ Все операции
```

---

## 🗄️ Шаблон миграции

```sql
-- 000XXX_create_table_name.up.sql
CREATE TABLE table_name (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    field1 VARCHAR(255) NOT NULL,
    field2 TEXT,
    user_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_table_name_workspace ON table_name(workspace_id);
CREATE INDEX idx_table_name_user ON table_name(user_id);

COMMENT ON TABLE table_name IS 'Описание';
```

```sql
-- 000XXX_create_table_name.down.sql
DROP TABLE IF EXISTS table_name;
```

---

## 🔗 Foreign Keys

```sql
ALTER TABLE child_table 
ADD CONSTRAINT fk_child_parent 
FOREIGN KEY (parent_id) REFERENCES parent_table(id) ON DELETE CASCADE;
```

**Варианты ON DELETE:**
- `CASCADE` - удалить дочерние записи
- `SET NULL` - обнулить foreign key
- `RESTRICT` - запретить удаление (по умолчанию)

---

## 🔀 Junction Table

```sql
CREATE TABLE junction_table (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity1_id UUID NOT NULL,
    entity2_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    UNIQUE(entity1_id, entity2_id)
);

ALTER TABLE junction_table 
ADD CONSTRAINT fk_junction_entity1 
FOREIGN KEY (entity1_id) REFERENCES entity1(id) ON DELETE CASCADE;

ALTER TABLE junction_table 
ADD CONSTRAINT fk_junction_entity2 
FOREIGN KEY (entity2_id) REFERENCES entity2(id) ON DELETE CASCADE;

CREATE INDEX idx_junction_entity1 ON junction_table(entity1_id);
CREATE INDEX idx_junction_entity2 ON junction_table(entity2_id);
```

---

## 💾 Транзакция

```go
tx, err := r.db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// Операции
_, err = tx.ExecContext(ctx, "INSERT INTO ...")
if err != nil {
    return err
}

// Коммитить
return tx.Commit()
```

**Использовать когда:**
- ✅ Несколько связанных операций
- ✅ Перемещение данных
- ✅ Обновление счетчиков

---

## 📝 Шаблон модели

```go
type Entity struct {
    ID          string  `json:"id" db:"id"`
    Title       string  `json:"title" db:"title"`
    Description *string `json:"description,omitempty" db:"description"` // NULL
    UserID      string  `json:"userId" db:"user_id"`
    WorkspaceID string  `json:"workspaceId" db:"workspace_id"`
    CreatedAt   string  `json:"createdAt" db:"created_at"`
    UpdatedAt   string  `json:"updatedAt" db:"updated_at"`
}

type CreateEntityDto struct {
    Title       string  `json:"title" binding:"required"`
    Description string  `json:"description,omitempty"`
}

type UpdateEntityDto struct {
    Title       *string `json:"title,omitempty"`
    Description *string `json:"description,omitempty"`
}
```

---

## 🔍 Частые SQL запросы

### SELECT с фильтрами
```sql
SELECT id, title
FROM table_name
WHERE workspace_id = $1 
  AND user_id = $2
  AND status = 'active'
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
```

### JOIN
```sql
SELECT t.id, t.title, u.name
FROM tasks t
LEFT JOIN users u ON t.assignee_id = u.id
WHERE t.workspace_id = $1;
```

### COUNT с GROUP BY
```sql
SELECT status, COUNT(*) AS count
FROM tasks
WHERE workspace_id = $1
GROUP BY status;
```

### Подзапрос
```sql
SELECT id, title
FROM tasks
WHERE workspace_id = $1
  AND assignee_id IN (
      SELECT id FROM users WHERE workspace_id = $1
  );
```

---

## 🔄 Обновление модели

### Добавить поле
```sql
ALTER TABLE table_name 
ADD COLUMN new_field VARCHAR(100);
```

### Удалить поле
```sql
ALTER TABLE table_name 
DROP COLUMN old_field;
```

### Изменить тип (безопасно)
```sql
-- 1. Добавить новое поле
ALTER TABLE table_name ADD COLUMN field_new NEW_TYPE;

-- 2. Копировать данные
UPDATE table_name SET field_new = CAST(field AS NEW_TYPE);

-- 3. Удалить старое
ALTER TABLE table_name DROP COLUMN field;

-- 4. Переименовать
ALTER TABLE table_name RENAME COLUMN field_new TO field;
```

---

## 🎯 Команды миграций

```bash
# Применить все
migrate -path ./migrations -database "postgres://..." up

# Откатить одну
migrate -path ./migrations -database "postgres://..." down 1

# Статус
migrate -path ./migrations -database "postgres://..." version

# Создать новую
migrate create -ext sql -dir ./migrations -seq create_table
```

---

## 📚 Полезные ссылки

- **Полный гайд:** [ERP_LEARNING_GUIDE.md](../guides/ERP_LEARNING_GUIDE.md)
- **Анализ сущностей:** [ENTITIES_ANALYSIS.md](../plans/ENTITIES_ANALYSIS.md)
- **Быстрая справка Habits:** [HABITS_QUICK_REFERENCE.md](./HABITS_QUICK_REFERENCE.md)

---

**Обновлено:** 2026-01-23
