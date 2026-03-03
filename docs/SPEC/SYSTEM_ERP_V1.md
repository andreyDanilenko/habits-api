## СИСТЕМНЫЙ АНАЛИЗ ПРОЕКТА ERP (АКТУАЛИЗИРОВАННЫЙ)

### На основе миграций (до 000021) и текущего состояния API (94 эндпоинта)

| Версия | 2.0 | Дата | Март 2026 |
|--------|-----|------|-----------|

---

## Содержание

1. **Общая архитектура системы**
2. **Модульная структура и лицензирование**
3. **Ядро системы (Core)**
4. **Модуль Habits (Привычки)**
5. **Модуль CRM**
6. **Модуль Projects**
7. **Модуль Notes (Заметки)**
8. **Shared-сущности (Currencies, Counterparties)**
9. **Активность пользователей (Recent Activity)**
10. **Административный модуль**
11. **Права доступа и ролевая модель**
12. **Мягкие связи и масштабируемость**
13. **Текущий статус реализации (по API)**
14. **План доработок (приоритеты)**
15. **План разработки по этапам**
16. **Критерии готовности**

---

## 1. Общая архитектура системы

### 1.1. Назначение
Корпоративная система управления (ERP) с модульной архитектурой, построенная на принципах мультитенантности и масштабируемости. Каждый модуль может быть включён/отключён независимо, что позволяет гибко настраивать систему под потребности конкретного бизнеса.

### 1.2. Ключевые принципы
- **Мультитенантность**: все таблицы содержат `workspace_id`, обеспечивая полную изоляцию данных между клиентами
- **Модульность**: справочник `modules` и таблица `workspace_modules` управляют доступом к функциональности
- **Core-модули**: `habits`, `crm`, `projects` – включены по умолчанию для всех workspace
- **Мягкие связи**: модули не имеют прямых FOREIGN KEY друг на друга, связи через отдельные таблицы (например, `project_entities`)
- **Мягкое удаление**: важные сущности имеют поле `deleted_at`
- **Аудитивность**: `habit_history` для отслеживания изменений, планируется общая таблица аудита

### 1.3. Состав модулей

| Код модуля | Название | Core | Статус | Эндпоинтов |
|------------|----------|------|--------|------------|
| `habits` | Привычки | ✅ Да | ✅ Полностью готов | 12 |
| `crm` | CRM | ✅ Да | ✅ Полностью готов | 23 |
| `projects` | Проекты | ✅ Да | ✅ Полностью готов | 8 |
| `notes` | Заметки | ❌ Нет | ✅ Полностью готов | 5 |
| `inventory` | Склад | ❌ Нет | 📅 В разработке | - |
| `finance` | Финансы | ❌ Нет | 📅 В разработке | - |
| `hr` | HR | ❌ Нет | 📅 В разработке | - |
| `tasks` | Задачи | ❌ Нет | 📅 План (Этап 5) | - |

**Общее количество эндпоинтов:** 94+ (из них 94 основных реализованы; число может расти по мере развития системы)

---

## 2. Модульная структура и лицензирование

### 2.1. Таблицы модулей

```sql
-- Справочник всех модулей системы
CREATE TABLE modules (
    id UUID PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,  -- 'habits', 'crm', 'projects'
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_core BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL
);

-- Какие модули включены в workspace
CREATE TABLE workspace_modules (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, trial, disabled
    activated_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP,
    settings JSONB,
    UNIQUE(workspace_id, module_id)
);

-- Лицензии пользователей на модули
CREATE TABLE user_module_licenses (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    scope VARCHAR(20) NOT NULL, -- all_workspaces, single_workspace
    workspace_id UUID, -- для single_workspace
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    source VARCHAR(20) NOT NULL DEFAULT 'purchase', -- purchase, admin_grant
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL
);
```

### 2.2. Бизнес-логика
- **Core-модули** (`is_core = true`) автоматически включаются для всех новых workspace (триггер `fn_workspace_enable_core_modules`)
- Существующие workspace получают core-модули через `INSERT ... ON CONFLICT`
- Пользователи могут иметь лицензии на модули:
  - `all_workspaces` – модуль доступен во всех workspace пользователя
  - `single_workspace` – модуль доступен только в указанном workspace
