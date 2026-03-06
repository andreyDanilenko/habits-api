# Роли: глобальная vs workspace

## Два уровня ролей

| Уровень | Таблица/источник | Значения | Назначение |
|---------|------------------|----------|------------|
| **Глобальная** | `users.role` | `USER`, `ADMIN` | Доступ к системным функциям (админ-панель, биллинг) |
| **Workspace** | `user_role_assignments` | `OWNER`, `ADMIN`, `MEMBER`, `GUEST` | Права внутри конкретного workspace |

## users.role (глобальная)

- **USER** — обычный пользователь. Не даёт доступ к `/admin`, биллингу.
- **ADMIN** — системный администратор. Полный доступ к админ-панели, биллингу, всем workspace.

**Не используется** для проверки прав внутри workspace. Только для системных эндпоинтов.

## Workspace-роли (user_role_assignments)

- **OWNER** — владелец workspace, полный доступ.
- **ADMIN** — админ workspace, почти полный доступ.
- **MEMBER** — участник, базовые права.
- **GUEST** — гость, только чтение.

Один пользователь может быть **OWNER** в одном workspace и **MEMBER** в другом.

## API

- `GET /auth/me` → `user.role` (глобальная: USER/ADMIN)
- `GET /me/permissions?workspaceId=...` → `systemRole` (workspace: OWNER/ADMIN/MEMBER/GUEST)

## Фронтенд

- Для **доступа к модулям, настройкам workspace** — использовать `effectivePermissions.systemRole`
- Для **доступа к /admin, биллингу** — использовать `user.role === 'ADMIN'`
