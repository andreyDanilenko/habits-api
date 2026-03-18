# Быстрый гайд: лицензии модулей и включение/отключение

## Как работает лицензия

### Два типа scope (области действия)

| Scope | Описание | Когда использовать |
|-------|----------|-------------------|
| **`all_workspaces`** | Модуль можно включать в **любом** workspace пользователя | Полная лицензия на модуль |
| **`single_workspace`** | Модуль только в **одном указанном** workspace | Ограниченная лицензия |

**Частичные или полные?** — Оба варианта есть:
- **Полная** = `all_workspaces` — доступ везде
- **Частичная** = `single_workspace` — доступ только в одном workspace

---

## Быстро выдать лицензию пользователю (админ)

### Через API (curl)

```bash
# Лицензия на все workspace пользователя
curl -X POST "https://<host>/api/v1/admin/users/<user_id>/licenses" \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{"moduleCode": "crm", "scope": "all_workspaces"}'

# Лицензия только на один workspace
curl -X POST "https://<host>/api/v1/admin/users/<user_id>/licenses" \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{"moduleCode": "crm", "scope": "single_workspace", "workspaceId": "<workspace_uuid>"}'
```

### Через БД (если нет доступа к API)

```sql
-- all_workspaces
INSERT INTO user_module_licenses (user_id, module_id, scope, workspace_id, status, source)
SELECT 
  '<user_uuid>'::uuid,
  m.id,
  'all_workspaces',
  NULL,
  'active',
  'admin_grant'
FROM modules m WHERE m.code = 'crm';

-- single_workspace
INSERT INTO user_module_licenses (user_id, module_id, scope, workspace_id, status, source)
SELECT 
  '<user_uuid>'::uuid,
  m.id,
  'single_workspace',
  '<workspace_uuid>'::uuid,
  'active',
  'admin_grant'
FROM modules m WHERE m.code = 'crm';
```

---

## Быстро включить модуль в workspace

### Кто может
- **Владелец workspace** (`workspaces.owner_id`)
- **Глобальный админ** (`users.role = ADMIN`)

### Через UI
Настройки воркспейса → Модули → «Активировать» у нужного модуля.

### Через API

```bash
curl -X POST "https://<host>/api/v1/workspaces/<workspace_id>/modules" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"moduleCode": "crm"}'
```

### Через БД

```sql
INSERT INTO workspace_modules (workspace_id, module_id, status)
SELECT '<workspace_uuid>'::uuid, id, 'active'
FROM modules WHERE code = 'crm'
ON CONFLICT (workspace_id, module_id) DO UPDATE SET status = 'active';
```

---

## Быстро отключить модуль

### Через UI
Настройки воркспейса → Модули → «Отключить» у модуля.

### Через API

```bash
curl -X DELETE "https://<host>/api/v1/workspaces/<workspace_id>/modules/crm" \
  -H "Authorization: Bearer <token>"
```

### Через БД

```sql
UPDATE workspace_modules
SET status = 'disabled'
WHERE workspace_id = '<workspace_uuid>'
  AND module_id = (SELECT id FROM modules WHERE code = 'crm');
```

---

## Отозвать лицензию у пользователя

Лицензия хранится в `user_module_licenses`. Отключение модуля в workspace **не удаляет** лицензию — пользователь может снова включить модуль.

Чтобы полностью отозвать доступ:

```sql
-- Отключить лицензию (status = revoked)
UPDATE user_module_licenses
SET status = 'revoked', updated_at = NOW()
WHERE user_id = '<user_uuid>'
  AND module_id = (SELECT id FROM modules WHERE code = 'crm');

-- Или удалить запись
DELETE FROM user_module_licenses
WHERE user_id = '<user_uuid>'
  AND module_id = (SELECT id FROM modules WHERE code = 'crm');
```

---

## Проверить лицензии пользователя

```bash
GET /api/v1/workspaces/me/module-licenses
```

Ответ: список активных лицензий с `moduleCode`, `scope`, `workspaceId` (для single_workspace).

---

## Core-модули

Модули с `is_core = true` (например `habits`) **не требуют лицензии** — владелец workspace может включать их без проверки `user_module_licenses`.