- Проверка доступа к модулю осуществляется middleware `RequireModule(moduleCode)`

---

## 3. Ядро системы (Core)

### 3.1. Пользователи (users)

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| email | VARCHAR(255) | Уникальный email |
| password | VARCHAR(255) | Хеш пароля |
| name | VARCHAR(100) | Имя пользователя |
| role | VARCHAR(20) | USER / ADMIN (глобальная роль) |
| avatar_url | TEXT | Ссылка на аватар |
| status | VARCHAR(20) | ACTIVE / BLOCKED |
| created_at | TIMESTAMP | Дата создания |
| updated_at | TIMESTAMP | Дата обновления |

### 3.2. Workspace (рабочие пространства)

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| name | VARCHAR(255) | Название компании |
| description | TEXT | Описание |
| color | VARCHAR(50) | Цвет для UI |
| owner_id | UUID | Владелец (ссылка на users) |
| created_at | TIMESTAMP | Дата создания |
| updated_at | TIMESTAMP | Дата обновления |

### 3.3. Связь пользователей с workspace

**Таблица:** `user_workspaces`
- `user_id`, `workspace_id` – составной ключ
- `role` – роль пользователя в данном workspace (OWNER, ADMIN, MEMBER, GUEST)

**Таблица:** `user_preferences`
- `user_id` – уникальная связь
- `current_workspace_id` – текущий выбранный workspace

### 3.4. API Workspace (12 эндпоинтов)

```
GET    /workspaces                    - список workspace пользователя
POST   /workspaces                    - создание нового workspace
GET    /workspaces/current             - получение текущего workspace
GET    /workspaces/me/module-licenses  - лицензии текущего пользователя
GET    /workspaces/:workspaceId        - получение конкретного workspace
PUT    /workspaces/:workspaceId        - обновление workspace
DELETE /workspaces/:workspaceId        - удаление workspace
GET    /workspaces/:workspaceId/members - список участников
POST   /workspaces/:workspaceId/switch - переключение текущего workspace
GET    /workspaces/:workspaceId/modules - список модулей workspace
POST   /workspaces/:workspaceId/modules - включение модуля
DELETE /workspaces/:workspaceId/modules/:moduleCode - отключение модуля
```

---

## 4. Модуль Habits (Привычки)

### 4.1. Назначение
Трекинг привычек и ведение дневника с поддержкой расписаний и версионирования.

### 4.2. Сущности

#### Habits (привычки)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| workspace_id | UUID | Принадлежность |
| user_id | UUID | Владелец |
| title | VARCHAR(255) | Название |
| description | TEXT | Описание |
| color | VARCHAR(50) | Цвет |
| icon | VARCHAR(100) | Иконка |
| target_days | INTEGER | Целевое количество дней |
| daily_goal | INTEGER | Цель на день |
| preferred_time | TIME | Предпочтительное время |
| category | VARCHAR(100) | Категория |
| schedule_type | VARCHAR(20) | recurring / one_time |
| recurring_days | INTEGER[] | Дни недели (0-6) |
| one_time_date | DATE | Дата для разовой привычки |
| is_active | BOOLEAN | Активна |
| created_at | TIMESTAMP | Дата создания |
| updated_at | TIMESTAMP | Дата обновления |

#### Habit Completions (выполнения)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| habit_id | UUID | Ссылка на привычку |
| user_id | UUID | Кто выполнил |
| date | DATE | Дата выполнения |
| notes | TEXT | Заметки |
| rating | INTEGER | Оценка (1-5) |
| time | TIME | Время выполнения |
| workspace_id | UUID | Принадлежность |
| created_at | TIMESTAMP | Дата создания |

#### Habit History (история изменений)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| habit_id | UUID | Ссылка на привычку |
| user_id | UUID | Кто изменил |
| action | VARCHAR(50) | CREATED, UPDATED, DELETED, COMPLETED |
| changes | JSONB | Старые/новые значения |
| metadata | JSONB | Доп. данные (IP, user agent) |
| created_at | TIMESTAMP | Дата |

