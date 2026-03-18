## СПЕЦИФИКАЦИЯ ДЛЯ БЕКЕНДА
### Этап 3: Модуль TASKS (Задачи и напоминания)

**Отдельный независимый модуль**

*Обновлено с учётом TRELO.md, TRECKER.md — конкуренция с Trello, ClickUp, Asana*

---

## 1. Общая информация

**Цель:** Реализовать полноценный модуль задач, который может работать независимо и интегрироваться с CRM через мягкие связи. Контекстные задачи, привязанные к бизнес-объектам — «лучше, чем Trello для менеджера по продажам».

**Базовый URL:** `/api/v1/workspaces/{workspaceId}/tasks`

**Принципы:**
- Модуль полностью независим (может продаваться отдельно)
- Не имеет FOREIGN KEY на CRM или другие модули
- Связь с CRM через `entityType` и `entityId` (мягкие ссылки, полиморфная связь)
- Все задачи привязаны к `workspaceId` (мультитенантность)
- RLS для изоляции данных
- ACL: просмотр у тех, кто имеет доступ к родительской сущности или добавлен явно (наблюдатели)

---

## 2. Модели данных

### 2.1. Tasks (Задачи)

**Таблица:** `tasks`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| id | UUID | Да | Первичный ключ |
| workspace_id | UUID | Да | ID рабочего пространства |
| title | VARCHAR(500) | Да | Название задачи |
| description | TEXT | Нет | Описание задачи |
| type | VARCHAR(50) | Да | call / meeting / email / lunch / other |
| priority | VARCHAR(20) | Да | low / medium / high / critical |
| status | VARCHAR(20) | Да | pending / in_progress / completed / cancelled |
| due_date | TIMESTAMP | Да | Срок выполнения |
| due_time | VARCHAR(10) | Нет | Время (ЧЧ:ММ) |
| reminder_date | TIMESTAMP | Нет | Когда напомнить |
| duration | INTEGER | Нет | Длительность в минутах |
| completed_at | TIMESTAMP | Нет | Дата выполнения |
| completed_by | UUID | Нет | Кто выполнил |
| completion_note | TEXT | Нет | Комментарий при выполнении |
| is_recurring | BOOLEAN | Да | Повторяющаяся задача |
| recurring_pattern | JSONB | Нет | Правила повторения |
| parent_id | UUID | Нет | ID родительской задачи (подзадачи) |
| assignee_id | UUID | Да | Кому назначена |
| created_by | UUID | Да | Кто создал |
| created_at | TIMESTAMP | Да | Дата создания |
| updated_at | TIMESTAMP | Да | Дата обновления |
| deleted_at | TIMESTAMP | Нет | Soft delete |
| spent_minutes | INTEGER | Нет | Затрачено времени (тайм-трекинг, суммарно) |

**Индексы:**
```sql
CREATE INDEX idx_tasks_workspace ON tasks(workspace_id, deleted_at);
CREATE INDEX idx_tasks_assignee ON tasks(assignee_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_status ON tasks(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_due_date ON tasks(due_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_reminder ON tasks(reminder_date) WHERE reminder_date IS NOT NULL;
```

---

### 2.2. Связи с внешними сущностями (мягкие ссылки)

**Таблица:** `task_entity_links`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| task_id | UUID | Да | ID задачи |
| entity_type | VARCHAR(50) | Да | 'crm_contact' / 'crm_deal' / 'crm_company' |
| entity_id | UUID | Да | ID сущности в другом модуле |
| entity_name | VARCHAR(255) | Нет | Денормализованное имя для отображения |
| created_at | TIMESTAMP | Да | Дата создания связи |

**Индексы:**
```sql
CREATE INDEX idx_task_links_task ON task_entity_links(task_id);
CREATE INDEX idx_task_links_entity ON task_entity_links(entity_type, entity_id);
```

---

### 2.3. Теги задач

**Таблица:** `task_tags`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| id | UUID | Да | Первичный ключ |
| workspace_id | UUID | Да | ID рабочего пространства |
| name | VARCHAR(100) | Да | Название тега |
| color | VARCHAR(20) | Нет | Цвет тега |
| created_by | UUID | Да | Кто создал |
| created_at | TIMESTAMP | Да | Дата создания |

