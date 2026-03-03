## СПЕЦИФИКАЦИЯ ДЛЯ БЕКЕНДА
### Этап 2: Модуль активности (Activity Feed)

**На основе реализованного фронтенда**

---

## 1. Общая информация

**Цель:** Реализовать серверную часть для модуля активности, обеспечивающую хранение и выдачу событий в ленту активности контактов, компаний и сделок.

**Базовый URL:** `/api/v1/workspaces/{workspaceId}/crm/activities`

**Принципы:**
- Все события привязаны к `workspaceId` (мультитенантность)
- События делятся на **ручные** (создаются пользователем) и **автоматические** (генерируются системой)
- Автоматические события создаются при изменениях в сущностях CRM (контакты, компании, сделки)

---

## 2. Модели данных (БД)

### 2.1. Activity (Событие)

**Таблица:** `crm_activities`

| Поле | Тип | Обязательное | Описание | Соответствие фронтенду |
|------|-----|--------------|----------|------------------------|
| id | UUID | Да | Первичный ключ | `Activity.id` |
| workspace_id | UUID | Да | ID рабочего пространства | — |
| type | VARCHAR(50) | Да | Тип события | `Activity.type` |
| entity_type | VARCHAR(20) | Да | Тип сущности (contact/company/deal) | `Activity.entityType` |
| entity_id | UUID | Да | ID сущности | `Activity.entityId` |
| title | VARCHAR(500) | Да | Заголовок | `Activity.title` |
| description | TEXT | Нет | Описание | `Activity.description` |
| metadata | JSONB | Нет | Метаданные события | `Activity.metadata` |
| is_important | BOOLEAN | Да | Важное событие | `Activity.isImportant` |
| created_by | UUID | Да | ID пользователя-автора | `Activity.createdBy.id` |
| created_by_name | VARCHAR(255) | Да | Имя автора (денормализация) | `Activity.createdBy.name` |
| created_by_avatar | VARCHAR(500) | Нет | Аватар автора | `Activity.createdBy.avatar` |
| is_editable | BOOLEAN | Да | Можно редактировать (только для note) | `Activity.isEditable` |
| is_deletable | BOOLEAN | Да | Можно удалять (только для note) | `Activity.isDeletable` |
| created_at | TIMESTAMP | Да | Дата создания | `Activity.createdAt` |
| updated_at | TIMESTAMP | Да | Дата обновления | — |
| deleted_at | TIMESTAMP | Нет | Soft delete | — |

**Индексы:**
```sql
CREATE INDEX idx_activities_workspace ON crm_activities(workspace_id);
CREATE INDEX idx_activities_entity ON crm_activities(entity_type, entity_id);
CREATE INDEX idx_activities_type ON crm_activities(type);
CREATE INDEX idx_activities_created_at ON crm_activities(created_at);
CREATE INDEX idx_activities_important ON crm_activities(is_important) WHERE is_important = true;
CREATE INDEX idx_activities_search ON crm_activities USING GIN(to_tsvector('russian', title || ' ' || COALESCE(description, '')));
```

### 2.2. Activity Files (Файлы, прикрепленные к заметкам)

**Таблица:** `crm_activity_files`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| id | UUID | Да | Первичный ключ |
| activity_id | UUID | Да | ID события |
| name | VARCHAR(500) | Да | Имя файла |
| size | INTEGER | Да | Размер в байтах |
| type | VARCHAR(100) | Да | MIME-тип |
| url | VARCHAR(1000) | Да | URL для скачивания |
| created_at | TIMESTAMP | Да | Дата загрузки |

### 2.3. Activity Reminders (Напоминания из заметок)

**Таблица:** `crm_activity_reminders`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| id | UUID | Да | Первичный ключ |
| activity_id | UUID | Да | ID события (заметки) |
| remind_at | TIMESTAMP | Да | Дата и время напоминания |
| assign_to | UUID | Нет | ID пользователя, которому назначено |
| is_completed | BOOLEAN | Да | Выполнено/закрыто |
| created_at | TIMESTAMP | Да | Дата создания |

---

## 3. API Эндпоинты

### 3.1. Получение списка активностей

#### `GET /api/v1/workspaces/{workspaceId}/crm/activities`

**Описание:** Получение ленты активности для конкретной сущности с фильтрацией и пагинацией.

**Query параметры:**
| Параметр | Тип | Обязательный | Описание |
|----------|-----|--------------|----------|
| entityType | string | **Да** | Тип сущности: `contact`, `company`, `deal` |
| entityId | UUID | **Да** | ID сущности |
| page | integer | Нет | Номер страницы (default: 1) |
| limit | integer | Нет | Лимит (default: 50, max: 100) |
| types | string | Нет | Фильтр по типам (через запятую: `note,call,email`) |
| dateFrom | date | Нет | Начало периода (ISO: YYYY-MM-DD) |
| dateTo | date | Нет | Конец периода (ISO: YYYY-MM-DD) |
| importantOnly | boolean | Нет | Только важные (default: false) |
| search | string | Нет | Поиск по title и description |