#### Habit Versions (версии привычек)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| habit_id | UUID | Ссылка |
| user_id | UUID | Владелец |
| workspace_id | UUID | Принадлежность |
| ... | ... | Копия всех полей habit |
| valid_from | DATE | Дата начала действия |
| valid_to | DATE | Дата окончания |

#### Journal Entries (записи дневника)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| workspace_id | UUID | Принадлежность |
| user_id | UUID | Автор |
| description | TEXT | Содержание |
| mood | INTEGER | Настроение (1-5) |
| date | DATE | Дата записи |
| tags | TEXT[] | Теги |
| content_type | VARCHAR(20) | text / markdown |
| metadata | JSONB | Доп. данные |
| created_at | TIMESTAMP | Дата создания |
| updated_at | TIMESTAMP | Дата обновления |

### 4.3. API Habits (12 эндпоинтов)

```
GET    /habits                    - список привычек
POST   /habits                    - создание привычки
GET    /habits/:habitId            - получение привычки
PUT    /habits/:habitId            - обновление привычки
DELETE /habits/:habitId            - удаление привычки
POST   /habits/:habitId/complete   - отметить выполнение
POST   /habits/:habitId/toggle     - включить/выключить
GET    /habits/:habitId/stats      - статистика привычки
GET    /habits/completions         - выполнения за период
GET    /habits/calendar            - календарь выполнения
GET    /journal                    - записи дневника
POST   /journal                    - создать запись
GET    /journal/:entryId           - получить запись
PUT    /journal/:entryId           - обновить запись
DELETE /journal/:entryId           - удалить запись
```

### 4.4. Статус: ✅ ПОЛНОСТЬЮ ГОТОВ

---

## 5. Модуль CRM

### 5.1. Назначение
Управление взаимоотношениями с клиентами: контакты, компании, сделки, воронки продаж.

### 5.2. Сущности

#### Контакты (crm_contacts)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| workspace_id | UUID | Принадлежность |
| first_name | VARCHAR(100) | Имя |
| last_name | VARCHAR(100) | Фамилия |
| middle_name | VARCHAR(100) | Отчество |
| company_id | UUID | Ссылка на компанию |
| position | VARCHAR(200) | Должность |
| birthday | DATE | День рождения |
| tags | TEXT[] | Теги |
| owner_id | UUID | Ответственный |
| created_by | UUID | Создатель |
| updated_by | UUID | Редактор |
| custom_fields | JSONB | Динамические поля |
| deleted_at | TIMESTAMPTZ | Мягкое удаление |

**Связанные таблицы:**
- `crm_contact_phones` – телефоны (тип, номер, основной)
- `crm_contact_emails` – email (тип, адрес, основной)

#### Компании (crm_companies)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| workspace_id | UUID | Принадлежность |
| name | VARCHAR(255) | Название |
| inn | VARCHAR(12) | ИНН |
| kpp | VARCHAR(9) | КПП |
| ogrn | VARCHAR(15) | ОГРН |
| phone | VARCHAR(50) | Телефон |
| email | VARCHAR(255) | Email |
| website | VARCHAR(255) | Сайт |
| legal_address | JSONB | Юридический адрес |
| actual_address | JSONB | Фактический адрес |
| tags | TEXT[] | Теги |
| owner_id | UUID | Ответственный |
| deleted_at | TIMESTAMPTZ | Мягкое удаление |

**Связи:** `crm_company_contacts` – связь компаний с контактами

#### Воронки и этапы
**crm_pipelines**
- id, workspace_id, name, is_default, created_by, created_at

**crm_stages**
- id, pipeline_id, name, order_index, color, probability, is_final, is_lost, created_at

**Ограничения:**
- В одной воронке только один финальный этап (`is_final`)
- В одной воронке только один этап проигрыша (`is_lost`)

#### Сделки (crm_deals)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| workspace_id | UUID | Принадлежность |
| name | VARCHAR(500) | Название |
| contact_id | UUID | Ссылка на контакт |
| company_id | UUID | Ссылка на компанию |
| budget | DECIMAL(15,2) | Сумма |
| currency | VARCHAR(3) | Валюта |
| pipeline_id | UUID | Воронка |
| stage_id | UUID | Текущий этап |
| expected_close_date | DATE | Плановая дата |
| actual_close_date | DATE | Фактическая дата |
| status | VARCHAR(20) | open / won / lost |
| lost_reason | TEXT | Причина проигрыша |
| description | TEXT | Описание |
| source | VARCHAR(100) | Источник |
| probability | INTEGER | Вероятность |
| tags | TEXT[] | Теги |
| owner_id | UUID | Ответственный |
| deleted_at | TIMESTAMPTZ | Мягкое удаление |

