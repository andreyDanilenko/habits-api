# Гайд: модули и лицензии

Полное руководство по системе модулей и лицензий: таблицы, связи, логика, API, UI.

---

## 1. Связи таблиц

```
modules (справочник модулей)
├── id, code, name, description, is_core
└── code = 'habits' | 'crm' | 'projects' | 'tasks' | ...

workspace_modules (модули, включённые в workspace)
├── workspace_id → workspaces
├── module_id → modules
├── status: 'active' | 'trial' | 'disabled'
└── Одна запись = модуль включён в этом workspace

user_module_licenses (лицензии пользователя на модуль)
├── user_id → users
├── module_id → modules
├── scope: 'all_workspaces' | 'single_workspace'
├── workspace_id (только при single_workspace)
├── status: 'active' | 'expired' | 'cancelled'
└── source: 'purchase' | 'admin_grant'
```

### Схема связей

```
users
  └── user_module_licenses (user_id)
        └── modules (module_id)

workspaces
  └── workspace_modules (workspace_id)
        └── modules (module_id)
```

**Важно:** `user_module_licenses` и `workspace_modules` — разные сущности:

- **user_module_licenses** — «право пользователя включить модуль» (лицензия)
- **workspace_modules** — «модуль фактически включён в workspace»

---

## 2. Почему модуль открыт по умолчанию, но после отключения нужна покупка?

### Core-модули (`is_core = true`)

- **habits** и другие core — бесплатны
- Владелец workspace может включать их без лицензии
- При создании workspace core-модули часто включаются автоматически (seed/миграция)

### Не-core модули (CRM, Projects, Tasks и т.п.)

1. **Включение модуля** — владелец workspace нажимает «Активировать» на странице «Модули».
2. **Проверка:** у владельца должна быть активная запись в `user_module_licenses` (или он ADMIN, или модуль core).
3. **Запись в workspace_modules:** `status = 'active'` — модуль включён.

### Отключение модуля

- Владелец нажимает «Отключить» → в `workspace_modules` ставится `status = 'disabled'`.
- **Лицензия в user_module_licenses НЕ удаляется** — пользователь может снова включить модуль без покупки.

### Отзыв лицензии (админ)

- Админ отзывает лицензию → в `user_module_licenses` ставится `status = 'cancelled'`.
- Теперь у пользователя **нет права** включать этот модуль.
- Модули, уже включённые в workspace, остаются в `workspace_modules` со status `active`, но при следующей проверке доступа (или при попытке включить модуль в другом workspace) — лицензия не найдена.
- **Итог:** после отзыва лицензии пользователь не может включать модуль в новых workspace; чтобы снова использовать — нужна новая лицензия (покупка или выдача админом).

### «Нужно купить потом»

Если имеется в виду **отключение модуля в workspace** — лицензия сохраняется, покупать заново не нужно.

Если имеется в виду **отзыв лицензии админом** — да, после отзыва нужна новая лицензия (покупка или повторная выдача).

---

## 3. Scope лицензии

| Scope | Описание | workspace_id |
|-------|----------|---------------|
| **all_workspaces** | Модуль можно включать в любом workspace пользователя | NULL |
| **single_workspace** | Модуль только в указанном workspace | UUID workspace |

Один пользователь может иметь:
- одну лицензию `all_workspaces` на модуль CRM;
- или несколько `single_workspace` на CRM для разных workspace.

---

## 4. API

### Текущий пользователь

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/api/v1/workspaces/me/module-licenses` | Список активных лицензий |

### Админ

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/api/v1/admin/modules` | Все модули |
| GET | `/api/v1/admin/workspaces` | Все workspace |
| GET | `/api/v1/admin/users/:id/licenses` | Лицензии пользователя |
| POST | `/api/v1/admin/users/:id/licenses` | Выдать лицензию |
| DELETE | `/api/v1/admin/users/:id/licenses/:licenseId` | Отозвать лицензию |

### Тело POST (выдача лицензии)

```json
{
  "moduleCode": "crm",
  "scope": "all_workspaces"
}
```

или

```json
{
  "moduleCode": "crm",
  "scope": "single_workspace",
  "workspaceId": "uuid-workspace"
}
```

---

## 5. UI: админ-панель

1. Админ-панель → список пользователей.
2. Кнопка **«Лицензии»** у пользователя.
3. Модалка:
   - список текущих лицензий;
   - кнопка **«Отозвать»** у каждой;
   - форма выдачи: модуль, scope, workspace (при single_workspace).

---

## 6. Цепочка при включении модуля

1. Владелец workspace вызывает `POST /workspaces/:id/modules` с `{ "moduleCode": "crm" }`.
2. Проверка: пользователь — владелец или ADMIN.
3. Модуль не core → проверка `user_module_licenses`: есть активная запись с подходящим scope.
4. `INSERT INTO workspace_modules ... ON CONFLICT DO UPDATE SET status = 'active'`.

---

## 7. Цепочка при отзыве лицензии

1. Админ вызывает `DELETE /admin/users/:id/licenses/:licenseId`.
2. `UPDATE user_module_licenses SET status = 'cancelled' WHERE id = :licenseId AND user_id = :id`.
3. Запись остаётся в БД, но `status = 'cancelled'` — при проверке HasLicense не учитывается.

---

## 8. Быстрые команды (для справки)

См. [MODULES_LICENSE_QUICK_GUIDE.md](./MODULES_LICENSE_QUICK_GUIDE.md) — curl, SQL, отключение через UI.