**Таблица:** `task_tag_assignments`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| task_id | UUID | Да | ID задачи |
| tag_id | UUID | Да | ID тега |
| created_at | TIMESTAMP | Да | Дата присвоения |

---

### 2.4. Комментарии к задачам

**Таблица:** `task_comments`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| id | UUID | Да | Первичный ключ |
| task_id | UUID | Да | ID задачи |
| comment | TEXT | Да | Текст комментария |
| created_by | UUID | Да | Автор |
| created_at | TIMESTAMP | Да | Дата создания |
| updated_at | TIMESTAMP | Да | Дата обновления |

---

### 2.5. Наблюдатели (Watchers)

*TRECKER: получают уведомления о ходе задачи*

**Таблица:** `task_watchers`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| task_id | UUID | Да | ID задачи |
| user_id | UUID | Да | ID наблюдателя |
| created_at | TIMESTAMP | Да | Дата добавления |

**Уникальность:** (task_id, user_id)

---

### 2.6. Вложения (Attachments)

*TRELO: прикрепление файлов*

**Таблица:** `task_attachments`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| id | UUID | Да | Первичный ключ |
| task_id | UUID | Да | ID задачи |
| file_name | VARCHAR(255) | Да | Имя файла |
| file_path | VARCHAR(500) | Да | Путь в хранилище |
| file_size | INTEGER | Нет | Размер в байтах |
| mime_type | VARCHAR(100) | Нет | MIME-тип |
| uploaded_by | UUID | Да | Кто загрузил |
| created_at | TIMESTAMP | Да | Дата загрузки |

---

### 2.7. История изменений (Audit Log)

*TRECKER: «Иванов изменил описание 12.01.2024 в 15:30»*

**Таблица:** `task_history`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| id | UUID | Да | Первичный ключ |
| task_id | UUID | Да | ID задачи |
| field | VARCHAR(50) | Да | Какое поле изменено |
| old_value | TEXT | Нет | Старое значение |
| new_value | TEXT | Нет | Новое значение |
| changed_by | UUID | Да | Кто изменил |
| changed_at | TIMESTAMP | Да | Когда |

---

### 2.8. Тайм-трекинг (интервалы)

**Таблица:** `task_time_entries`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| id | UUID | Да | Первичный ключ |
| task_id | UUID | Да | ID задачи |
| user_id | UUID | Да | Кто вёл учёт |
| started_at | TIMESTAMP | Да | Начало интервала |
| stopped_at | TIMESTAMP | Нет | Конец (NULL = в процессе) |
| minutes | INTEGER | Нет | Ручной ввод (если не интервал) |
| note | TEXT | Нет | Комментарий |
| created_at | TIMESTAMP | Да | Дата создания |

---

### 2.9. Повторяющиеся задачи (структура recurring_pattern)

```json
{
  "frequency": "daily",      // daily / weekly / monthly / yearly
  "interval": 1,             // каждые N дней/недель
  "weekdays": [1,2,3,4,5],   // для weekly: дни недели (1-пн, 7-вс)
  "monthday": 15,            // для monthly: число месяца
  "end_type": "never",       // never / date / count
  "end_date": "2026-12-31",  // если end_type = date
  "end_count": 10,           // если end_type = count
  "completed_dates": []      // даты, когда уже выполнена
}
```

---

## 3. API Эндпоинты

### 3.1. Базовые операции с задачами

#### `GET /api/v1/workspaces/{workspaceId}/tasks`

**Query параметры:**
| Параметр | Тип | Описание |
|----------|-----|----------|
| status | string | pending,in_progress,completed,cancelled (через запятую) |
| priority | string | low,medium,high,critical |
| type | string | call,meeting,email,lunch,other |
| assigneeId | UUID | Фильтр по исполнителю |
| entityType | string | crm_contact, crm_deal, crm_company |
| entityId | UUID | ID сущности для связи |
| dueDateFrom | date | Срок выполнения с |
| dueDateTo | date | Срок выполнения по |
| overdue | boolean | Только просроченные |
| today | boolean | Только на сегодня |
| search | string | Поиск по названию |
| page | integer | Пагинация |
| limit | integer | Лимит |