#### Активности (crm_activities)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| workspace_id | UUID | Принадлежность |
| type | VARCHAR(50) | note / call / email / deal_stage_changed |
| entity_type | VARCHAR(20) | contact / company / deal |
| entity_id | UUID | ID сущности |
| title | VARCHAR(500) | Заголовок |
| description | TEXT | Описание |
| metadata | JSONB | Метаданные |
| is_important | BOOLEAN | Флаг важности |
| created_by | UUID | Автор |
| created_by_name | VARCHAR(255) | Денормализованное имя |
| created_by_avatar | VARCHAR(500) | Аватар |
| is_editable | BOOLEAN | Можно редактировать |
| is_deletable | BOOLEAN | Можно удалять |
| deleted_at | TIMESTAMPTZ | Мягкое удаление |

**Связанные таблицы:**
- `crm_activity_files` – файлы, прикреплённые к активности
- `crm_activity_reminders` – напоминания из заметок

### 5.3. API CRM (23 эндпоинта, все реализованы)

**Контакты (6)**
```
GET    /contacts        ✅
GET    /contacts/:id    ✅
POST   /contacts        ✅
PUT    /contacts/:id    ✅
DELETE /contacts/:id    ✅
```

**Компании (7)**
```
GET    /companies                     ✅
GET    /companies/:id                 ✅
POST   /companies                     ✅
PUT    /companies/:id                 ✅
DELETE /companies/:id                 ✅
POST   /companies/:id/contacts/:contactId ✅
DELETE /companies/:id/contacts/:contactId ✅
```

**Воронки и этапы (полный CRUD, 11 эндпоинтов)**
```
GET    /pipelines                     ✅
POST   /pipelines                     ✅
GET    /pipelines/:id                 ✅
PUT    /pipelines/:id                 ✅
DELETE /pipelines/:id                 ✅
GET    /pipelines/:pipelineId/stages              ✅
GET    /pipelines/:pipelineId/stages/:id          ✅
POST   /pipelines/:pipelineId/stages              ✅
PUT    /pipelines/:pipelineId/stages/:id          ✅
DELETE /pipelines/:pipelineId/stages/:id          ✅
POST   /pipelines/:pipelineId/stages/reorder      ✅
```

**Сделки (5)**
```
GET    /deals          ✅
GET    /deals/:id      ✅
POST   /deals          ✅
PUT    /deals/:id      ✅
DELETE /deals/:id      ✅
```

**Активности (5)**
```
GET    /activities             ✅
POST   /activities             ✅
GET    /activities/:id         ✅
PUT    /activities/:id         ✅
DELETE /activities/:id         ✅
POST   /activities/:id/important ✅
```

### 5.4. Статус: ⚠️ ЧАСТИЧНО (78%, требуется 5-8 новых эндпоинтов)

---

## 6. Модуль Projects

### 6.1. Назначение
Сквозная группировка любых сущностей системы в проекты (контексты).

### 6.2. Сущности

#### Проекты (projects)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| workspace_id | UUID | Принадлежность |
| name | VARCHAR(255) | Название |
| description | TEXT | Описание |
| created_at | TIMESTAMPTZ | Дата создания |
| updated_at | TIMESTAMPTZ | Дата обновления |

#### Связи проектов (project_entities)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| project_id | UUID | Ссылка на проект |
| entity_type | VARCHAR(50) | 'crm_contact', 'crm_company', 'crm_deal', 'habit', 'note' |
| entity_id | UUID | ID сущности |
| created_at | TIMESTAMPTZ | Дата привязки |

**Уникальность:** один проект не может содержать дубликат сущности

### 6.3. API Projects (8 эндпоинтов)