**Пример запроса:**
```
GET /api/v1/workspaces/wks_123/crm/activities?entityType=contact&entityId=contact_456&page=1&limit=50&types=note,call&importantOnly=true
```

**Ответ:**
```json
{
  "data": [
    {
      "id": "act_123",
      "type": "call",
      "entityType": "contact",
      "entityId": "contact_456",
      "title": "Исходящий звонок",
      "description": "Обсудили условия поставки",
      "metadata": {
        "callDuration": 315,
        "callDirection": "out",
        "callStatus": "answered"
      },
      "isImportant": true,
      "createdBy": {
        "id": "user_789",
        "name": "Анна Менеджер",
        "avatar": "/avatars/anna.jpg"
      },
      "createdAt": "2026-02-28T10:23:00Z",
      "isEditable": false,
      "isDeletable": false
    }
  ],
  "total": 150,
  "page": 1,
  "limit": 50
}
```

**Коды ответов:**
- `200 OK` — успешно
- `400 Bad Request` — неверные параметры
- `403 Forbidden` — нет прав на просмотр

---

### 3.2. Создание события (ручное)

#### `POST /api/v1/workspaces/{workspaceId}/crm/activities`

**Описание:** Создание нового события (заметки или записи звонка).

**Тело запроса (для заметки — `CreateNoteDto`):**
```json
{
  "type": "note",
  "entityType": "contact",
  "entityId": "contact_456",
  "title": "Важная встреча",
  "description": "Обсудили детали сотрудничества",
  "isImportant": true,
  "reminder": {
    "date": "2026-03-15T14:00:00Z",
    "assignTo": "user_789"
  }
}
```

**Тело запроса (для звонка — `CreateCallDto`):**
```json
{
  "type": "call",
  "entityType": "contact",
  "entityId": "contact_456",
  "title": "Исходящий звонок",
  "description": "Клиент запросил коммерческое предложение",
  "metadata": {
    "callDirection": "out",
    "callStatus": "answered",
    "callDuration": 315
  },
  "isImportant": false
}
```

**Валидация:**
- `type` — только `note` или `call` (остальные типы генерируются системой)
- `entityType` и `entityId` — обязательны, сущность должна существовать
- `title` — обязателен, максимум 500 символов
- Для `type=call`:
  - `metadata.callDirection` и `metadata.callStatus` обязательны
  - Если `callStatus = answered`, то `metadata.callDuration` обязателен (минимум 1)
- Для `type=note` с `reminder`:
  - `reminder.date` обязателен, должен быть в будущем

**Ответ:** `201 Created` + созданный объект (как в GET)

**Коды ответов:**
- `201 Created` — успешно создано
- `400 Bad Request` — ошибка валидации
- `403 Forbidden` — нет прав на создание
- `404 Not Found` — entityType/entityId не найдены

---

### 3.3. Обновление заметки

#### `PATCH /api/v1/workspaces/{workspaceId}/crm/activities/{id}`

**Описание:** Обновление существующей заметки (только для `type=note`).

**Тело запроса (`UpdateNoteDto`):**
```json
{
  "title": "Обновленный заголовок",
  "description": "Новый текст заметки",
  "isImportant": true
}
```

**Валидация:**
- Событие должно существовать и иметь `type=note`
- Пользователь должен быть автором или иметь права администратора
- Нельзя обновлять автоматические события

**Ответ:** `200 OK` + обновленный объект

**Коды ответов:**
- `200 OK` — успешно
- `400 Bad Request` — неверные данные
- `403 Forbidden` — нет прав
- `404 Not Found` — событие не найдено
- `409 Conflict` — попытка обновить не-заметку

---

### 3.4. Удаление заметки

#### `DELETE /api/v1/workspaces/{workspaceId}/crm/activities/{id}`

**Описание:** Удаление заметки (только для `type=note`).

**Валидация:**
- Событие должно существовать и иметь `type=note`
- Пользователь должен быть автором или иметь права администратора
- Автоматические события удалять нельзя

**Ответ:** `204 No Content`

**Коды ответов:**
- `204 No Content` — успешно удалено
- `403 Forbidden` — нет прав
- `404 Not Found` — событие не найдено
- `409 Conflict` — попытка удалить не-заметку

---

### 3.5. Пометка важности

#### `POST /api/v1/workspaces/{workspaceId}/crm/activities/{id}/important`

**Описание:** Установка или снятие флага важности для любого события.

**Тело запроса:**
```json
{
  "isImportant": true
}
```

