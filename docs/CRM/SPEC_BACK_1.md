## СПЕЦИФИКАЦИЯ ДЛЯ БЕКЕНДА (Первый этап - Ядро CRM)

### На основе ваших моделей

---

## 1. Общая информация

**Цель:** Реализовать серверную часть для базовых сущностей CRM, обеспечивающую CRUD операции и бизнес-логику.

**Базовый URL:** `/api/v1/workspaces/{workspaceId}/crm`

**Принципы:**
- Все сущности привязаны к `workspaceId` (мультитенантность)
- Архитектура позволяет в будущем выделить каждый модуль в отдельный микросервис
- Четкое разделение по доменам: contacts, companies, deals, pipelines

---

## 2. Модели данных (БД)

### 2.1. Contact (из вашей модели)

**Таблица:** `crm_contacts`

| Поле | Тип | Обязательное | Описание | Из модели |
|------|-----|--------------|----------|-----------|
| id | UUID | Да | Первичный ключ | `Contact.id` |
| workspace_id | UUID | Да | ID рабочего пространства | *(добавить)* |
| first_name | VARCHAR(100) | Да | Имя | `Contact.firstName` |
| last_name | VARCHAR(100) | Да | Фамилия | `Contact.lastName` |
| middle_name | VARCHAR(100) | Нет | Отчество | `Contact.middleName` |
| company_id | UUID | Нет | ID компании | `Contact.companyId` |
| position | VARCHAR(200) | Нет | Должность | `Contact.position` |
| birthday | DATE | Нет | Дата рождения | `Contact.birthday` |
| tags | TEXT[] | Нет | Массив тегов | `Contact.tags` |
| owner_id | UUID | Да | Ответственный | `Contact.ownerId` |
| created_by | UUID | Да | Кто создал | `Contact.createdBy` |
| updated_by | UUID | Да | Кто изменил | `Contact.updatedBy` |
| created_at | TIMESTAMP | Да | Дата создания | `Contact.createdAt` |
| updated_at | TIMESTAMP | Да | Дата изменения | `Contact.updatedAt` |
| custom_fields | JSONB | Нет | Динамические поля | `Contact.customFields` |
| deleted_at | TIMESTAMP | Нет | Soft delete | *(для безопасности)* |

**Таблица:** `crm_contact_phones`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| id | UUID | Да | Первичный ключ |
| contact_id | UUID | Да | ID контакта |
| type | VARCHAR(20) | Да | mobile/work/home |
| number | VARCHAR(50) | Да | Номер |
| is_primary | BOOLEAN | Да | Основной |

**Таблица:** `crm_contact_emails`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| id | UUID | Да | Первичный ключ |
| contact_id | UUID | Да | ID контакта |
| type | VARCHAR(20) | Да | work/personal |
| address | VARCHAR(255) | Да | Email |
| is_primary | BOOLEAN | Да | Основной |

---

### 2.2. Company (из вашей модели)

**Таблица:** `crm_companies`

| Поле | Тип | Обязательное | Описание | Из модели |
|------|-----|--------------|----------|-----------|
| id | UUID | Да | Первичный ключ | `Company.id` |
| workspace_id | UUID | Да | ID рабочего пространства | *(добавить)* |
| name | VARCHAR(255) | Да | Название | `Company.name` |
| inn | VARCHAR(12) | Нет | ИНН | `Company.inn` |
| kpp | VARCHAR(9) | Нет | КПП | `Company.kpp` |
| ogrn | VARCHAR(15) | Нет | ОГРН | `Company.ogrn` |
| phone | VARCHAR(50) | Нет | Телефон | `Company.phone` |
| email | VARCHAR(255) | Нет | Email | `Company.email` |
| website | VARCHAR(255) | Нет | Сайт | `Company.website` |
| legal_address | JSONB | Нет | Юр. адрес | `Company.legalAddress` |
| actual_address | JSONB | Нет | Факт. адрес | `Company.actualAddress` |
| tags | TEXT[] | Нет | Теги | `Company.tags` |
| owner_id | UUID | Да | Ответственный | `Company.ownerId` |
| created_at | TIMESTAMP | Да | Дата создания | `Company.createdAt` |
| updated_at | TIMESTAMP | Да | Дата изменения | `Company.updatedAt` |
| deleted_at | TIMESTAMP | Нет | Soft delete | *(добавить)* |

