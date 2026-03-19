# Backend: Резюме для собеседования

**Цель:** Структурированное описание backend-проекта для подготовки к собеседованию и самопроверки.

---

## 1. Краткий elevator pitch (30 сек)

«Я разрабатывал backend ERP-системы на Go: мультитенантная архитектура с workspace, модули CRM, Habits, Tasks, Projects. Стек: Gin, PostgreSQL, JWT, Casbin RBAC. Слоистая архитектура (Handler → Service → Repository), DI-контейнер, миграции через golang-migrate. Реализовал приглашения, лицензирование модулей, realtime через Redis.»

---

## 2. Технологический стек

| Компонент | Технология | Зачем |
|-----------|------------|-------|
| Язык | Go 1.25+ | Производительность, простота, статическая типизация |
| Web-фреймворк | Gin | Быстрый роутинг, middleware, JSON |
| БД | PostgreSQL | ACID, JSONB, сложные запросы |
| Доступ к БД | database/sql + lib/pq | Нативный SQL, без ORM для бизнес-логики |
| ORM | GORM | Только для Casbin-адаптера |
| Миграции | golang-migrate | Версионирование схемы |
| Auth | JWT (golang-jwt/jwt/v5) | Access + refresh токены |
| Authorization | Casbin | RBAC с доменами (workspace) |
| Валидация | go-playground/validator | Структуры, теги |
| API Docs | Swagger (swaggo) | Генерация из аннотаций |
| Realtime | Redis Pub/Sub | События (опционально) |
| Email | SMTP / Noop | Верификация, приглашения |

---

## 3. Архитектура

### 3.1. Слоистая архитектура

```
Handler → Service → Repository → Database
```

| Слой | Ответственность | Пример |
|------|-----------------|--------|
| **Handler** | HTTP, парсинг, валидация, ответы | `CrmHandler.CreateDeal()` |
| **Service** | Бизнес-логика, оркестрация | `CrmService.CreateDeal()` |
| **Repository** | SQL, работа с БД | `CrmRepository.CreateDeal()` |

**Правило:** Handler не знает о БД, Service не знает о HTTP, Repository не содержит бизнес-логики.

### 3.2. Dependency Injection

- Центральный контейнер в `internal/di/container.go`
- Порядок: Repositories → Services → Handlers
- Контейнер регистрирует маршруты и передаёт зависимости

### 3.3. Цепочка middleware (защищённые маршруты)

1. **CORS**
2. **RequestLogger** — логирование запросов
3. **GinAuthMiddleware** — JWT из cookie или `Authorization`
4. **WorkspacePathMiddleware** — проверка доступа к workspace
5. **ModuleLicenseMiddleware** — модуль включён, лицензия есть
6. **PermissionMiddleware** — Casbin RBAC

### 3.4. Доменная структура

Каждый домен (auth, workspace, crm, habits, tasks и т.д.) имеет:

- `handler/` — HTTP-обработчики
- `service/` — бизнес-логика
- `repository/` — доступ к данным

---

## 4. Ключевые фичи и модули

| Модуль | Назначение | Ключевые таблицы |
|--------|------------|------------------|
| **Auth** | Регистрация, логин, refresh, верификация email | users, registration_tokens |
| **Workspaces** | Мультитенантность, участники | workspaces, user_workspaces |
| **Permissions** | RBAC, роли, наследование | permission_catalog, workspace_roles, user_role_assignments |
| **CRM** | Контакты, компании, сделки, воронки | crm_contacts, crm_deals, crm_pipelines |
| **Projects** | Группировка сущностей | projects, project_entities |
| **Tasks** | Задачи, комментарии | tasks, task_comments |
| **Habits** | Привычки, выполнения | habits, habit_completions |
| **Notes / Journal** | Заметки, дневник | notes, journal_entries |
| **Invitations** | Приглашения в workspace | invitations |
| **Notifications** | Уведомления | notifications |
| **Admin** | Управление workspace, пользователями, модулями | — |

---

## 5. Мультитенантность и workspace

- **Workspace** — рабочее пространство (организация).
- Все данные изолированы по `workspace_id`.
- Один пользователь может быть в нескольких workspace.
- Роли в workspace: OWNER, ADMIN, MEMBER, GUEST.

### Цепочка проверки доступа