```
GET    /projects                               ✅
POST   /projects                               ✅
GET    /projects/:projectId                     ✅
PUT    /projects/:projectId                     ✅
DELETE /projects/:projectId                     ✅
GET    /projects/:projectId/entities            ✅
POST   /projects/:projectId/entities            ✅
DELETE /projects/:projectId/entities/:entityType/:entityId ✅
GET    /entities/:entityType/:entityId/projects ✅
```

### 6.4. Статус: ✅ ПОЛНОСТЬЮ ГОТОВ

---

## 7. Модуль Notes (Заметки)

### 7.1. Сущность
**Таблица:** `notes`
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| workspace_id | UUID | Принадлежность |
| user_id | UUID | Автор |
| title | VARCHAR(500) | Заголовок |
| content | TEXT | Содержание |
| created_at | TIMESTAMP | Дата создания |
| updated_at | TIMESTAMP | Дата обновления |

### 7.2. API Notes (5 эндпоинтов)

```
GET    /notes          ✅
POST   /notes          ✅
GET    /notes/:noteId  ✅
PUT    /notes/:noteId  ✅
DELETE /notes/:noteId  ✅
```

### 7.3. Статус: ✅ ПОЛНОСТЬЮ ГОТОВ

---

## 8. Shared-сущности (общие для модулей)

### 8.1. Валюты (currencies)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| workspace_id | UUID | Принадлежность |
| code | VARCHAR(10) | Код валюты (RUB, USD) |
| name | VARCHAR(100) | Название |
| symbol | VARCHAR(10) | Символ |
| created_at | TIMESTAMP | Дата создания |
| updated_at | TIMESTAMP | Дата обновления |

### 8.2. Контрагенты (counterparties)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| workspace_id | UUID | Принадлежность |
| name | VARCHAR(255) | Наименование |
| type | VARCHAR(20) | client / supplier / both |
| email | VARCHAR(255) | Email |
| phone | VARCHAR(50) | Телефон |
| comment | TEXT | Комментарий |
| created_at | TIMESTAMP | Дата создания |
| updated_at | TIMESTAMP | Дата обновления |

### 8.3. API Master Data (10 эндпоинтов)

**Валюты:**
```
GET    /currencies                 ✅
POST   /currencies                 ✅
GET    /currencies/:currencyId     ✅
PUT    /currencies/:currencyId     ✅
DELETE /currencies/:currencyId     ✅
```

**Контрагенты:**
```
GET    /counterparties                 ✅
POST   /counterparties                 ✅
GET    /counterparties/:counterpartyId ✅
PUT    /counterparties/:counterpartyId ✅
DELETE /counterparties/:counterpartyId ✅
```

### 8.4. Статус: ✅ ПОЛНОСТЬЮ ГОТОВ

---

## 9. Активность пользователей (Recent Activity)

### 9.1. Сущность
**Таблица:** `activities`
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Первичный ключ |
| user_id | UUID | Автор |
| workspace_id | UUID | Принадлежность |
| type | VARCHAR(50) | HABIT_CREATED, HABIT_COMPLETED |
| entity_type | VARCHAR(50) | habit, completion, workspace |
| entity_id | UUID | ID сущности |
| title | VARCHAR(255) | Текст для отображения |
| emoji | VARCHAR(10) | Эмодзи |
| created_at | TIMESTAMP | Дата создания |

### 9.2. Назначение
- Виджет "Недавняя активность" в интерфейсе
- Лента событий пользователя
- Не зависит от CRM-активностей (отдельная таблица для общих событий)

---

## 10. Административный модуль

### 10.1. API Admin (4 эндпоинта)

```
GET    /admin/workspaces           - список всех workspace
GET    /admin/users                 - список всех пользователей
DELETE /admin/users/:id             - удаление пользователя
POST   /admin/users/:id/licenses    - выдача лицензии пользователю
```

### 10.2. Статус: ✅ ПОЛНОСТЬЮ ГОТОВ

---

## 11. Права доступа и ролевая модель

### 11.1. Роли в workspace (из user_workspaces.role)
| Роль | Уровень | Права |
|------|---------|-------|
| **OWNER** | 100 | Полный доступ, управление workspace, удаление |
| **ADMIN** | 80 | Управление настройками, пользователями, модулями |
| **MEMBER** | 50 | Работа с данными (создание, редактирование) |
| **GUEST** | 10 | Только просмотр (read-only) |

