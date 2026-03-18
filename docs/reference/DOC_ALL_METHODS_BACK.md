# Документация по методам бэкенда и плану разбиения на сервисы

**Версия:** 1.0  
**Дата:** Март 2026

---

## Содержание

1. [API эндпоинты (HTTP)](#1-api-эндпоинты-http)
2. [Сервисы (Service)](#2-сервисы-service)
3. [Репозитории (Repository)](#3-репозитории-repository)
4. [Casbin: как работает и инструменты](#4-casbin-как-работает-и-инструменты)
5. [План разбиения на сервисы и Docker](#5-план-разбиения-на-сервисы-и-docker)

---

## 1. API эндпоинты (HTTP)

Базовый префикс: `/api/v1`.  
Защищённые маршруты проходят цепочку: `GinAuthMiddleware` → `WorkspacePathMiddleware` → `ModuleLicenseMiddleware` → `PermissionMiddleware` (где применимо).

### 1.1. Health

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/api/v1/health` | HealthCheck | Проверка живости сервиса |

### 1.2. Auth (публичные)

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| POST | `/api/v1/auth/login` | Auth.Login | Вход по email/password |
| POST | `/api/v1/auth/register` | Auth.Register | Регистрация |
| POST | `/api/v1/auth/logout` | Auth.Logout | Выход |
| POST | `/api/v1/auth/refresh` | Auth.Refresh | Обновление токена |

### 1.3. Auth (защищённые)

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/api/v1/auth/me` | Auth.Me | Текущий пользователь (профиль) |

### 1.4. Me (права текущего пользователя)

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/api/v1/me/permissions` | Permission.GetMyPermissions | Эффективные права в workspace (query: `workspaceId`) |

### 1.5. Workspaces

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/api/v1/workspaces` | Workspace.List | Список workspace пользователя |
| POST | `/api/v1/workspaces` | Workspace.Create | Создание workspace |
| GET | `/api/v1/workspaces/current` | Workspace.GetCurrent | Текущий выбранный workspace |
| GET | `/api/v1/workspaces/me/module-licenses` | Workspace.GetMyLicenses | Лицензии пользователя на модули |
| GET | `/api/v1/workspaces/:workspaceId` | Workspace.Get | Один workspace |
| PUT | `/api/v1/workspaces/:workspaceId` | Workspace.Update | Обновление workspace |
| DELETE | `/api/v1/workspaces/:workspaceId` | Workspace.Delete | Удаление workspace |
| GET | `/api/v1/workspaces/:workspaceId/members` | Workspace.GetMembers | Участники workspace |
| POST | `/api/v1/workspaces/:workspaceId/switch` | Workspace.Switch | Переключение текущего workspace |
| GET | `/api/v1/workspaces/:workspaceId/modules` | Workspace.GetModules | Модули workspace |
| POST | `/api/v1/workspaces/:workspaceId/modules` | Workspace.EnableModule | Включить модуль |
| DELETE | `/api/v1/workspaces/:workspaceId/modules/:moduleCode` | Workspace.DisableModule | Выключить модуль |

### 1.6. Permissions (в контексте workspace)

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/api/v1/workspaces/:workspaceId/permissions/catalog` | Permission.GetCatalog | Каталог прав |
| GET | `/api/v1/workspaces/:workspaceId/roles` | Permission.ListRoles | Список ролей |
| POST | `/api/v1/workspaces/:workspaceId/roles` | Permission.CreateRole | Создать роль |
| GET | `/api/v1/workspaces/:workspaceId/roles/:roleId` | Permission.GetRole | Одна роль |
| PUT | `/api/v1/workspaces/:workspaceId/roles/:roleId` | Permission.UpdateRole | Обновить роль |
| DELETE | `/api/v1/workspaces/:workspaceId/roles/:roleId` | Permission.DeleteRole | Удалить роль |
| GET | `/api/v1/workspaces/:workspaceId/roles/:roleId/permissions` | Permission.GetRolePermissions | Права роли |
| POST | `/api/v1/workspaces/:workspaceId/roles/:roleId/inherit/:parentRoleId` | Permission.AddRoleInheritance | Добавить наследование (roleId наследует parentRoleId) |
| DELETE | `/api/v1/workspaces/:workspaceId/roles/:roleId/inherit/:parentRoleId` | Permission.RemoveRoleInheritance | Удалить наследование |
| GET | `/api/v1/workspaces/:workspaceId/users/:userId/roles` | Permission.GetUserRoles | Роли пользователя |
| POST | `/api/v1/workspaces/:workspaceId/users/:userId/roles/:roleId` | Permission.AssignRole | Назначить роль |
| DELETE | `/api/v1/workspaces/:workspaceId/users/:userId/roles/:roleId` | Permission.RemoveRole | Снять роль |
| GET | `/api/v1/workspaces/:workspaceId/users/:userId/permissions` | Permission.GetUserPermissions | Индивидуальные права |
| POST | `/api/v1/workspaces/:workspaceId/users/:userId/permissions` | Permission.GrantPermission | Выдать право |
| DELETE | `/api/v1/workspaces/:workspaceId/users/:userId/permissions/:permissionId` | Permission.RevokePermission | Отозвать право |

### 1.7. Master (currencies, counterparties)

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/api/v1/workspaces/:workspaceId/currencies` | Master.ListCurrencies | Список валют |
| POST | `/api/v1/workspaces/:workspaceId/currencies` | Master.CreateCurrency | Создать валюту |
| GET | `/api/v1/workspaces/:workspaceId/currencies/:currencyId` | Master.GetCurrency | Одна валюта |
| PUT | `/api/v1/workspaces/:workspaceId/currencies/:currencyId` | Master.UpdateCurrency | Обновить валюту |
| DELETE | `/api/v1/workspaces/:workspaceId/currencies/:currencyId` | Master.DeleteCurrency | Удалить валюту |
| GET | `/api/v1/workspaces/:workspaceId/counterparties` | Master.ListCounterparties | Список контрагентов |
| POST | `/api/v1/workspaces/:workspaceId/counterparties` | Master.CreateCounterparty | Создать контрагента |
| GET | `/api/v1/workspaces/:workspaceId/counterparties/:counterpartyId` | Master.GetCounterparty | Один контрагент |
| PUT | `/api/v1/workspaces/:workspaceId/counterparties/:counterpartyId` | Master.UpdateCounterparty | Обновить контрагента |
| DELETE | `/api/v1/workspaces/:workspaceId/counterparties/:counterpartyId` | Master.DeleteCounterparty | Удалить контрагента |

### 1.8. CRM

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/api/v1/workspaces/:workspaceId/contacts` | Crm.ContactList | Список контактов |
| GET | `/api/v1/workspaces/:workspaceId/contacts/:id` | Crm.ContactGet | Один контакт |
| POST | `/api/v1/workspaces/:workspaceId/contacts` | Crm.ContactCreate | Создать контакт |
| PUT | `/api/v1/workspaces/:workspaceId/contacts/:id` | Crm.ContactUpdate | Обновить контакт |
| DELETE | `/api/v1/workspaces/:workspaceId/contacts/:id` | Crm.ContactDelete | Удалить контакт |
| GET | `/api/v1/workspaces/:workspaceId/companies` | Crm.CompanyList | Список компаний |
| GET | `/api/v1/workspaces/:workspaceId/companies/:id` | Crm.CompanyGet | Одна компания |
| POST | `/api/v1/workspaces/:workspaceId/companies` | Crm.CompanyCreate | Создать компанию |
| PUT | `/api/v1/workspaces/:workspaceId/companies/:id` | Crm.CompanyUpdate | Обновить компанию |
| DELETE | `/api/v1/workspaces/:workspaceId/companies/:id` | Crm.CompanyDelete | Удалить компанию |
| POST | `/api/v1/workspaces/:workspaceId/companies/:id/contacts/:contactId` | Crm.CompanyAttachContact | Привязать контакт |
| DELETE | `/api/v1/workspaces/:workspaceId/companies/:id/contacts/:contactId` | Crm.CompanyDetachContact | Отвязать контакт |
| GET | `/api/v1/workspaces/:workspaceId/pipelines` | Crm.PipelineList | Список воронок |
| POST | `/api/v1/workspaces/:workspaceId/pipelines` | Crm.PipelineCreate | Создать воронку |
| GET | `/api/v1/workspaces/:workspaceId/pipelines/:pipelineId` | Crm.PipelineGet | Одна воронка |
| PUT | `/api/v1/workspaces/:workspaceId/pipelines/:pipelineId` | Crm.PipelineUpdate | Обновить воронку |
| DELETE | `/api/v1/workspaces/:workspaceId/pipelines/:pipelineId` | Crm.PipelineDelete | Удалить воронку |
| GET | `/api/v1/workspaces/:workspaceId/pipelines/:pipelineId/stages` | Crm.StageList | Этапы воронки |
| GET | `/api/v1/workspaces/:workspaceId/pipelines/:pipelineId/stages/:id` | Crm.StageGet | Один этап |
| POST | `/api/v1/workspaces/:workspaceId/pipelines/:pipelineId/stages` | Crm.StageCreate | Создать этап |
| PUT | `/api/v1/workspaces/:workspaceId/pipelines/:pipelineId/stages/:id` | Crm.StageUpdate | Обновить этап |
| DELETE | `/api/v1/workspaces/:workspaceId/pipelines/:pipelineId/stages/:id` | Crm.StageDelete | Удалить этап |
| POST | `/api/v1/workspaces/:workspaceId/pipelines/:pipelineId/stages/reorder` | Crm.StageReorder | Изменить порядок этапов |
| GET | `/api/v1/workspaces/:workspaceId/deals` | Crm.DealList | Список сделок |
| GET | `/api/v1/workspaces/:workspaceId/deals/:id` | Crm.DealGet | Одна сделка |
| POST | `/api/v1/workspaces/:workspaceId/deals` | Crm.DealCreate | Создать сделку |
| PUT | `/api/v1/workspaces/:workspaceId/deals/:id` | Crm.DealUpdate | Обновить сделку |
| DELETE | `/api/v1/workspaces/:workspaceId/deals/:id` | Crm.DealDelete | Удалить сделку |
| GET | `/api/v1/workspaces/:workspaceId/activities` | Crm.ActivityList | Список активностей |
| GET | `/api/v1/workspaces/:workspaceId/activities/:id` | Crm.ActivityGet | Одна активность |
| POST | `/api/v1/workspaces/:workspaceId/activities` | Crm.ActivityCreate | Создать активность |
| PUT | `/api/v1/workspaces/:workspaceId/activities/:id` | Crm.ActivityUpdate | Обновить активность |
| DELETE | `/api/v1/workspaces/:workspaceId/activities/:id` | Crm.ActivityDelete | Удалить активность |
| POST | `/api/v1/workspaces/:workspaceId/activities/:id/important` | Crm.ActivityToggleImportant | Переключить важность |

### 1.9. Notes

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/api/v1/workspaces/:workspaceId/notes` | Notes.List | Список заметок |
| GET | `/api/v1/workspaces/:workspaceId/notes/:noteId` | Notes.Get | Одна заметка |
| POST | `/api/v1/workspaces/:workspaceId/notes` | Notes.Create | Создать заметку |
| PUT | `/api/v1/workspaces/:workspaceId/notes/:noteId` | Notes.Update | Обновить заметку |
| DELETE | `/api/v1/workspaces/:workspaceId/notes/:noteId` | Notes.Delete | Удалить заметку |

### 1.10. Habits

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/api/v1/workspaces/:workspaceId/habits` | Habits.List | Список привычек |
| POST | `/api/v1/workspaces/:workspaceId/habits` | Habits.Create | Создать привычку |
| GET | `/api/v1/workspaces/:workspaceId/habits/:habitId` | Habits.Get | Одна привычка |
| PUT | `/api/v1/workspaces/:workspaceId/habits/:habitId` | Habits.Update | Обновить привычку |
| DELETE | `/api/v1/workspaces/:workspaceId/habits/:habitId` | Habits.Delete | Удалить привычку |
| POST | `/api/v1/workspaces/:workspaceId/habits/:habitId/complete` | Habits.Complete | Отметить выполнение |
| POST | `/api/v1/workspaces/:workspaceId/habits/:habitId/toggle` | Habits.Toggle | Переключить выполнение за дату |
| GET | `/api/v1/workspaces/:workspaceId/habits/:habitId/stats` | Habits.GetStats | Статистика привычки |
| GET | `/api/v1/workspaces/:workspaceId/habits/:habitId/completions` | Habits.GetCompletions | Выполнения за период |
| GET | `/api/v1/workspaces/:workspaceId/habits/calendar` | Habits.GetCalendar | Календарь выполнений |

### 1.11. Journal

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/api/v1/workspaces/:workspaceId/journal` | Journal.List | Записи журнала (опционально по дате) |
| POST | `/api/v1/workspaces/:workspaceId/journal` | Journal.Create | Создать запись |
| GET | `/api/v1/workspaces/:workspaceId/journal/:entryId` | Journal.Get | Одна запись |
| PUT | `/api/v1/workspaces/:workspaceId/journal/:entryId` | Journal.Update | Обновить запись |
| DELETE | `/api/v1/workspaces/:workspaceId/journal/:entryId` | Journal.Delete | Удалить запись |

### 1.12. Projects

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/api/v1/workspaces/:workspaceId/projects` | Project.List | Список проектов |
| POST | `/api/v1/workspaces/:workspaceId/projects` | Project.Create | Создать проект |
| GET | `/api/v1/workspaces/:workspaceId/projects/:projectId` | Project.Get | Один проект |
| PUT | `/api/v1/workspaces/:workspaceId/projects/:projectId` | Project.Update | Обновить проект |
| DELETE | `/api/v1/workspaces/:workspaceId/projects/:projectId` | Project.Delete | Удалить проект |
| GET | `/api/v1/workspaces/:workspaceId/projects/:projectId/entities` | Project.ListEntityIDs | ID сущностей по типу |
| POST | `/api/v1/workspaces/:workspaceId/projects/:projectId/entities` | Project.AttachEntity | Привязать сущность |
| DELETE | `/api/v1/workspaces/:workspaceId/projects/:projectId/entities/:entityType/:entityId` | Project.DetachEntity | Отвязать сущность |
| GET | `/api/v1/workspaces/:workspaceId/entities/:entityType/:entityId/projects` | Project.GetProjectIDsForEntity | Проекты для сущности |

### 1.13. Admin (требуется глобальная роль ADMIN)

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/api/v1/admin/workspaces` | Admin.ListWorkspaces | Все workspace |
| GET | `/api/v1/admin/users` | Admin.ListUsers | Все пользователи |
| DELETE | `/api/v1/admin/users/:id` | Admin.DeleteUser | Удалить пользователя |
| POST | `/api/v1/admin/users/:id/licenses` | Admin.GrantLicense | Выдать лицензию на модуль |

### 1.14. Logs

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/api/v1/logs` | Logger.GetLogs | Логи по дате (query: `date`) |
| POST | `/api/v1/logs/sync` | Logger.SyncToDB | Синхронизация логов в БД |

---

## 2. Сервисы (Service)

### 2.1. auth (internal/service/auth)

| Метод | Описание |
|-------|----------|
| Register(ctx, req) | Регистрация пользователя |
| Login(ctx, req) | Вход, возврат токенов |
| GetUserProfile(ctx, userID) | Профиль по ID |
| Logout(ctx, userID) | Выход (инвалидация и т.п.) |

### 2.2. permission (internal/service/permission)

| Метод | Описание |
|-------|----------|
| GetCatalog(ctx) | Каталог прав из БД |
| ListRoles(ctx, workspaceID) | Роли workspace |
| GetRole(ctx, roleID) | Одна роль |
| GetRolePermissions(ctx, roleID) | Права роли (из Casbin) |
| CreateRole(ctx, workspaceID, name, description, permissions, createdBy) | Создать роль + политики Casbin |
| UpdateRole(ctx, roleID, name, description, permissions) | Обновить роль и политики (учёт смены имени) |
| DeleteRole(ctx, roleID) | Удалить роль (только кастомную) |
| AssignRole(ctx, userID, roleID, workspaceID, assignedBy) | Назначить роль + группировка в Casbin |
| RemoveRole(ctx, userID, roleID, workspaceID) | Снять роль |
| GetUserRoles(ctx, userID, workspaceID) | Имена ролей пользователя |
| GetUserRolesFull(ctx, userID, workspaceID) | Роли с полными данными |
| GrantPermission(ctx, userID, workspaceID, permissionID, grantedBy, expiresAt) | Выдать индивидуальное право |
| RevokePermission(ctx, userID, workspaceID, permissionID) | Отозвать право |
| GetUserPermissions(ctx, userID, workspaceID) | Индивидуальные права пользователя |
| GetEffectivePermissions(ctx, userID, workspaceID) | Все права (роли + индивидуальные) |
| EnsureSystemRolePolicies(ctx) | Загрузить базовые политики для системных ролей по всем workspace |
| SyncGroupingPoliciesFromAssignments(ctx) | Синхронизировать g(user, role, domain) из БД |
| SeedSystemPoliciesForWorkspace(workspaceID) | Залить системные политики для одного workspace |

### 2.3. workspace (internal/service/workspace)

| Метод | Описание |
|-------|----------|
| List(ctx, userID, userRole) | Список workspace пользователя |
| Create(ctx, dto, userID) | Создать workspace |
| Get(ctx, workspaceID, userID, userRole) | Один workspace |
| Update(ctx, workspaceID, dto, userID, userRole) | Обновить |
| Delete(ctx, workspaceID, userID, userRole) | Удалить |
| SetCurrentWorkspace(ctx, userID, workspaceID) | Установить текущий workspace |
| GetCurrentWorkspace(ctx, userID, userRole) | Текущий workspace |
| HasAccess(ctx, workspaceID, userID, userRole) | Есть ли доступ (в т.ч. глобальный ADMIN) |
| ListAllForAdmin(ctx) | Все workspace (админка) |
| GetWorkspaceModules(ctx, workspaceID, userID, userRole) | Модули workspace с состоянием |
| EnableModule(ctx, workspaceID, userID, userRole, moduleCode) | Включить модуль |
| DisableModule(ctx, workspaceID, userID, userRole, moduleCode) | Выключить модуль |
| ListMyLicenses(ctx, userID) | Лицензии пользователя |
| CanEnableModuleInWorkspace(ctx, workspaceID, userID, userRole, moduleCode) | Можно ли включить модуль |
| GrantLicense(ctx, targetUserID, moduleCode, scope, workspaceID) | Выдать лицензию |

### 2.4. crm (internal/service/crm)

| Метод | Описание |
|-------|----------|
| ContactList, ContactGet, ContactCreate, ContactUpdate, ContactDelete | Контакты |
| CompanyList, CompanyGet, CompanyCreate, CompanyUpdate, CompanyDelete | Компании |
| CompanyAttachContact, CompanyDetachContact | Связь компания–контакт |
| PipelineList, PipelineGet, PipelineCreate, PipelineUpdate, PipelineDelete | Воронки |
| StageList, StageGet, StageCreate, StageUpdate, StageDelete, StageReorder | Этапы |
| DealList, DealGet, DealCreate, DealUpdate, DealDelete | Сделки |
| ActivityList, ActivityGet, ActivityCreateNote, ActivityCreateCall, ActivityUpdate, ActivityDelete, ActivitySetImportant | Активности |

### 2.5. project (internal/service/project)

| Метод | Описание |
|-------|----------|
| List, Get, Create, Update, Delete | Проекты |
| AttachEntity, DetachEntity | Привязка/отвязка сущностей |
| ListEntityIDs, GetProjectIDsForEntity | Связи проект–сущность |

### 2.6. habits (internal/service/habits)

| Метод | Описание |
|-------|----------|
| List, Create, Get, Update, Delete | Привычки |
| Complete, Toggle | Выполнения |
| GetStats, GetCompletions, GetAllCompletions, GetCalendar | Статистика и календарь |

### 2.7. journal (internal/service/journal)

| Метод | Описание |
|-------|----------|
| List, Get, Create, Update, Delete | Записи журнала |

### 2.8. notes (internal/service/notes)

| Метод | Описание |
|-------|----------|
| List, Get, Create, Update, Delete | Заметки |

### 2.9. master (internal/service/master)

| Метод | Описание |
|-------|----------|
| ListCurrencies, GetCurrency, CreateCurrency, UpdateCurrency, DeleteCurrency | Валюты |
| ListCounterparties, GetCounterparty, CreateCounterparty, UpdateCounterparty, DeleteCounterparty | Контрагенты |

### 2.10. logger (internal/service/logger)

| Метод | Описание |
|-------|----------|
| WriteLog(logLine) | Запись строки лога |
| SyncToDB() | Синхронизация логов в БД |
| GetLogsByDate(ctx, date) | Логи за дату |

---

## 3. Репозитории (Repository)

### 3.1. permission (internal/repository/permission)

| Метод | Описание |
|-------|----------|
| ListCatalog, GetCatalogByID | Каталог прав |
| ListRolesByWorkspace, GetRoleByID, GetRoleByName | Роли |
| CreateRole, UpdateRole, DeleteRole | CRUD ролей |
| ListUserRoleAssignments, CreateUserRoleAssignment, DeleteUserRoleAssignment | Назначения ролей |
| ListUserPermissions, CreateUserPermission, DeleteUserPermission | Индивидуальные права |
| CountAssignmentsByRole | Количество назначений по роли |
| ListDistinctWorkspaceIDs | Список workspace_id из ролей |
| ListAllUserRoleAssignments | Все назначения (для синхронизации Casbin) |

### 3.2. workspace (internal/repository/workspace)

| Метод | Описание |
|-------|----------|
| List, ListAll, Create, Get, Update, Delete | Workspace |
| HasAccess, CheckAccess, IsOwner | Доступ |
| GetModuleByCode, ListAllModules, ListWorkspaceModules, ListAllModulesWithWorkspaceState | Модули |
| AddWorkspaceModule, SetWorkspaceModuleStatus | Включение/выключение модулей |

### 3.3. user (internal/repository/user)

| Метод | Описание |
|-------|----------|
| Create, FindByEmail, FindByEmailAnyStatus, FindByID, Update, ListAll, Delete | Пользователи |

### 3.4. crm (internal/repository/crm)

| Метод | Описание |
|-------|----------|
| ContactList, ContactGet, ContactCreate, ContactUpdate, ContactDelete | Контакты |
| CompanyList, CompanyGet, CompanyCreate, CompanyUpdate, CompanyDelete, CompanyAttachContact, CompanyDetachContact | Компании |
| PipelineList, PipelineGetByID, PipelineCreate, PipelineUpdate, PipelineDelete | Воронки |
| StageList, StageGet, StageCreate, StageUpdate, StageDelete, StageReorder | Этапы |
| DealList, DealGet, DealCreate, DealUpdate, DealDelete | Сделки |
| ActivityList, ActivityGet, ActivityCreate, ActivityUpdate, ActivitySetImportant, ActivityDelete | Активности |

### 3.5. project, habits, journal, notes, master, license, logger

Репозитории по доменам: CRUD и специализированные запросы для проектов, привычек, журнала, заметок, справочников (валюты, контрагенты), лицензий, логов. Перечень методов соответствует вызовам из одноимённых сервисов.

---

## 4. Casbin: как работает и инструменты

### 4.1. Модель (internal/authz/casbin.go)

- **Request:** `r = sub, dom, obj, act`  
  - `sub` — субъект (у нас: `user:<userID>` или `role:<roleName>` при проверке роли).  
  - `dom` — домен (workspace_id).  
  - `obj` — объект (например, `crm:deal`).  
  - `act` — действие (например, `read`, `create`).

- **Policy (p):** `p = sub, dom, obj, act` — одна строка: «субъект sub в домене dom имеет право (obj, act)».

- **Grouping (g):** `g = _, _, _` — тройка (user, role, domain): пользователь имеет роль в домене.  
  Пример: `g("user:uuid-1", "role:MEMBER", "workspace-uuid")`.

- **Matcher:**  
  `m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act`  
  То есть: запрос (sub, dom, obj, act) разрешён, если есть роль p.sub у r.sub в r.dom и есть политика (p.sub, dom, obj, act).

### 4.2. Где хранятся данные

- **Таблица БД:** `casbin_rule` (поля: ptype, v0, v1, v2, v3, v4, v5).  
  Заполняется через GORM-адаптер Casbin.

- **Типы записей:**  
  - `p` — политика: v0=sub, v1=dom, v2=obj, v3=act.  
  - `g` — группировка: v0=user, v1=role, v2=dom.

### 4.3. Инструменты (API Enforcer)

| Метод | Назначение |
|-------|------------|
| LoadPolicy() | Загрузить политики и группировки из БД в память |
| SavePolicy() | Сохранить текущие политики из памяти в БД |
| Enforce(sub, dom, obj, act) | Проверить доступ: true/false |
| AddPolicy(sub, dom, obj, act) | Добавить политику (p) |
| RemovePolicy(sub, dom, obj, act) | Удалить политику |
| GetFilteredPolicy(fieldIndex, value) | Получить политики по значению поля (например, по sub) |
| AddGroupingPolicy(user, role, dom) | Добавить g(user, role, dom) |
| RemoveGroupingPolicy(user, role, dom) | Удалить g(user, role, dom) |
| GetGroupingPolicy() | Все группировки |

### 4.4. Как используется в приложении

1. **Старт:** `InitEnforcer(gormDB)` создаёт Enforcer, загружает модель и вызывает `LoadPolicy()`.
2. **Системные роли:** при старте вызываются `EnsureSystemRolePolicies` и `SyncGroupingPoliciesFromAssignments` — в Casbin заливаются политики для OWNER/ADMIN/MEMBER/GUEST и все g(user, role, workspace) из `user_role_assignments`.
3. **Кастомные роли:** при CreateRole/UpdateRole в Casbin добавляются/обновляются политики с sub=`role:<name>`, dom=workspaceID.
4. **Назначение ролей:** AssignRole/RemoveRole добавляют/удаляют группировки g(user:id, role:name, workspaceID).
5. **Проверка в запросе:** PermissionMiddleware по (method, path) получает (obj, act), затем вызывает `Enforcer.Enforce("user:"+userID, workspaceID, obj, act)`. Casbin по g находит роли пользователя в домене и проверяет наличие политики (role, dom, obj, act).
6. **Индивидуальные права:** в GetEffectivePermissions к правам из ролей (через политики) добавляются права из таблицы `user_permissions`; в текущей реализации проверка в middleware идёт только через роли (индивидуальные права можно учесть в Enforce через доп. слой или отдельные политики p с sub=user:id).

### 4.5. Graceful shutdown

В `App.Shutdown` перед закрытием БД вызывается `Enforcer.SavePolicy()`, чтобы изменения политик не терялись.

---

## 5. План разбиения на сервисы и Docker

### 5.1. Текущее состояние

- Один Go-монолит: API, все сервисы, репозитории, Casbin в одном процессе.
- Одна PostgreSQL: все таблицы, включая `casbin_rule`.
- Роутер регистрирует все маршруты; middleware общие (auth, workspace, module, permission).

### 5.2. Варианты разбиения

#### Вариант A: Монолит в одном Docker-образе

- **Сервис:** один контейнер (backend).
- **БД:** один контейнер (postgres).
- **docker-compose:** 2 сервиса (api, db). Масштабирование — несколько реплик api за балансировщиком; политики Casbin у каждого инстанса свои в памяти (см. SPEC_ROLE_BACK.md — инвалидация/LoadPolicy при изменении).

#### Вариант B: Вынос Authz (микросервис прав)

- **authz-service:** небольшой сервис с Casbin: только проверка Enforce(sub, dom, obj, act) и при необходимости выдача «эффективных прав». Хранит политики в той же PostgreSQL или в своей копии.
- **api (монолит):** все остальные эндпоинты; PermissionMiddleware делает HTTP/gRPC вызов к authz-service вместо локального Enforcer. Изменение ролей/назначений по-прежнему в api, затем вызов authz на обновление политик или публикация события.
- **Docker:** 3 сервиса: api, authz, postgres. Общая сеть; authz подключается к БД для casbin_rule.

#### Вариант C: По доменам (CRM, Habits, Projects, Workspace, Auth)

- Каждый домен — отдельный сервис (crm-service, habits-service, project-service, workspace-service, auth-service).
- Общие: БД (или схема/база на сервис), единый API Gateway или BFF, который маршрутизирует на сервисы.
- Права: либо общий authz-service, либо каждый сервис запрашивает право у центрального Authz.

### 5.3. Рекомендуемый пошаговый план

1. **Сейчас:** один Docker-образ backend + postgres, docker-compose для разработки и деплоя.
2. **При росте нагрузки:** несколько реплик backend; кэш GetEffectivePermissions; при изменении прав — инвалидация кэша или вызов LoadPolicy на всех инстансах (через очередь/бродкаст).
3. **При необходимости изоляции прав:** вынести Authz в отдельный сервис (Вариант B), оставив остальной API в монолите.
4. **При дальнейшем росте:** разбивать по доменам (Вариант C), начиная с самого нагруженного модуля (например, CRM).

### 5.4. Пример docker-compose (монолит)

```yaml
services:
  api:
    build: ./backend
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://user:pass@db:5432/erp?sslmode=disable
    depends_on:
      - db

  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
      POSTGRES_DB: erp
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

При разбиении на authz и api добавить сервис `authz`, оба подключать к `db`; api в переменных окружения указывает URL authz-сервиса для проверки прав.

---

**Конец документа.**