**Таблица связи:** `crm_company_contacts`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| company_id | UUID | Да | ID компании |
| contact_id | UUID | Да | ID контакта |
| position | VARCHAR(200) | Нет | Должность (избыточно, но удобно) |
| created_at | TIMESTAMP | Да | Дата привязки |

---

### 2.3. Pipeline и Stage (из вашей модели)

**Таблица:** `crm_pipelines`

| Поле | Тип | Обязательное | Описание | Из модели |
|------|-----|--------------|----------|-----------|
| id | UUID | Да | Первичный ключ | `Pipeline.id` |
| workspace_id | UUID | Да | ID рабочего пространства | *(добавить)* |
| name | VARCHAR(255) | Да | Название | `Pipeline.name` |
| is_default | BOOLEAN | Да | По умолчанию | `Pipeline.isDefault` |
| created_by | UUID | Да | Кто создал | *(добавить)* |
| created_at | TIMESTAMP | Да | Дата создания | *(добавить)* |

**Таблица:** `crm_stages`

| Поле | Тип | Обязательное | Описание | Из модели |
|------|-----|--------------|----------|-----------|
| id | UUID | Да | Первичный ключ | `Stage.id` |
| pipeline_id | UUID | Да | ID воронки | (связь) |
| name | VARCHAR(255) | Да | Название | `Stage.name` |
| order_index | INTEGER | Да | Порядок | `Stage.order` |
| color | VARCHAR(20) | Нет | Цвет | `Stage.color` |
| probability | INTEGER | Да | Вероятность | `Stage.probability` |
| is_final | BOOLEAN | Да | Финальный | `Stage.isFinal` |
| is_lost | BOOLEAN | Да | Проигрыш | `Stage.isLost` |
| created_at | TIMESTAMP | Да | Дата создания | *(добавить)* |

**Ограничения:**
- В одной воронке только один этап с `is_final = true`
- В одной воронке только один этап с `is_lost = true`
- `is_final` и `is_lost` не могут быть одновременно true

---

### 2.4. Deal (из вашей модели)

**Таблица:** `crm_deals`

| Поле | Тип | Обязательное | Описание | Из модели |
|------|-----|--------------|----------|-----------|
| id | UUID | Да | Первичный ключ | `Deal.id` |
| workspace_id | UUID | Да | ID рабочего пространства | *(добавить)* |
| name | VARCHAR(500) | Да | Название | `Deal.name` |
| contact_id | UUID | Нет | ID контакта | `Deal.contactId` |
| company_id | UUID | Нет | ID компании | `Deal.companyId` |
| budget | DECIMAL(15,2) | Да | Бюджет | `Deal.budget` |
| currency | VARCHAR(3) | Да | Валюта | `Deal.currency` |
| pipeline_id | UUID | Да | ID воронки | `Deal.pipelineId` |
| stage_id | UUID | Да | ID этапа | `Deal.stageId` |
| expected_close_date | DATE | Нет | План. дата | `Deal.expectedCloseDate` |
| actual_close_date | DATE | Нет | Факт. дата | `Deal.actualCloseDate` |
| status | VARCHAR(20) | Да | Статус | `Deal.status` |
| lost_reason | TEXT | Нет | Причина проигрыша | `Deal.lostReason` |
| description | TEXT | Нет | Описание | `Deal.description` |
| source | VARCHAR(100) | Нет | Источник | `Deal.source` |
| probability | INTEGER | Нет | Вероятность | `Deal.probability` |
| tags | TEXT[] | Нет | Теги | `Deal.tags` |
| owner_id | UUID | Да | Ответственный | `Deal.ownerId` |
| created_at | TIMESTAMP | Да | Дата создания | `Deal.createdAt` |
| updated_at | TIMESTAMP | Да | Дата изменения | `Deal.updatedAt` |
| deleted_at | TIMESTAMP | Нет | Soft delete | *(добавить)* |

**Ограничения:**
- Должен быть указан хотя бы один: `contact_id` или `company_id`
- `stage_id` должен принадлежать `pipeline_id`
- При `status = won` или `lost` должна быть заполнена `actual_close_date`

---