### 11.2. Глобальные роли пользователя (users.role)
- `USER` – обычный пользователь
- `ADMIN` – глобальный администратор системы (доступ к /admin/*)

### 11.3. Проверка доступа
- **К workspace**: middleware `requireWorkspaceAccess`
- **К модулям**: middleware `RequireModule(moduleCode)`
- **К ролям**: планируется middleware `RequireWorkspaceRole(OWNER, ADMIN)`

---

## 12. Мягкие связи и масштабируемость

### 12.1. Принципы
1. **Мультитенантность**: все таблицы содержат `workspace_id`
2. **Отсутствие прямых FK между модулями**: связи через отдельные таблицы (`project_entities`)
3. **Core-модули** – всегда включены, остальные – опционально
4. **Мягкое удаление**: поля `deleted_at` для важных сущностей

### 12.2. Преимущества
- Возможность вынести модули в отдельные БД/микросервисы
- Независимое развитие модулей
- Гибкая система лицензирования

---

## 13. Текущий статус реализации (по API)

| Модуль | Всего эндпоинтов | Реализовано | % | Статус |
|--------|------------------|-------------|---|--------|
| Auth | 5 | 5 | 100% | ✅ Готов |
| Workspace | 12 | 12 | 100% | ✅ Готов |
| CRM | 23 | 23 | 100% | ✅ Готов |
| Projects | 8 | 8 | 100% | ✅ Готов |
| Habits | 12 | 12 | 100% | ✅ Готов |
| Notes | 5 | 5 | 100% | ✅ Готов |
| Master Data | 10 | 10 | 100% | ✅ Готов |
| Admin | 4 | 4 | 100% | ✅ Готов |
| Logger | 2 | 2 | 100% | ✅ Готов |
| Swagger/Health | 3 | 3 | 100% | ✅ Готов |
| **ИТОГО:** | **94+** | **94** | **≈100%** | |

---

## 14. План доработок (приоритеты)

### 🔴 Высокий приоритет (CRM Pipelines)
Все эндпоинты для воронок и этапов реализованы. Блок переносится в разряд истории; для будущих задач по CRM остаются только улучшения ниже.

### 🟡 Средний приоритет
- [ ] Middleware для проверки ролей (OWNER/ADMIN/MEMBER/GUEST)
- [ ] Приглашения пользователей в workspace
- [ ] Массовые операции для контактов (batch delete/update)
- [ ] Массовое перемещение сделок (batch move)

### 🟢 Низкий приоритет
- [ ] Автоматические активности (триггеры)
- [ ] Расширенная валидация (email, ИНН, КПП)
- [ ] Полнотекстовый поиск

---

## 15. План разработки по этапам

| Этап | Название | Модули | Статус |
|------|----------|--------|--------|
| **Этап 1** | Ядро системы | users, workspaces, auth | ✅ Готов |
| **Этап 2** | Базовые модули | habits, notes, master data | ✅ Готов |
| **Этап 3** | CRM (ядро) | contacts, companies, deals | ✅ Готов |
| **Этап 4** | Projects | projects, project_entities | ✅ Готов |
| **Этап 5** | Tasks | задачи и напоминания | 📅 План |
| **Этап 6** | Финансы и Склад | finance, inventory | 📅 План |
| **Этап 7** | HR и Отчёты | hr, reports, automation | 📅 План |

---

## 16. Критерии готовности

### По модулям

#### CRM (для завершения этапа 3)
- [x] Полный CRUD для воронок (все эндпоинты)
- [ ] Автоматические активности при создании/изменении сущностей

#### Workspace
- [ ] Приглашения пользователей по email
- [ ] Middleware для проверки ролей
- [ ] Изменение ролей участников

#### Общие
- [ ] Row Level Security (RLS) для всех таблиц
- [ ] Индексы для всех частых запросов
- [ ] Swagger-документация для всех эндпоинтов

---

## Заключение

Система находится в высокой степени готовности (≈100% API реализовано по текущей спецификации). **Все основные модули, включая CRM, завершены**, далее фокус смещается на новые модули (Tasks, Finance, HR) и продвинутые возможности (автоматизации, массовые операции, отчёты).