**Ответ:** `200 OK` + обновленный объект

**Коды ответов:**
- `200 OK` — успешно
- `400 Bad Request` — неверные данные
- `403 Forbidden` — нет прав
- `404 Not Found` — событие не найдено

---

### 3.6. Загрузка файлов для заметки

#### `POST /api/v1/workspaces/{workspaceId}/crm/activities/{id}/files`

**Описание:** Загрузка файлов, прикрепленных к заметке.

**Формат:** `multipart/form-data`

**Параметры:**
- `files[]` — массив файлов (максимум 10 файлов, каждый до 25 МБ)

**Ответ:** `200 OK` + обновленный объект с metadata.files

**Коды ответов:**
- `200 OK` — файлы загружены
- `400 Bad Request` — превышен размер, неверный формат
- `403 Forbidden` — нет прав
- `404 Not Found` — событие не найдено

---

## 4. Автоматические события (системные триггеры)

### 4.1. При создании сущностей

| Событие | Когда срабатывает | Формат title |
|---------|-------------------|--------------|
| `contact_created` | Создан новый контакт | "Контакт создан" |
| `company_created` | Создана новая компания | "Компания создана" |
| `deal_created` | Создана новая сделка | "Сделка создана на этапе {stageName}" |

**Пример генерации:**
```sql
-- Триггер после INSERT на crm_contacts
INSERT INTO crm_activities (
  id, workspace_id, type, entity_type, entity_id,
  title, created_by, created_by_name, is_editable, is_deletable, created_at
)
SELECT
  gen_random_uuid(),
  NEW.workspace_id,
  'contact_created',
  'contact',
  NEW.id,
  'Контакт создан',
  NEW.created_by,
  COALESCE(u.name, 'Система'),
  false,
  false,
  NOW()
FROM users u WHERE u.id = NEW.created_by;
```

### 4.2. При изменении сущностей

| Событие | Когда срабатывает | Формат |
|---------|-------------------|--------|
| `contact_updated` | Изменены поля контакта | "Изменены поля: телефон, email" |
| `company_updated` | Изменены поля компании | "Изменены реквизиты компании" |
| `deal_stage_changed` | Смена этапа сделки | "Сделка перешла: {from} → {to}" |
| `file_attached` | Прикреплен файл к сущности | "Прикреплен файл {filename}" |

**Логика для `contact_updated`:**
- Сравнивать старые и новые значения значимых полей
- Записывать в `metadata.changedFields` массив изменений
- Не создавать событие, если изменения незначительны (например, только updated_at)

**Логика для `deal_stage_changed`:**
- Срабатывает при обновлении `stage_id`
- Получать названия старого и нового этапов
- Записывать в `metadata.fromStage` и `metadata.toStage`

### 4.3. Техническая реализация

**Вариант 1: Триггеры в БД (рекомендуется для простоты)**
```sql
CREATE OR REPLACE FUNCTION trigger_deal_stage_changed()
RETURNS TRIGGER AS $$
BEGIN
  IF OLD.stage_id != NEW.stage_id THEN
    INSERT INTO crm_activities (...)
    VALUES (...);
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER deal_stage_trigger
AFTER UPDATE ON crm_deals
FOR EACH ROW
EXECUTE FUNCTION trigger_deal_stage_changed();
```

**Вариант 2: На уровне приложения (более гибко)**
- В сервисе после успешного обновления сущности вызывать метод `activityService.createSystemEvent()`
- Плюс: можно добавить дополнительную логику, минус: риск забыть вызвать

---

## 5. Индексы для производительности

```sql
-- Составной индекс для быстрого получения ленты
CREATE INDEX idx_activities_lookup ON crm_activities(workspace_id, entity_type, entity_id, created_at DESC);

-- Индекс для фильтрации по типам
CREATE INDEX idx_activities_type_filter ON crm_activities(entity_type, entity_id, type) WHERE type IN ('note', 'call', 'email');

-- Индекс для поиска по датам
CREATE INDEX idx_activities_date_range ON crm_activities(created_at) WHERE created_at > NOW() - INTERVAL '30 days';

-- Полнотекстовый поиск
CREATE INDEX idx_activities_fts ON crm_activities USING GIN(to_tsvector('russian', title || ' ' || COALESCE(description, '')));
```

---

## 6. Обработка ошибок

### 6.1. Специфичные для модуля активности

| Ситуация | Код | Сообщение |
|----------|-----|-----------|
| Попытка обновить системное событие | 409 | "Cannot update system-generated activity" |
| Попытка удалить системное событие | 409 | "Cannot delete system-generated activity" |
| Файл превышает размер | 400 | "File size exceeds 25MB limit" |
| Неверный формат файла | 400 | "Unsupported file type" |
| Событие не принадлежит workspace | 404 | "Activity not found" |
| Нет прав на создание для этой сущности | 403 | "No permission to create activity for this entity" |