## 3. API Эндпоинты

### 3.1. Контакты (Contacts)

#### `GET /api/v1/workspaces/{workspaceId}/crm/contacts`

**Query параметры:**
| Параметр | Тип | Описание |
|----------|-----|----------|
| page | integer | Номер страницы (default: 1) |
| limit | integer | Лимит (default: 50, max: 100) |
| search | string | Поиск по имени, email, телефону |
| companyId | UUID | Фильтр по компании |
| ownerId | UUID | Фильтр по ответственному |
| tag | string | Фильтр по тегу |
| dateFrom | date | Созданы после |
| dateTo | date | Созданы до |
| sortBy | string | firstName, lastName, createdAt |
| sortOrder | string | asc, desc |

**Ответ:**
```json
{
  "data": [
    {
      "id": "uuid",
      "firstName": "Иван",
      "lastName": "Петров",
      "middleName": "Иванович",
      "phones": [
        { "type": "mobile", "number": "+79991234567", "isPrimary": true }
      ],
      "emails": [
        { "type": "work", "address": "ivan@company.ru", "isPrimary": true }
      ],
      "companyId": "uuid",
      "position": "Директор",
      "birthday": "1990-01-01",
      "tags": ["VIP"],
      "ownerId": "uuid",
      "createdBy": "uuid",
      "updatedBy": "uuid",
      "createdAt": "2026-01-01T10:00:00Z",
      "updatedAt": "2026-01-01T10:00:00Z",
      "customFields": {}
    }
  ],
  "total": 100,
  "page": 1,
  "limit": 50
}
```

#### `GET /api/v1/workspaces/{workspaceId}/crm/contacts/{id}`

**Ответ:** Объект контакта (как выше) + `company` (объект компании, если есть)

#### `POST /api/v1/workspaces/{workspaceId}/crm/contacts`

**Тело запроса:** `CreateContactDto` (из вашей модели)

```json
{
  "firstName": "Иван",
  "lastName": "Петров",
  "middleName": "Иванович",
  "phones": [
    { "type": "mobile", "number": "+79991234567", "isPrimary": true }
  ],
  "emails": [
    { "type": "work", "address": "ivan@company.ru", "isPrimary": true }
  ],
  "companyId": "uuid",
  "position": "Директор",
  "birthday": "1990-01-01",
  "tags": ["VIP"],
  "ownerId": "uuid"
}
```

**Валидация:**
- `firstName` — обязателен
- Хотя бы один телефон или email
- Если есть `companyId` — проверить существование компании
- `ownerId` — должен быть пользователем workspace

**Ответ:** 201 Created + созданный объект

#### `PATCH /api/v1/workspaces/{workspaceId}/crm/contacts/{id}`

**Тело запроса:** `UpdateContactDto` (частичное обновление)

**Ответ:** 200 OK + обновленный объект

#### `DELETE /api/v1/workspaces/{workspaceId}/crm/contacts/{id}`

**Ответ:** 204 No Content

**Важно:** Проверить, нет ли связанных сделок. Если есть — либо запретить удаление (409 Conflict), либо отвязать (по бизнес-логике).

---

### 3.2. Компании (Companies)

#### `GET /api/v1/workspaces/{workspaceId}/crm/companies`

**Query параметры:**
| Параметр | Тип | Описание |
|----------|-----|----------|
| page | integer | Номер страницы |
| limit | integer | Лимит |
| search | string | Поиск по названию, ИНН |
| ownerId | UUID | Фильтр по ответственному |
| tag | string | Фильтр по тегу |