```
GET /api/v1/workspaces/{id}/crm/deals

1. JWT → user_id
2. WorkspacePathMiddleware → HasAccess(user_id, workspace_id)
3. ModuleLicenseMiddleware → CRM включён, лицензия есть
4. PermissionMiddleware → Casbin: crm:deal:read
```

---

## 6. Лицензирование модулей

| Тип | Примеры | Поведение |
|-----|---------|-----------|
| **Core** | habits, crm, projects, tasks | Бесплатно, включаются при создании workspace |
| **Не-core** | notes, inventory, finance | Требуется лицензия или триал |

- `workspace_modules` — какие модули включены в workspace.
- `user_module_licenses` — право пользователя включить модуль.
- Scope: `all_workspaces` или `single_workspace`.

---

## 7. Casbin и RBAC

- **Модель:** `sub, dom, obj, act` (пользователь, workspace, ресурс, действие).
- **Политики** в БД через GORM-адаптер.
- **Формат права:** `crm:deal:create`, `habits:habit:complete`.
- **Роли:** системные (OWNER, ADMIN, MEMBER, GUEST) + кастомные.
- **Наследование:** role_inheritance (child ← parent).

---

## 8. API

- **Base path:** `/api/v1`
- **REST:** GET, POST, PUT/PATCH, DELETE
- **Workspace в пути:** `/workspaces/:workspaceId/crm/deals`
- **Swagger:** `/swagger/*`

---

## 9. Миграции

- **Инкрементальные:** 000001–000032 + constraints
- **Clean baseline:** 001–027 по сущностям для чистой БД
- **Правило:** при изменении миграций обновлять clean_baseline

---

## 10. Самопроверка: что я знаю / что подтянуть

### ✅ Скорее всего знаешь (раз делал)

| Тема | Уровень | Вопрос для самопроверки |
|------|---------|-------------------------|
| Go: структуры, интерфейсы | Базовый | Как устроен DI-контейнер? |
| Gin: роутинг, middleware | Базовый | Что делает WorkspacePathMiddleware? |
| PostgreSQL: CRUD, JOIN | Базовый | Как связаны habits и habit_completions? |
| JWT | Базовый | Где хранится токен? Cookie vs Header |
| REST API | Базовый | Как организованы маршруты? |

### ⚠️ Стоит повторить

| Тема | Вопрос для самопроверки |
|------|-------------------------|
| Casbin | Как устроена модель? Что такое sub, dom, obj, act? |
| Транзакции | Где в коде используются tx.Begin/Commit? |
| Индексы | Какие индексы на crm_deals и зачем? |
| Мягкое удаление | Как реализовано deleted_at? |
| Rate limiting | Как ограничивается логин? |

### ❓ Возможные пробелы

| Тема | Что изучить |
|------|-------------|
| Конкурентность в Go | goroutines, channels, sync |
| Тестирование | unit, integration, mock |
| Профилирование | pprof, трассировка |
| Безопасность | SQL injection, XSS, CSRF |
| Деплой | Docker, env, секреты |

---

## 11. Вопросы, которые могут задать

### Архитектура

- «Опишите слои приложения и как они взаимодействуют»
- «Почему выбрали Handler → Service → Repository?»
- «Как устроен DI? Зачем он нужен?»

### БД

- «Как устроена мультитенантность?»
- «Почему в CRM нет FK на users/workspaces?»
- «Как работают миграции? Что такое clean baseline?»

### Безопасность

- «Как реализована аутентификация?»
- «Как работает авторизация? Что такое Casbin?»
- «Как защищены эндпоинты от несанкционированного доступа?»

### Масштабирование

- «Как можно вынести CRM в отдельный микросервис?»
- «Как добавить новый модуль?»
- «Что делать, если миграций стало слишком много?»

---

## 12. Связанные документы

- [01_OVERVIEW_AND_STRUCTURE.md](./guides/developer-guide/01_OVERVIEW_AND_STRUCTURE.md) — обзор и структура
- [02_DATABASE_AND_TABLES.md](./guides/developer-guide/02_DATABASE_AND_TABLES.md) — схема БД
- [03_MODULES_INTERACTION.md](./guides/developer-guide/03_MODULES_INTERACTION.md) — взаимодействие модулей
- [SQL_MIGRATIONS_SCALING_GUIDE.md](./guides/SQL_MIGRATIONS_SCALING_GUIDE.md) — масштабирование миграций