**Ответ:**
```json
{
  "data": [
    {
      "id": "uuid",
      "title": "Позвонить клиенту",
      "description": "Обсудить условия",
      "type": "call",
      "priority": "high",
      "status": "pending",
      "dueDate": "2026-03-15T14:00:00Z",
      "reminderDate": "2026-03-15T13:00:00Z",
      "duration": 30,
      "isRecurring": false,
      "assigneeId": "uuid",
      "assigneeName": "Иван Петров",
      "createdBy": "uuid",
      "createdByName": "Анна Менеджер",
      "createdAt": "2026-03-01T10:00:00Z",
      "entities": [
        {
          "type": "crm_contact",
          "id": "uuid",
          "name": "Сергей Иванов"
        },
        {
          "type": "crm_deal",
          "id": "uuid",
          "name": "Поставка оборудования"
        }
      ],
      "tags": ["важно", "срочно"],
      "commentsCount": 2
    }
  ],
  "total": 50,
  "page": 1,
  "limit": 20
}
```

#### `GET /api/v1/workspaces/{workspaceId}/tasks/{id}`

**Ответ:** Детальная информация о задаче + все связи + комментарии

#### `POST /api/v1/workspaces/{workspaceId}/tasks`

**Тело запроса:**
```json
{
  "title": "Позвонить клиенту",
  "description": "Обсудить условия поставки",
  "type": "call",
  "priority": "high",
  "dueDate": "2026-03-15T14:00:00Z",
  "reminderDate": "2026-03-15T13:00:00Z",
  "duration": 30,
  "assigneeId": "uuid",
  "isRecurring": false,
  "recurringPattern": null,
  "parentId": null,
  "entities": [
    {
      "type": "crm_contact",
      "id": "uuid"
    }
  ],
  "tags": ["важно"]
}
```

**Валидация:**
- `title` — обязательно
- `dueDate` — обязательно, не может быть в прошлом (кроме сегодня)
- `assigneeId` — обязательно
- `type` — одно из допустимых значений

#### `PATCH /api/v1/workspaces/{workspaceId}/tasks/{id}`

**Тело запроса:** Частичное обновление

#### `DELETE /api/v1/workspaces/{workspaceId}/tasks/{id}`

**Ответ:** 204 No Content (soft delete)

---

### 3.2. Специальные операции

#### `POST /api/v1/workspaces/{workspaceId}/tasks/{id}/complete`

**Тело запроса:**
```json
{
  "note": "Клиент согласился",
  "completionDate": "2026-03-15T15:30:00Z" // опционально
}
```

**Ответ:** Обновленная задача

#### `POST /api/v1/workspaces/{workspaceId}/tasks/{id}/reopen`

**Ответ:** Задача снова в работе

#### `POST /api/v1/workspaces/{workspaceId}/tasks/{id}/remind`

**Описание:** Перенести напоминание

**Тело запроса:**
```json
{
  "reminderDate": "2026-03-16T10:00:00Z"
}
```

---

### 3.3. Мои задачи

#### `GET /api/v1/workspaces/{workspaceId}/tasks/my`

**Описание:** Задачи, где текущий пользователь исполнитель

**Query параметры:** те же, что в общем списке

#### `GET /api/v1/workspaces/{workspaceId}/tasks/overdue`

**Описание:** Просроченные задачи текущего пользователя

#### `GET /api/v1/workspaces/{workspaceId}/tasks/today`

**Описание:** Задачи на сегодня

---

### 3.4. Управление тегами

#### `GET /api/v1/workspaces/{workspaceId}/tags`

**Ответ:** Список всех тегов в workspace

#### `POST /api/v1/workspaces/{workspaceId}/tags`

**Тело запроса:**
```json
{
  "name": "важно",
  "color": "#FF0000"
}
```

#### `DELETE /api/v1/workspaces/{workspaceId}/tags/{id}`

---

### 3.5. Комментарии

#### `GET /api/v1/workspaces/{workspaceId}/tasks/{taskId}/comments`

#### `POST /api/v1/workspaces/{workspaceId}/tasks/{taskId}/comments`

```json
{
  "comment": "Обсудили детали. @userId — упомянуть коллегу (уведомление)"
}
```