**Ответ:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "ООО Ромашка",
      "inn": "7701234567",
      "kpp": "770101001",
      "ogrn": "1027700123456",
      "phone": "+74951234567",
      "email": "info@romashka.ru",
      "website": "romashka.ru",
      "legalAddress": {
        "country": "Россия",
        "city": "Москва",
        "street": "Ленина",
        "building": "1"
      },
      "actualAddress": {
        "country": "Россия",
        "city": "Москва",
        "street": "Ленина",
        "building": "1"
      },
      "contacts": ["uuid1", "uuid2"],
      "tags": ["поставщик"],
      "ownerId": "uuid",
      "createdAt": "2026-01-01T10:00:00Z",
      "updatedAt": "2026-01-01T10:00:00Z"
    }
  ],
  "total": 50,
  "page": 1,
  "limit": 50
}
```

#### `GET /api/v1/workspaces/{workspaceId}/crm/companies/{id}`

**Ответ:** Объект компании + массив связанных контактов (полные объекты)

#### `POST /api/v1/workspaces/{workspaceId}/crm/companies`

**Тело запроса:** `CreateCompanyDto` (из вашей модели)

```json
{
  "name": "ООО Ромашка",
  "inn": "7701234567",
  "kpp": "770101001",
  "ogrn": "1027700123456",
  "phone": "+74951234567",
  "email": "info@romashka.ru",
  "website": "romashka.ru",
  "legalAddress": {
    "country": "Россия",
    "city": "Москва",
    "street": "Ленина",
    "building": "1"
  },
  "tags": ["поставщик"],
  "ownerId": "uuid"
}
```

**Валидация:**
- `name` — обязательно
- ИНН, если указан — проверка формата (10 или 12 цифр)
- Email — проверка формата

#### `PATCH /api/v1/workspaces/{workspaceId}/crm/companies/{id}`

**Тело запроса:** `UpdateCompanyDto` (частичное обновление)

#### `DELETE /api/v1/workspaces/{workspaceId}/crm/companies/{id}`

**Ответ:** 204 No Content

**Важно:** Проверить связанные контакты и сделки. Если есть — запретить удаление (409 Conflict).

---

### 3.3. Связи между контактами и компаниями

#### `POST /api/v1/workspaces/{workspaceId}/crm/companies/{companyId}/contacts/{contactId}`

**Описание:** Привязать контакт к компании

**Тело запроса (опционально):**
```json
{
  "position": "Генеральный директор"
}
```

**Ответ:** 200 OK

#### `DELETE /api/v1/workspaces/{workspaceId}/crm/companies/{companyId}/contacts/{contactId}`

**Описание:** Отвязать контакт от компании

**Ответ:** 204 No Content

---

### 3.4. Воронки (Pipelines)

#### `GET /api/v1/workspaces/{workspaceId}/crm/pipelines`

**Описание:** Получение всех воронок workspace с этапами

**Ответ:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "Основные продажи",
      "isDefault": true,
      "stages": [
        {
          "id": "uuid",
          "name": "Первичный контакт",
          "order": 1,
          "color": "#2196F3",
          "probability": 20,
          "isFinal": false,
          "isLost": false
        },
        {
          "id": "uuid",
          "name": "Успешно",
          "order": 5,
          "color": "#4CAF50",
          "probability": 100,
          "isFinal": true,
          "isLost": false
        }
      ]
    }
  ]
}
```

#### `POST /api/v1/workspaces/{workspaceId}/crm/pipelines`

**Описание:** Создание воронки с этапами

**Тело запроса:**
```json
{
  "name": "Новая воронка",
  "isDefault": false,
  "stages": [
    { "name": "Этап 1", "probability": 10, "color": "#FF9800", "order": 1 },
    { "name": "Этап 2", "probability": 50, "color": "#2196F3", "order": 2 },
    { "name": "Успешно", "probability": 100, "isFinal": true, "color": "#4CAF50", "order": 3 },
    { "name": "Проигрыш", "probability": 0, "isLost": true, "color": "#F44336", "order": 4 }
  ]
}
```

**Валидация:**
- Должен быть один этап с `isFinal: true`
- Должен быть один этап с `isLost: true` (опционально)
- Порядок этапов определяется полем `order`

---

### 3.5. Сделки (Deals)

#### `GET /api/v1/workspaces/{workspaceId}/crm/deals`

**Query параметры:**
| Параметр | Тип | Описание |
|----------|-----|----------|
| page | integer | Номер страницы |
| limit | integer | Лимит |
| pipelineId | UUID | Фильтр по воронке |
| stageId | UUID | Фильтр по этапу |
| companyId | UUID | Фильтр по компании |
| contactId | UUID | Фильтр по контакту |
| ownerId | UUID | Фильтр по ответственному |
| status | string | open/won/lost |
| dateFrom | date | Созданы после |
| dateTo | date | Созданы до |