### 6.2. Формат ошибки

```json
{
  "code": 409,
  "message": "Cannot update system-generated activity",
  "details": {
    "activityId": "act_123",
    "activityType": "deal_stage_changed"
  }
}
```

---

## 7. Интеграция с существующими модулями

### 7.1. Интеграция с контактами

При создании контакта:
```sql
-- Автоматически создавать activity
INSERT INTO crm_activities (...)
VALUES (..., 'contact_created', 'Контакт создан', ...);
```

### 7.2. Интеграция с компаниями

При изменении компании:
```sql
-- Если изменились важные поля
INSERT INTO crm_activities (...)
VALUES (..., 'company_updated', 'Изменены реквизиты компании', 
        jsonb_build_object('changedFields', ...));
```

### 7.3. Интеграция со сделками

При смене этапа:
```sql
-- Получаем названия этапов
SELECT name INTO old_stage_name FROM crm_stages WHERE id = OLD.stage_id;
SELECT name INTO new_stage_name FROM crm_stages WHERE id = NEW.stage_id;

INSERT INTO crm_activities (...)
VALUES (..., 'deal_stage_changed',
        format('Сделка перешла: %s → %s', old_stage_name, new_stage_name),
        jsonb_build_object('fromStage', jsonb_build_object('id', OLD.stage_id, 'name', old_stage_name),
                           'toStage', jsonb_build_object('id', NEW.stage_id, 'name', new_stage_name),
                           'dealValue', NEW.budget));
```

---

## 8. Безопасность и права

### 8.1. Проверка прав

| Действие | Необходимое право |
|----------|-------------------|
| Просмотр ленты | `CRM_VIEW` |
| Создание заметки/звонка | `CRM_CREATE` |
| Редактирование заметки | `CRM_EDIT` + автор или администратор |
| Удаление заметки | `CRM_DELETE` + автор или администратор |
| Пометка важности | `CRM_EDIT` |

### 8.2. Проверка workspace

```sql
-- Все запросы должны проверять принадлежность к workspace
SELECT 1 FROM crm_activities 
WHERE id = :activityId 
AND workspace_id = :workspaceId;
```

---

## 9. Ограничения и допущения

### 9.1. Бизнес-ограничения
- Максимальная длина заголовка: 500 символов
- Максимальная длина описания: 10000 символов
- Максимальное количество файлов на заметку: 10
- Максимальный размер файла: 25 МБ
- Поддерживаемые типы файлов: изображения, PDF, DOC, DOCX, XLS, XLSX, TXT

### 9.2. Технические ограничения
- Пагинация: не более 100 записей за запрос
- Период для фильтрации: не более 1 года (во избежание перегрузки)
- Полнотекстовый поиск: минимальная длина слова 3 символа

---

## 10. Критерии приемки

- [ ] API возвращает корректные списки активностей с пагинацией
- [ ] Работают все фильтры (по типам, датам, важности, поиск)
- [ ] Можно создать заметку с текстом и файлами
- [ ] Можно записать звонок (вручную)
- [ ] Можно пометить любое событие важным
- [ ] Можно отредактировать/удалить только свои заметки
- [ ] Система автоматически создает события при:
  - Создании контакта
  - Изменении контакта
  - Создании компании
  - Изменении компании
  - Создании сделки
  - Смене этапа сделки
- [ ] Все события правильно группируются по датам
- [ ] Индексы обеспечивают быстрый поиск (проверить EXPLAIN)
- [ ] Права доступа работают согласно таблице в п. 8.1

---

## 11. Примеры запросов (для тестирования)

### Создать заметку
```bash
curl -X POST /api/v1/workspaces/wks_123/crm/activities \
  -H "Authorization: Bearer token" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "note",
    "entityType": "contact",
    "entityId": "contact_456",
    "title": "Важная встреча",
    "description": "Обсудили детали",
    "isImportant": true
  }'
```

### Получить ленту с фильтрами
```bash
curl "/api/v1/workspaces/wks_123/crm/activities?entityType=contact&entityId=contact_456&types=note,call&importantOnly=true&dateFrom=2026-02-01&dateTo=2026-02-28"
```

### Пометить важным
```bash
curl -X POST /api/v1/workspaces/wks_123/crm/activities/act_123/important \
  -H "Authorization: Bearer token" \
  -H "Content-Type: application/json" \
  -d '{"isImportant": true}'
```

---

Эта спецификация полностью соответствует реализованному фронтенду и обеспечивает:
1. Все необходимые эндпоинты
2. Правильную структуру данных
3. Автоматическую генерацию системных событий
4. Масштабируемость через индексы
5. Безопасность через проверку прав
