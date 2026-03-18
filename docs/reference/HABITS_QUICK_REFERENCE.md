# Быстрая справка: Habits система

## 📋 Основные сущности

### 1. Habits (Привычки)
- **Таблица:** `habits`
- **Связи:** `user_id` → `users.id`, `workspace_id` → `workspaces.id`
- **Основные поля:** title, color, icon, target_days, daily_goal, category
- **Автообновление:** `updated_at` обновляется триггером при каждом UPDATE

### 2. Habit_Completions (Выполнения)
- **Таблица:** `habit_completions`
- **Связи:** `habit_id` → `habits.id` (CASCADE), `user_id` → `users.id` (CASCADE)
- **Ограничение:** `UNIQUE(habit_id, date, user_id)` - одно выполнение в день
- **Поля:** date, notes, rating (1-5), time

### 3. Habit_History (История изменений) ⭐ НОВОЕ
- **Таблица:** `habit_history`
- **Назначение:** Хранит все изменения привычек (создание, обновление, удаление)
- **Поля:** action (CREATED/UPDATED/DELETED), changes (JSONB), metadata (JSONB)

### 4. Activities (Активность) ⭐ НОВОЕ
- **Таблица:** `activities`
- **Назначение:** Для виджета RecentActivity
- **Поля:** type, entity_type, entity_id, title, emoji

---

## 🔄 Жизненный цикл привычки

```
1. CREATE
   POST /api/habits
   → Создается запись в habits
   → Создается запись в habit_history (action: CREATED)
   → Создается запись в activities (type: HABIT_CREATED)

2. UPDATE
   PUT /api/habits/:id
   → Обновляется запись в habits
   → Триггер обновляет updated_at
   → Создается запись в habit_history (action: UPDATED, changes: {old/new})
   → Создается запись в activities (type: HABIT_UPDATED)

3. COMPLETE
   POST /api/habits/:id/complete
   → Создается запись в habit_completions
   → Создается запись в habit_history (action: COMPLETED)
   → Создается запись в activities (type: HABIT_COMPLETED)

4. DELETE
   DELETE /api/habits/:id
   → Удаляются все habit_completions (CASCADE)
   → Создается запись в habit_history (action: DELETED)
   → Удаляется запись в habits
   → Создается запись в activities (type: HABIT_DELETED)
```

---

## 📊 Связи между таблицами

```
users (1) ──< (N) habits (1) ──< (N) habit_completions
  │                              │
  │                              │
  └──< (N) user_workspaces (N) >──┘
              │
              │
              └──> (1) workspaces (1) ──< (N) habits

habits (1) ──< (N) habit_history
habits (1) ──< (N) activities
```

---

## 🔍 Частые SQL запросы

### Получить все привычки пользователя в workspace
```sql
SELECT * FROM habits 
WHERE user_id = $1 AND workspace_id = $2
ORDER BY created_at DESC;
```

### Получить выполнения за период
```sql
SELECT * FROM habit_completions
WHERE habit_id = $1 AND user_id = $2 
  AND date BETWEEN $3 AND $4
ORDER BY date DESC;
```

### Получить историю изменений привычки
```sql
SELECT * FROM habit_history
WHERE habit_id = $1 AND user_id = $2
ORDER BY created_at DESC
LIMIT 50;
```

### Получить последние активности
```sql
SELECT * FROM activities
WHERE user_id = $1 AND workspace_id = $2
ORDER BY created_at DESC
LIMIT 10;
```

---

## 🎯 Endpoints API

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/habits` | Список привычек |
| POST | `/api/habits` | Создать привычку |
| GET | `/api/habits/:id` | Получить привычку |
| PUT | `/api/habits/:id` | Обновить привычку |
| DELETE | `/api/habits/:id` | Удалить привычку |
| POST | `/api/habits/:id/complete` | Отметить выполнение |
| POST | `/api/habits/:id/toggle` | Переключить выполнение |
| GET | `/api/habits/:id/stats` | Статистика привычки |
| GET | `/api/habits/calendar` | Календарь выполнений |
| GET | `/api/habits/completions` | Список выполнений |

---

## 📝 Примеры данных

### Habit (JSON)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Чтение 30 минут",
  "description": "Читать каждый день",
  "color": "#3B82F6",
  "icon": "📚",
  "targetDays": 7,
  "dailyGoal": 1,
  "preferredTime": "20:00:00",
  "category": "Развитие",
  "userId": "user-uuid",
  "workspaceId": "workspace-uuid",
  "createdAt": "2026-01-23T10:30:00Z",
  "updatedAt": "2026-01-23T10:30:00Z"
}
```

### HabitCompletion (JSON)
```json
{
  "id": "completion-uuid",
  "habitId": "habit-uuid",
  "userId": "user-uuid",
  "date": "2026-01-23",
  "notes": "Прочитал главу 5",
  "rating": 4,
  "time": "20:30:00",
  "createdAt": "2026-01-23T20:30:00Z"
}
```

### HabitHistory (JSON)
```json
{
  "id": "history-uuid",
  "habitId": "habit-uuid",
  "userId": "user-uuid",
  "action": "UPDATED",
  "changes": {
    "title": {
      "old": "Чтение",
      "new": "Чтение книг"
    },
    "color": {
      "old": "#3B82F6",
      "new": "#10B981"
    }
  },
  "metadata": {
    "ip": "192.168.1.1",
    "workspace_id": "workspace-uuid"
  },
  "createdAt": "2026-01-23T15:45:00Z"
}
```

### Activity (JSON)
```json
{
  "id": "activity-uuid",
  "userId": "user-uuid",
  "workspaceId": "workspace-uuid",
  "type": "HABIT_COMPLETED",
  "entityType": "completion",
  "entityId": "completion-uuid",
  "title": "Завершена привычка \"Чтение\"",
  "emoji": "✅",
  "createdAt": "2026-01-23T20:30:00Z"
}
```

---

## 🚀 Следующие шаги

1. ✅ Применить миграции `000008` и `000009`
2. ✅ Реализовать методы в Repository для habit_history и activities
3. ✅ Обновить методы Create/Update/Delete для логирования в историю
4. ✅ Создать endpoint `/api/activities` для виджета
5. ✅ Обновить фронтенд RecentActivityWidget для использования реальных данных

---

**Подробная документация:** [ENTITIES_ANALYSIS.md](../plans/ENTITIES_ANALYSIS.md)