**Ответ:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "Поставка оборудования",
      "contactId": "uuid",
      "companyId": "uuid",
      "budget": 1500000.00,
      "currency": "RUB",
      "pipelineId": "uuid",
      "stageId": "uuid",
      "expectedCloseDate": "2026-03-15",
      "actualCloseDate": null,
      "status": "open",
      "lostReason": null,
      "description": "Поставка станков",
      "source": "Сайт",
      "probability": 80,
      "tags": ["срочно"],
      "ownerId": "uuid",
      "createdAt": "2026-02-01T10:00:00Z",
      "updatedAt": "2026-02-01T10:00:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "limit": 50
}
```

#### `GET /api/v1/workspaces/{workspaceId}/crm/deals/{id}`

**Ответ:** Объект сделки + контакт и компания (полные объекты, если есть)

#### `POST /api/v1/workspaces/{workspaceId}/crm/deals`

**Тело запроса:** `CreateDealDto` (из вашей модели)

```json
{
  "name": "Поставка оборудования",
  "contactId": "uuid",
  "companyId": "uuid",
  "budget": 1500000.00,
  "currency": "RUB",
  "pipelineId": "uuid",
  "stageId": "uuid",
  "expectedCloseDate": "2026-03-15",
  "description": "Поставка станков",
  "source": "Сайт",
  "probability": 80,
  "tags": ["срочно"],
  "ownerId": "uuid"
}
```

**Валидация:**
- `name`, `budget`, `pipelineId`, `stageId`, `ownerId` — обязательны
- `contactId` или `companyId` — хотя бы один
- `stageId` должен принадлежать `pipelineId`

#### `PATCH /api/v1/workspaces/{workspaceId}/crm/deals/{id}`

**Тело запроса:** `UpdateDealDto` (частичное обновление)

```json
{
  "stageId": "новый_этап",
  "budget": 2000000.00
}
```

#### `POST /api/v1/workspaces/{workspaceId}/crm/deals/{id}/stage` (опционально)

**Описание:** Специальный эндпоинт для смены этапа (из канбана)

**Тело запроса:**
```json
{
  "stageId": "uuid"
}
```

**Ответ:** 200 OK + обновленная сделка

#### `DELETE /api/v1/workspaces/{workspaceId}/crm/deals/{id}`

**Ответ:** 204 No Content

---

## 4. Бизнес-логика и правила

### 4.1. При создании контакта
- Если передан `companyId`, автоматически добавить контакт в связь компания-контакты (`crm_company_contacts`)
- Установить `createdBy` и `updatedBy` равными текущему пользователю

### 4.2. При обновлении контакта
- Если изменился `companyId`:
  - Удалить старую связь (если была)
  - Создать новую связь
- Заполнить `updatedBy` и `updatedAt`

### 4.3. При удалении контакта
- Проверить наличие связанных сделок
- Если сделки есть — вернуть 409 Conflict с сообщением
- Удалить связи из `crm_company_contacts`
- Soft delete (пометить `deleted_at`)

### 4.4. При создании сделки
- Если передан `contactId`, проверить существование контакта
- Если передан `companyId`, проверить существование компании
- Установить статус `open`
- Вероятность взять из этапа, если не указана

### 4.5. При смене этапа сделки
- Проверить, что этап принадлежит воронке сделки
- Если новый этап `isFinal = true`:
  - Установить `status = 'won'`
  - Установить `actualCloseDate = now()`
- Если новый этап `isLost = true`:
  - Установить `status = 'lost'`
  - Установить `actualCloseDate = now()`
- Обновить `probability` из этапа (если в этапе указана)

### 4.6. При смене статуса сделки вручную
- Если `status = 'won'`:
  - Проверить, что текущий этап финальный (или разрешить)
  - Установить `actualCloseDate = now()`
- Если `status = 'lost'`:
  - Можно заполнить `lostReason`
  - Установить `actualCloseDate = now()`

---

## 5. Индексы БД

### Контакты
```sql
CREATE INDEX idx_contacts_workspace ON crm_contacts(workspace_id);
CREATE INDEX idx_contacts_company ON crm_contacts(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_contacts_owner ON crm_contacts(owner_id);
CREATE INDEX idx_contacts_name ON crm_contacts(last_name, first_name);
CREATE INDEX idx_contacts_tags ON crm_contacts USING GIN(tags);
CREATE INDEX idx_contacts_created ON crm_contacts(created_at);
```

### Компании
```sql
CREATE INDEX idx_companies_workspace ON crm_companies(workspace_id);
CREATE INDEX idx_companies_inn ON crm_companies(inn) WHERE inn IS NOT NULL;
CREATE INDEX idx_companies_owner ON crm_companies(owner_id);
CREATE INDEX idx_companies_tags ON crm_companies USING GIN(tags);
CREATE INDEX idx_companies_name ON crm_companies(name);
```

### Связи компания-контакты
```sql
CREATE INDEX idx_company_contacts_company ON crm_company_contacts(company_id);
CREATE INDEX idx_company_contacts_contact ON crm_company_contacts(contact_id);
```

### Телефоны и email
```sql
CREATE INDEX idx_contact_phones_contact ON crm_contact_phones(contact_id);
CREATE INDEX idx_contact_phones_number ON crm_contact_phones(number);
CREATE INDEX idx_contact_emails_contact ON crm_contact_emails(contact_id);
CREATE INDEX idx_contact_emails_address ON crm_contact_emails(address);
```

### Сделки
```sql
CREATE INDEX idx_deals_workspace ON crm_deals(workspace_id);
CREATE INDEX idx_deals_contact ON crm_deals(contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX idx_deals_company ON crm_deals(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_deals_pipeline ON crm_deals(pipeline_id);
CREATE INDEX idx_deals_stage ON crm_deals(stage_id);
CREATE INDEX idx_deals_owner ON crm_deals(owner_id);
CREATE INDEX idx_deals_status ON crm_deals(status);
CREATE INDEX idx_deals_expected_date ON crm_deals(expected_close_date);
CREATE INDEX idx_deals_created ON crm_deals(created_at);
```

### Воронки
```sql
CREATE INDEX idx_pipelines_workspace ON crm_pipelines(workspace_id);
CREATE INDEX idx_stages_pipeline ON crm_stages(pipeline_id);
```

---

## 6. Коды ответов

| Код | Описание | Когда использовать |
|-----|----------|-------------------|
| 200 | OK | Успешный GET, PATCH |
| 201 | Created | Успешный POST |
| 204 | No Content | Успешный DELETE |
| 400 | Bad Request | Ошибка валидации |
| 401 | Unauthorized | Нет токена или невалидный |
| 403 | Forbidden | Нет прав на операцию |
| 404 | Not Found | Сущность не найдена |
| 409 | Conflict | Конфликт (например, удаление с зависимостями) |
| 422 | Unprocessable Entity | Бизнес-правило нарушено |
| 500 | Internal Server Error | Ошибка сервера |

**Формат ошибки:**
```json
{
  "code": 400,
  "message": "Ошибка валидации",
  "details": [
    {
      "field": "email",
      "error": "неверный формат"
    }
  ]
}
```

---

## 7. Проверка workspace (для будущих микросервисов)

Чтобы в будущем легко выделить микросервисы, используйте middleware:

```
Запрос → API Gateway (проверяет workspace) → Микросервис (доверяет)
```

**Заголовки, которые приходят в микросервис:**
- `X-Workspace-ID` — ID workspace (проверен Gateway)
- `X-User-ID` — ID пользователя
- `X-User-Role` — роль пользователя

Микросервис использует эти заголовки для фильтрации данных, но не проверяет workspace самостоятельно.

---

## 8. Требования к производительности

- Время ответа API: < 200 мс для 95% запросов
- Пагинация: не более 100 записей за раз
- Поиск: использовать индексы, ограничить время выполнения
- Connection pool: настроить под ожидаемую нагрузку
- Soft delete: все удаления помечать `deleted_at`, не удалять физически

---

## 9. Заключение

Данная спецификация полностью соответствует вашим моделям и обеспечивает:

1. **Мультитенантность** через `workspace_id`
2. **Возможность перехода на микросервисы** (каждый домен изолирован)
3. **Все CRUD операции** для четырех сущностей
4. **Бизнес-логику** для связей и транзакций
5. **Производительность** через индексы
6. **Безопасность** через проверку workspace

Модели можно использовать как есть — они полностью покрывают требования первого этапа.
