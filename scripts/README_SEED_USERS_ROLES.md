# Seed: пользователи и роли (мультитенаси)

Сид создаёт тестовых пользователей с разными ролями в одном workspace для проверки мультитенаси и прав **без реализации приглашений**.

## Запуск

```bash
cd backend
go run ./cmd/seed_users_roles
```

Переменные окружения (из `.env` или окружения):

- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` — подключение к PostgreSQL
- `SKIP_EXISTING=1` — не перезаписывать существующих пользователей/workspace
- `RESET=1` — удалить seed workspaces перед созданием
- `RESET_FULL=1` — удалить seed users + все их workspaces, затем создать заново (осторожно: удалит все данные этих пользователей)

## Откуда участник (в т.ч. владелец) получает роли

| Источник | Назначение |
|----------|------------|
| `user_workspaces` | Членство в workspace, поле `role` (OWNER/ADMIN/MEMBER/GUEST) |
| `user_role_assignments` | Права для Casbin и API: `GetUserRoles` → `GetMyPermissions` → `systemRole` |

Оба должны быть синхронизированы. При создании workspace через API:
1. `workspace.Repo.Create` добавляет владельца в `user_workspaces` (role=OWNER)
2. `permSvc.AssignRoleByName` добавляет запись в `user_role_assignments`
3. При сбое шага 2 миграция **000024** при следующем старте API исправит запись

## Созданные данные

### Пользователи (пароль для всех: `Password123!`)

| Email              | Имя          | Роль в "Демо" | Роль в "Личный" |
|--------------------|--------------|---------------|------------------|
| owner@demo.local   | Владелец     | OWNER         | —                |
| admin@demo.local   | Администратор| ADMIN         | —                |
| member@demo.local  | Сотрудник    | MEMBER        | —                |
| guest@demo.local   | Гость        | GUEST         | —                |
| multi@demo.local   | Мультитенант | MEMBER        | OWNER            |

### Workspaces

- **Демо** — основной workspace с 5 участниками и разными ролями
- **Личный** — второй workspace, владелец `multi@demo.local` (для проверки мультитенаси)

## Тестирование мультитенаси

1. Запустите сид: `go run ./cmd/seed_users_roles`
2. Запустите API: `go run ./cmd/api`
3. Войдите как `multi@demo.local` / `Password123!`
4. Переключайте workspace в UI — должны быть доступны «Демо» и «Личный»
5. В «Демо» — права MEMBER, в «Личный» — OWNER

## Тестирование разных ролей в одном workspace

1. Войдите как `owner@demo.local` — полный доступ
2. Войдите как `admin@demo.local` — доступ администратора
3. Войдите как `member@demo.local` — ограниченный доступ
4. Войдите как `guest@demo.local` — минимальный доступ

## Важно

- Сид заполняет **user_workspaces** (членство) и **user_role_assignments** (права для Casbin)
- После сида перезапустите API, чтобы Casbin подхватил новые назначения (`SyncGroupingPoliciesFromAssignments` вызывается при старте)
- Приглашения не используются — пользователи добавляются напрямую в БД