#### `DELETE /api/v1/workspaces/{workspaceId}/tasks/{taskId}/comments/{id}`

---

### 3.6. Наблюдатели (Watchers)

*TRECKER: люди, получающие уведомления о ходе задачи, но не являющиеся исполнителями*

#### `GET /api/v1/workspaces/{workspaceId}/tasks/{taskId}/watchers`

#### `POST /api/v1/workspaces/{workspaceId}/tasks/{taskId}/watchers`

```json
{ "userId": "uuid" }
```

#### `DELETE /api/v1/workspaces/{workspaceId}/tasks/{taskId}/watchers/{userId}`

---

### 3.7. Вложения (Attachments)

*TRELO: прикрепление файлов к задаче*

#### `GET /api/v1/workspaces/{workspaceId}/tasks/{taskId}/attachments`

#### `POST /api/v1/workspaces/{workspaceId}/tasks/{taskId}/attachments`

`multipart/form-data` с файлом. Проверка прав доступа к родительской сделке.

#### `DELETE /api/v1/workspaces/{workspaceId}/tasks/{taskId}/attachments/{id}`

---

### 3.8. Тайм-трекинг

*TRELO/TRECKER: кнопка «Старт/Стоп» или ручной ввод часов*

#### `POST /api/v1/workspaces/{workspaceId}/tasks/{taskId}/time/start`

Начать учёт времени.

#### `POST /api/v1/workspaces/{workspaceId}/tasks/{taskId}/time/stop`

Остановить и записать интервал. Тело: `{ "note": "опционально" }`.

#### `POST /api/v1/workspaces/{workspaceId}/tasks/{taskId}/time/entry`

Ручное добавление: `{ "minutes": 30, "date": "2026-03-15" }`.

---

### 3.9. Сохранённые фильтры

*TRELO: «Сохранить выборку» (например, «Мои сделки > 100к на этапе КП»)*

#### `GET /api/v1/workspaces/{workspaceId}/tasks/filters`

**Ответ:** Список сохранённых фильтров пользователя.

#### `POST /api/v1/workspaces/{workspaceId}/tasks/filters`

```json
{
  "name": "Мои срочные",
  "params": { "status": "pending", "priority": "high", "assigneeId": "me" }
}
```

#### `DELETE /api/v1/workspaces/{workspaceId}/tasks/filters/{id}`

---

### 3.10. Массовые действия (Bulk Actions)

*TRECKER: выбрать 20 задач и одним кликом сменить дедлайн или ответственного*

#### `POST /api/v1/workspaces/{workspaceId}/tasks/bulk`

**Тело запроса:**
```json
{
  "taskIds": ["uuid1", "uuid2"],
  "action": "update_status" | "update_assignee" | "update_due_date" | "delete",
  "payload": { "status": "completed" } | { "assigneeId": "uuid" } | { "dueDate": "2026-03-20" }
}
```

---

### 3.11. Статистика

#### `GET /api/v1/workspaces/{workspaceId}/tasks/stats`

**Ответ:**
```json
{
  "total": 150,
  "byStatus": {
    "pending": 45,
    "in_progress": 30,
    "completed": 70,
    "cancelled": 5
  },
  "byPriority": {
    "low": 20,
    "medium": 60,
    "high": 50,
    "critical": 20
  },
  "overdue": 12,
  "today": 8,
  "upcoming": 25,
  "byAssignee": [
    {
      "userId": "uuid",
      "userName": "Иван Петров",
      "count": 35
    }
  ]
}
```

---

## 4. Бизнес-логика

### 4.1. При создании задачи
- Проверить существование assignee в workspace
- Если есть `entities`, сохранить связи в `task_entity_links`
- Установить статус `pending`

### 4.2. При обновлении задачи
- Проверить права (автор или admin)
- Если меняется статус на `completed`:
  - Заполнить `completed_at` и `completed_by`
  - Если задача повторяющаяся, создать следующую

### 4.3. Повторяющиеся задачи
- При завершении повторяющейся задачи:
  - Рассчитать следующую дату
  - Создать новую задачу с теми же параметрами
  - Пометить текущую как выполненную

### 4.4. Напоминания
- При создании задачи с `reminder_date`:
  - Запланировать уведомление
  - При наступлении времени отправить уведомление

