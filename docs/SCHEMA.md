# Схема базы данных ERP

Визуализация структуры таблиц и связей. SQL-миграция: `NEW_MIGRATE.sql`.

---

## 1. Обзор

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              CORE (ядро)                                         │
├─────────────────────────────────────────────────────────────────────────────────┤
│  users ◄── registration_tokens                                                    │
│    │                                                                              │
│    └── workspaces ◄── user_workspaces ◄── user_preferences                        │
│           │                  │                                                     │
│           ├── invitations   ├── modules ◄── workspace_modules ◄── user_module_licenses
│           │                  │                                                     │
│           └── permission_catalog ◄── workspace_roles ◄── user_role_assignments    │
│                                    │              user_permissions                 │
│                                    └── role_inheritance                            │
└─────────────────────────────────────────────────────────────────────────────────┘
                                         │
        ┌────────────────────────────────┼────────────────────────────────┐
        │                                │                                │
        ▼                                ▼                                ▼
┌───────────────┐              ┌───────────────┐              ┌───────────────┐
│   HABITS      │              │     CRM       │              │   DOMAINS     │
├───────────────┤              ├───────────────┤              ├───────────────┤
│ habits        │              │ crm_contacts  │              │ notes         │
│ habit_completions│           │ crm_companies │              │ projects      │
│ habit_history │              │ crm_deals     │              │ project_entities
│ habit_versions│              │ crm_activities│              │ activities    │
│ journal_entries│             │ crm_pipelines │              │ currencies    │
└───────────────┘              │ crm_stages    │              │ counterparties│
                               └───────────────┘              └───────────────┘
        │                                │                                │
        └────────────────────────────────┘                                │
                         │                                                │
                         ▼                                                ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│  request_logs (observability)                                                    │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Таблицы по группам

### 2.1 Логирование
| Таблица | Назначение |
|---------|------------|
| `request_logs` | Логи HTTP-запросов (timestamp, status, duration, path) |

### 2.2 Аутентификация
| Таблица | Назначение |
|---------|------------|
| `users` | Пользователи (email, password, role, status) |
| `registration_tokens` | Токены подтверждения регистрации (invite_token — приглашение) |

### 2.3 Workspace (мультиарендность)
| Таблица | Назначение |
|---------|------------|
| `workspaces` | Воркспейсы (owner_id → users) |
| `user_workspaces` | Связь пользователь ↔ воркспейс (роль) |
| `user_preferences` | Текущий выбранный workspace |

### 2.4 Модули и лицензии
| Таблица | Назначение |
|---------|------------|
| `modules` | Справочник модулей (habits, crm, notes, projects...) |
| `workspace_modules` | Какие модули включены в workspace |
| `user_module_licenses` | Лицензия пользователя (all_workspaces / single_workspace) |

### 2.5 Привычки / Journal
| Таблица | Назначение |
|---------|------------|
| `habits` | Привычки (schedule_type: recurring / one_time) |
| `habit_completions` | Выполнения (habit_id, date, user_id) |
| `habit_history` | История изменений (action, changes JSONB) |
| `habit_versions` | Версионность (valid_from, valid_to) |
| `journal_entries` | Записи дневника (mood, tags, content_type) |

### 2.6 Заметки
| Таблица | Назначение |
|---------|------------|
| `notes` | Простые заметки (workspace_id, user_id) |

### 2.7 CRM
| Таблица | Назначение |
|---------|------------|
| `crm_contacts` | Контакты (phones, emails — отдельные таблицы) |
| `crm_contact_phones` | Телефоны контакта |
| `crm_contact_emails` | Email контакта |
| `crm_companies` | Компании (ИНН, КПП, ОГРН, адреса) |
| `crm_company_contacts` | Связь компания ↔ контакт |
| `crm_pipelines` | Воронки продаж |
| `crm_stages` | Этапы воронки |
| `crm_deals` | Сделки (pipeline, stage, contact, company) |
| `crm_activities` | Активности по сделкам/контактам |
| `crm_activity_files` | Файлы к активности |
| `crm_activity_reminders` | Напоминания |

### 2.8 Проекты
| Таблица | Назначение |
|---------|------------|
| `projects` | Проекты (группировка сущностей) |
| `project_entities` | Связь проект ↔ entity (entity_type, entity_id) |

### 2.9 Активность / Shared
| Таблица | Назначение |
|---------|------------|
| `activities` | Общая лента (type, entity_type, entity_id) |
| `currencies` | Справочник валют (workspace) |
| `counterparties` | Контрагенты (client/supplier/both) |

### 2.10 Приглашения и права
| Таблица | Назначение |
|---------|------------|
| `invitations` | Приглашения в workspace (token, system_role) |
| `permission_catalog` | Каталог прав (module_code, entity_type, action) |
| `workspace_roles` | Роли (OWNER, ADMIN, MEMBER, GUEST) |
| `user_role_assignments` | Назначение ролей пользователям |
| `user_permissions` | Прямые права (permission_id) |
| `role_inheritance` | Наследование ролей |

---

## 3. Ключевые связи

| От | К | Тип |
|----|---|-----|
| workspaces | users | owner_id |
| user_workspaces | users, workspaces | М:N |
| habits | users, workspaces | user_id, workspace_id |
| habit_completions | habits, users | habit_id, user_id |
| journal_entries | workspaces, users | workspace_id, user_id |
| notes | workspaces, users | workspace_id, user_id |
| crm_* | workspaces | workspace_id (без FK) |
| projects | workspaces | workspace_id (без FK) |
| activities | users, workspaces | user_id, workspace_id |

---

## 4. Триггеры

| Триггер | Таблица | Действие |
|---------|---------|----------|
| `update_workspaces_updated_at` | workspaces | Обновление updated_at |
| `update_habits_updated_at` | habits | Обновление updated_at |
| `tr_workspace_enable_core_modules` | workspaces | AFTER INSERT → включение core-модулей |
| `tr_create_system_roles` | workspaces | AFTER INSERT → создание OWNER, ADMIN, MEMBER, GUEST |

---

## 5. Индексы (ключевые)

- `users`: email, status
- `workspaces`: owner_id, created_at
- `habits`: user_id, workspace_id, schedule_type, is_active, recurring_days (GIN)
- `habit_completions`: habit_id, user_id, date, (habit_id, user_id, date)
- `crm_contacts`: workspace_id, company_id, owner_id, deleted_at
- `crm_deals`: workspace_id, pipeline_id, stage_id, stage_id, deleted_at
- `crm_activities`: (workspace_id, entity_type, entity_id, created_at DESC)
- `activities`: (user_id, workspace_id, created_at DESC)