### 4.5. Связи с CRM
- При удалении задачи связи удаляются
- При удалении сущности в CRM задачи остаются (связь обнуляется)

---

## 5. Дополнительно (планируемо)

**Webhooks** (TRECKER): события `task.created`, `task.completed`, `task.updated`, `task.deleted` для интеграций (Zapier, Make).

**Гибкие статусы** (TRECKER): возможность создавать кастомные статусы задач (как колонки в Trello), а не фиксированный набор pending/in_progress/completed. Таблица `workspace_task_statuses`.

**Полнотекстовый поиск**: GIN-индекс по title+description с морфологией (уже в п.6). Для поиска по комментариям — расширить или использовать pg_search.

---

## 6. Индексы для производительности

```sql
-- Основные индексы
CREATE INDEX idx_tasks_workspace_status ON tasks(workspace_id, status, due_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_assignee_status ON tasks(assignee_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_due_date_reminder ON tasks(due_date) WHERE reminder_date IS NOT NULL;

-- Поиск по названию
CREATE INDEX idx_tasks_search ON tasks USING GIN(to_tsvector('russian', title || ' ' || COALESCE(description, '')));

-- Связи
CREATE INDEX idx_task_links_composite ON task_entity_links(entity_type, entity_id);
```

---

## 7. RLS (Row Level Security)

```sql
ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE task_entity_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE task_comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE task_watchers ENABLE ROW LEVEL SECURITY;
ALTER TABLE task_attachments ENABLE ROW LEVEL SECURITY;
ALTER TABLE task_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE task_time_entries ENABLE ROW LEVEL SECURITY;

CREATE POLICY tasks_workspace_isolation ON tasks
    USING (workspace_id = current_setting('app.current_workspace_id')::UUID);

CREATE POLICY task_links_isolation ON task_entity_links
    USING (task_id IN (
        SELECT id FROM tasks 
        WHERE workspace_id = current_setting('app.current_workspace_id')::UUID
    ));
```

---

## 8. Интеграция с CRM (мягкая)

### При создании задачи из CRM:
```json
POST /api/v1/workspaces/{workspaceId}/tasks
{
  "title": "Позвонить контакту",
  "type": "call",
  "dueDate": "...",
  "assigneeId": "...",
  "entities": [
    {
      "type": "crm_contact",
      "id": "contact_123"
    }
  ]
}
```

### При получении задач для контакта:
```
GET /api/v1/workspaces/{workspaceId}/tasks?entityType=crm_contact&entityId=contact_123
```

---

## 9. Коды ответов

| Код | Описание |
|-----|----------|
| 200 | OK |
| 201 | Created |
| 204 | No Content |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 409 | Conflict |
| 422 | Unprocessable Entity |

---

## 10. Критерии приемки

**Базовые (MVP):**
- [ ] Можно создать задачу с заполнением всех полей
- [ ] Работают все фильтры (по статусу, приоритету, типу, датам)
- [ ] Работает полнотекстовый поиск по названию и описанию (морфология)
- [ ] Можно отметить задачу выполненной
- [ ] Работают повторяющиеся задачи
- [ ] Можно привязать задачу к CRM (через entities)
- [ ] Работают комментарии к задачам (в т.ч. @упоминания с уведомлением)
- [ ] Работают теги
- [ ] Есть статистика по задачам
- [ ] RLS изолирует данные между workspace
- [ ] Нет FOREIGN KEY на таблицы CRM
- [ ] Все операции проверяют права доступа

**Расширенные (TRELO/TRECKER):**
- [ ] Наблюдатели (watchers) получают уведомления
- [ ] Вложения (файлы) к задачам
- [ ] Тайм-трекинг (старт/стоп, ручной ввод)
- [ ] Сохранённые фильтры
- [ ] Массовые действия (bulk)
- [ ] История изменений (audit log)
- [ ] Виджет задач в карточке сделки CRM
- [ ] Кнопка «Быстрая задача» из любой точки CRM

---

## 11. Заключение

Модуль TASKS полностью независим:
- Не требует CRM для работы
- Может продаваться отдельно
- Интегрируется с CRM через мягкие связи
- Масштабируется горизонтально
- Поддерживает RLS для мультитенантности
