# Техническое задание: Реализация приглашений пользователей в workspace (Backend)

## Версия 1.0 | Март 2026

---

## Содержание

1. **Цель и задачи**
2. **Бизнес-логика процесса приглашения**
3. **Сценарии использования**
4. **Модель данных**
5. **API спецификация**
6. **Интеграция с существующей системой**
7. **Обработка пограничных случаев**
8. **Детальная декомпозиция задач**
9. **Критерии приемки**
10. **Оценка сроков**

---

## 1. Цель и задачи

### 1.1. Цель
Реализовать механизм приглашения пользователей в workspace, который позволит администраторам добавлять новых участников (как существующих, так и новых пользователей) с предопределёнными ролями и правами.

### 1.2. Задачи

- Создать механизм генерации уникальных ссылок-приглашений
- Реализовать отправку email-уведомлений
- Обработать два ключевых сценария:
  - Приглашение **существующего** пользователя системы
  - Приглашение **нового** пользователя (ещё не зарегистрированного)
- Интегрировать приглашения с системой ролей и прав
- Обеспечить безопасность (защита от подделки ссылок, ограничение по времени)

---

## 2. Бизнес-логика процесса приглашения

### 2.1. Общая схема

```
[ADMIN] 
    │
    ├─ 1. Заполняет форму приглашения
    │      • Email
    │      • Системная роль (MEMBER/GUEST)
    │      • Кастомные роли
    │      • Индивидуальные права
    │      • Срок действия приглашения
    │
    ▼
[Бэкенд] 
    │
    ├─ 2. Создаёт запись в таблице invitations
    │      • Генерирует уникальный токен
    │      • Устанавливает статус PENDING
    │      • Рассчитывает expires_at
    │
    ▼
[Email] 
    │
    ├─ 3. Отправляет письмо со ссылкой
    │      • https://app.com/invite/{token}
    │
    ▼
[Пользователь]
    │
    ├─ 4. Переходит по ссылке
    │
    ▼
[Бэкенд] 
    │
    ├─ 5. Проверяет токен
    │      • Существует ли
    │      • Не истёк ли
    │      • Не принят ли уже
    │
    ├─ 6. Определяет статус пользователя
    │      • Вариант А: Пользователь НЕ зарегистрирован
    │      • Вариант Б: Пользователь зарегистрирован и вошёл в систему
    │      • Вариант В: Пользователь зарегистрирован, но не в системе
    │
    ▼
[Результат]
    ├─ • Пользователь добавляется в workspace
    │  • Назначаются роли и права
    │  • Приглашение помечается как ACCEPTED
    │  • Пользователь перенаправляется в workspace
```

### 2.2. Ключевое решение: Что делать с новым пользователем?

**Вопрос:** Если пользователя нет в системе, создавать его или сообщать, что его нет?

**Ответ: СОЗДАВАТЬ ПОЛЬЗОВАТЕЛЯ.**

Это стандартная практика (Slack, Notion, Figma) — приглашение работает как "пред-регистрация". Пользователь переходит по ссылке, видит форму регистрации (или сразу создаётся аккаунт, если используется SSO).

### 2.3. Детализация сценариев

#### Сценарий 1: Пользователь существует в системе

| Шаг | Действие |
|-----|----------|
| 1 | Пользователь переходит по ссылке `/invite/{token}` |
| 2 | Бэкенд проверяет токен (валидный, не истёк) |
| 3 | Бэкенд определяет, что пользователь с таким email уже есть в `users` |
| 4 | Проверяет, аутентифицирован ли пользователь в текущей сессии |
| 5 | **Если аутентифицирован** — сразу добавляет в workspace и редиректит |
| 6 | **Если не аутентифицирован** — предлагает войти, после входа добавляет в workspace |

#### Сценарий 2: Пользователь НЕ существует в системе

| Шаг | Действие |
|-----|----------|
| 1 | Пользователь переходит по ссылке `/invite/{token}` |
| 2 | Бэкенд проверяет токен |
| 3 | Определяет, что пользователь с таким email не найден |
| 4 | Перенаправляет на страницу регистрации с pre-filled email |
| 5 | После регистрации пользователь автоматически добавляется в workspace |
| 6 | Приглашение помечается как ACCEPTED |

---

## 3. Сценарии использования

### 3.1. Акторы

- **ADMIN / OWNER workspace** — создаёт приглашения
- **Приглашённый пользователь** — принимает приглашение

### 3.2. Use Cases

#### UC-01: Создание приглашения

**Предусловия:** Пользователь имеет роль ADMIN или OWNER в workspace

**Основной поток:**
1. ADMIN заполняет форму с email, системной ролью, кастомными ролями, правами
2. Система создаёт запись в `invitations` с уникальным токеном
3. Система отправляет email со ссылкой
4. Система возвращает информацию о созданном приглашении

**Альтернативные потоки:**
- Email уже есть в workspace — ошибка "Пользователь уже в workspace"
- Неверный формат email — ошибка валидации

#### UC-02: Просмотр списка приглашений

**Предусловия:** ADMIN в workspace

**Основной поток:**
1. ADMIN запрашивает список приглашений
2. Система возвращает все приглашения для данного workspace с фильтрацией по статусу

#### UC-03: Отзыв приглашения

**Предусловия:** ADMIN в workspace, приглашение существует и имеет статус PENDING

**Основной поток:**
1. ADMIN отзывает приглашение
2. Система меняет статус на CANCELLED
3. Ссылка больше не работает

#### UC-04: Принятие приглашения (существующий пользователь)

**Предусловия:** Приглашение существует и не истекло, пользователь с таким email существует

**Основной поток:**
1. Пользователь переходит по ссылке
2. Система проверяет токен
3. Пользователь аутентифицирован → добавляется в workspace
4. Приглашение помечается ACCEPTED
5. Пользователь перенаправляется в workspace

**Альтернативные потоки:**
- Пользователь не аутентифицирован → перенаправление на логин, после логина автоматическое принятие

#### UC-05: Принятие приглашения (новый пользователь)

**Предусловия:** Приглашение существует и не истекло, пользователь с таким email не существует

**Основной поток:**
1. Пользователь переходит по ссылке
2. Система проверяет токен
3. Пользователя нет в системе → перенаправление на регистрацию с pre-filled email
4. После успешной регистрации пользователь добавляется в workspace
5. Приглашение помечается ACCEPTED
6. Пользователь перенаправляется в workspace

---

## 4. Модель данных

### 4.1. Таблица invitations

```sql
CREATE TABLE invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Кто и куда приглашает
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    invited_by UUID NOT NULL REFERENCES users(id),
    
    -- Роли и права (полностью повторяет структуру назначения)
    system_role VARCHAR(20) NOT NULL CHECK (system_role IN ('MEMBER', 'GUEST')),
    role_ids UUID[],  -- массив ID кастомных ролей из workspace_roles
    
    -- Индивидуальные права (JSON массив для гибкости)
    permissions JSONB DEFAULT '[]'::jsonb,  -- [{ permissionId: 'uuid', expiresAt: '2026-...' }]
    
    -- Статус и токен
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' 
        CHECK (status IN ('PENDING', 'ACCEPTED', 'EXPIRED', 'CANCELLED')),
    token VARCHAR(255) UNIQUE NOT NULL,
    
    -- Временные метки
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    accepted_at TIMESTAMP,
    
    -- Индексы для быстрого поиска
    CONSTRAINT fk_invitations_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
    CONSTRAINT fk_invitations_invited_by FOREIGN KEY (invited_by) REFERENCES users(id)
);

-- Индексы
CREATE INDEX idx_invitations_token ON invitations(token);
CREATE INDEX idx_invitations_email ON invitations(email);
CREATE INDEX idx_invitations_status ON invitations(status);
CREATE INDEX idx_invitations_expires ON invitations(expires_at) WHERE status = 'PENDING';
CREATE INDEX idx_invitations_workspace ON invitations(workspace_id);

-- Комментарии
COMMENT ON TABLE invitations IS 'Приглашения пользователей в workspace';
COMMENT ON COLUMN invitations.system_role IS 'Системная роль: MEMBER или GUEST';
COMMENT ON COLUMN invitations.role_ids IS 'Массив ID кастомных ролей из workspace_roles';
COMMENT ON COLUMN invitations.permissions IS 'JSON массив индивидуальных прав: [{ permissionId, expiresAt }]';
COMMENT ON COLUMN invitations.token IS 'Уникальный токен для ссылки-приглашения';
COMMENT ON COLUMN invitations.status IS 'PENDING, ACCEPTED, EXPIRED, CANCELLED';
```

### 4.2. Пример записи

```json
{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "workspace_id": "123e4567-e89b-12d3-a456-426614174000",
    "email": "newuser@example.com",
    "invited_by": "user-789",
    "system_role": "MEMBER",
    "role_ids": ["role-111", "role-222"],
    "permissions": [
        {
            "permissionId": "perm-333",
            "expiresAt": "2026-04-01T00:00:00Z"
        }
    ],
    "status": "PENDING",
    "token": "abc123def456",
    "expires_at": "2026-03-10T00:00:00Z",
    "created_at": "2026-03-03T10:00:00Z",
    "accepted_at": null
}
```

---

## 5. API спецификация

### 5.1. Создание приглашения

```http
POST /api/v1/workspaces/{workspaceId}/invitations
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**
```json
{
    "email": "user@example.com",
    "systemRole": "MEMBER",              // "MEMBER" или "GUEST"
    "roleIds": ["uuid1", "uuid2"],        // опционально, ID кастомных ролей
    "permissions": [                       // опционально
        {
            "permissionId": "uuid3",
            "expiresAt": "2026-04-01T00:00:00Z"  // опционально
        }
    ],
    "expiresIn": 604800                    // опционально, срок действия в секундах (по умолчанию 7 дней)
}
```

**Response 201 Created:**
```json
{
    "id": "uuid",
    "email": "user@example.com",
    "workspaceId": "uuid",
    "invitedBy": {
        "id": "uuid",
        "name": "Admin Name",
        "email": "admin@example.com"
    },
    "systemRole": "MEMBER",
    "roleIds": ["uuid1", "uuid2"],
    "permissions": [
        {
            "permissionId": "uuid3",
            "expiresAt": "2026-04-01T00:00:00Z"
        }
    ],
    "status": "PENDING",
    "inviteLink": "https://app.com/invite/abc123def456",
    "expiresAt": "2026-03-10T00:00:00Z",
    "createdAt": "2026-03-03T10:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` — неверный email, неверная системная роль
- `400 Bad Request` — пользователь уже в workspace
- `403 Forbidden` — нет прав (не ADMIN/OWNER)
- `404 Not Found` — workspace не найден

### 5.2. Массовое создание приглашений

```http
POST /api/v1/workspaces/{workspaceId}/invitations/bulk
```

**Request Body:**
```json
{
    "emails": ["user1@example.com", "user2@example.com"],
    "systemRole": "MEMBER",
    "roleIds": ["uuid1", "uuid2"],
    "expiresIn": 604800
}
```

**Response 201 Created:**
```json
{
    "invitations": [
        {
            "email": "user1@example.com",
            "status": "PENDING",
            "inviteLink": "https://app.com/invite/abc123"
        },
        {
            "email": "user2@example.com",
            "status": "PENDING",
            "inviteLink": "https://app.com/invite/def456"
        }
    ],
    "failed": [
        {
            "email": "already@example.com",
            "reason": "User already in workspace"
        }
    ]
}
```

### 5.3. Получение списка приглашений

```http
GET /api/v1/workspaces/{workspaceId}/invitations?status=PENDING&limit=50&offset=0
```

**Query Parameters:**
- `status` — фильтр по статусу (PENDING, ACCEPTED, EXPIRED, CANCELLED)
- `limit` — количество записей (по умолчанию 50)
- `offset` — смещение для пагинации

**Response 200 OK:**
```json
{
    "invitations": [
        {
            "id": "uuid",
            "email": "user@example.com",
            "invitedBy": {
                "id": "uuid",
                "name": "Admin Name"
            },
            "systemRole": "MEMBER",
            "roleIds": ["uuid1", "uuid2"],
            "status": "PENDING",
            "expiresAt": "2026-03-10T00:00:00Z",
            "createdAt": "2026-03-03T10:00:00Z"
        }
    ],
    "total": 1,
    "limit": 50,
    "offset": 0
}
```

### 5.4. Отзыв приглашения

```http
DELETE /api/v1/workspaces/{workspaceId}/invitations/{invitationId}
```

**Response 204 No Content**

**Error Responses:**
- `403 Forbidden` — нет прав
- `404 Not Found` — приглашение не найдено
- `400 Bad Request` — приглашение уже принято (нельзя отозвать)

### 5.5. Получение информации о приглашении по токену (публичный)

```http
GET /api/v1/public/invitations/{token}
```

**Response 200 OK:**
```json
{
    "email": "user@example.com",
    "workspaceName": "Ромашка",
    "invitedByName": "Admin Name",
    "systemRole": "MEMBER",
    "expiresAt": "2026-03-10T00:00:00Z",
    "userExists": true,  // существует ли пользователь с таким email
    "isAuthenticated": false  // аутентифицирован ли текущий пользователь (если есть сессия)
}
```

**Error Responses:**
- `404 Not Found` — токен не существует
- `410 Gone` — приглашение истекло или уже принято

### 5.6. Принятие приглашения

```http
POST /api/v1/public/invitations/{token}/accept
```

**Request Body (опционально, если пользователь не аутентифицирован):**
```json
{
    "action": "login" | "register"  // указывает, что делать дальше
}
```

**Response 200 OK (пользователь аутентифицирован):**
```json
{
    "status": "accepted",
    "redirectTo": "/workspace/123"
}
```

**Response 401 Unauthorized (пользователь не аутентифицирован):**
```json
{
    "status": "requires_auth",
    "email": "user@example.com",
    "userExists": true,
    "redirectTo": "/login?email=user@example.com&inviteToken=abc123"
}
```

**Response 409 Conflict (приглашение уже принято или истекло):**
```json
{
    "status": "expired",
    "message": "Invitation has expired"
}
```

---

## 6. Интеграция с существующей системой

### 6.1. Интеграция с аутентификацией

При переходе по ссылке `/invite/{token}` нужно:

1. Проверить, аутентифицирован ли пользователь
2. Проверить, совпадает ли email из токена с email аутентифицированного пользователя
3. Принять решение:

```go
func handleInvite(token string, currentUser *User) (*Invitation, error) {
    inv, err := getInvitationByToken(token)
    if err != nil {
        return nil, err
    }
    
    // Проверка статуса и срока
    if inv.status != "PENDING" || inv.expiresAt.Before(time.Now()) {
        return nil, errors.New("invitation expired or already used")
    }
    
    // Случай 1: Пользователь аутентифицирован
    if currentUser != nil {
        if currentUser.Email != inv.email {
            // Ошибка: приглашение на другой email
            return nil, errors.New("invitation is for different email")
        }
        // Добавляем в workspace и возвращаем успех
        return acceptInvitation(inv, currentUser.ID)
    }
    
    // Случай 2: Пользователь не аутентифицирован
    // Проверяем, существует ли пользователь с таким email
    existingUser := findUserByEmail(inv.email)
    if existingUser != nil {
        // Отправляем на логин
        return nil, &AuthRequiredError{
            Email:      inv.email,
            UserExists: true,
            Token:      token,
        }
    }
    
    // Отправляем на регистрацию
    return nil, &AuthRequiredError{
        Email:      inv.email,
        UserExists: false,
        Token:      token,
    }
}
```

### 6.2. Интеграция с системой ролей

При принятии приглашения нужно выполнить те же действия, что и при ручном назначении ролей:

```go
func acceptInvitation(inv *Invitation, userID string) error {
    // Транзакция
    tx := db.Begin()
    
    // 1. Добавляем пользователя в workspace (user_workspaces)
    userWorkspace := &UserWorkspace{
        UserID:      userID,
        WorkspaceID: inv.WorkspaceID,
        Role:        inv.SystemRole,
    }
    tx.Create(userWorkspace)
    
    // 2. Назначаем кастомные роли через PermissionService
    for _, roleID := range inv.RoleIDs {
        permissionService.AssignRole(userID, roleID, inv.WorkspaceID, inv.InvitedBy)
    }
    
    // 3. Назначаем индивидуальные права
    for _, perm := range inv.Permissions {
        permissionService.GrantPermission(userID, inv.WorkspaceID, perm.PermissionID, inv.InvitedBy, perm.ExpiresAt)
    }
    
    // 4. Обновляем статус приглашения
    inv.Status = "ACCEPTED"
    inv.AcceptedAt = time.Now()
    tx.Save(inv)
    
    tx.Commit()
    return nil
}
```

### 6.3. Интеграция с email

Создать интерфейс для отправки email:

```go
type EmailService interface {
    SendInviteEmail(to string, inviteLink string, workspaceName string, invitedByName string) error
}
```

**Пример письма:**

```
Тема: Вас пригласили в workspace "Ромашка"

Здравствуйте!

Admin Name приглашает вас присоединиться к workspace "Ромашка" 
в системе ERP.

Ваша роль: MEMBER
Ссылка действительна до: 10 марта 2026

Перейдите по ссылке, чтобы принять приглашение:
https://app.com/invite/abc123def456

Если у вас нет аккаунта, вы сможете создать его при переходе по ссылке.
```

---

## 7. Обработка пограничных случаев

### 7.1. Приглашение уже существующего участника

**Проблема:** ADMIN пытается пригласить email, который уже есть в workspace

**Решение:** При создании приглашения проверять `user_workspaces`:

```sql
SELECT 1 FROM user_workspaces 
WHERE workspace_id = $1 AND user_id IN (
    SELECT id FROM users WHERE email = $2
)
```

**Ответ:** `400 Bad Request` с сообщением "User already in workspace"

### 7.2. Повторная отправка приглашения

**Проблема:** ADMIN хочет отправить приглашение повторно (например, если письмо потерялось)

**Решение:** Сделать возможность создать новое приглашение на тот же email (старое при этом автоматически отменяется или помечается как заменённое)

### 7.3. Истечение срока действия

**Проблема:** Пользователь перешёл по ссылке после истечения срока

**Решение:** 
- При проверке токена сравнивать `expires_at` с текущим временем
- Если истекло, возвращать `410 Gone`
- Можно добавить фоновый процесс, который помечает истекшие приглашения как `EXPIRED`

### 7.4. Конфликт email при регистрации

**Проблема:** Пользователь пытается зарегистрироваться с email, на который уже есть приглашение, но регистрируется с другим email

**Решение:** При регистрации проверять, есть ли приглашение на указанный email. Если есть, автоматически принимать его после успешной регистрации.

### 7.5. Приглашение и SSO

**Проблема:** Компания использует SSO, пользователь должен входить через корпоративный аккаунт

**Решение:** 
- При определении, что пользователь не существует, проверять, настроен ли SSO для домена email
- Если да, перенаправлять на SSO-логин, а не на обычную регистрацию

---

## 8. Детальная декомпозиция задач

### Этап 1: Подготовка (1 день)

- [ ] Создать миграцию для таблицы `invitations`
- [ ] Добавить индексы
- [ ] Написать откат миграции

### Этап 2: Базовая функциональность (3 дня)

- [ ] Создать модель `Invitation` в коде (GORM)
- [ ] Реализовать генерацию уникального токена
- [ ] Создать репозиторий для работы с приглашениями (CRUD + поиск по токену)
- [ ] Реализовать проверку срока действия

### Этап 3: API для создания приглашений (2 дня)

- [ ] Реализовать эндпоинт `POST /workspaces/{workspaceId}/invitations`
- [ ] Валидация входных данных (email, systemRole)
- [ ] Проверка прав (только ADMIN/OWNER)
- [ ] Проверка, что пользователь уже не в workspace
- [ ] Интеграция с PermissionService для валидации roleIds и permissions

### Этап 4: API для управления приглашениями (2 дня)

- [ ] Реализовать `GET /workspaces/{workspaceId}/invitations` (с пагинацией и фильтрацией)
- [ ] Реализовать `DELETE /workspaces/{workspaceId}/invitations/{invitationId}` (отзыв)
- [ ] Реализовать массовое создание `POST /invitations/bulk`

### Этап 5: Публичные эндпоинты (2 дня)

- [ ] Реализовать `GET /public/invitations/{token}` (информация о приглашении)
- [ ] Реализовать `POST /public/invitations/{token}/accept` (принятие)

### Этап 6: Интеграция с аутентификацией (2 дня)

- [ ] Интеграция с текущим пользователем (из контекста)
- [ ] Логика определения: аутентифицирован / не аутентифицирован
- [ ] Перенаправление на логин с параметрами
- [ ] Перенаправление на регистрацию с pre-filled email

### Этап 7: Интеграция с системой ролей (2 дня)

- [ ] При принятии приглашения вызывать `PermissionService.AssignRole` для каждой роли
- [ ] При принятии приглашения вызывать `PermissionService.GrantPermission` для каждого права
- [ ] Создание записи в `user_workspaces` с системной ролью

### Этап 8: Email уведомления (2 дня)

- [ ] Создать интерфейс `EmailService`
- [ ] Реализовать заглушку (логирование в консоль) для разработки
- [ ] Интегрировать с реальным почтовым сервисом (SendGrid, AWS SES, etc.)
- [ ] Создать шаблон письма

### Этап 9: Фоновые задачи (1 день)

- [ ] Создать задачу для автоматического проставления статуса `EXPIRED` у истекших приглашений
- [ ] Запускать по расписанию (cron, раз в час)

### Этап 10: Тестирование (2 дня)

- [ ] Unit-тесты для всех ключевых сценариев
- [ ] Интеграционные тесты для API
- [ ] Тестирование пограничных случаев

---

## 9. Критерии приемки

### 9.1. Функциональные критерии

- [ ] ADMIN может создать приглашение для существующего пользователя
- [ ] ADMIN может создать приглашение для нового пользователя
- [ ] Приглашение содержит системную роль (MEMBER/GUEST)
- [ ] Приглашение может содержать кастомные роли
- [ ] Приглашение может содержать индивидуальные права с истечением срока
- [ ] ADMIN может просматривать список активных приглашений
- [ ] ADMIN может отозвать приглашение
- [ ] При переходе по ссылке существующий пользователь добавляется в workspace
- [ ] При переходе по ссылке новый пользователь направляется на регистрацию
- [ ] После регистрации новый пользователь автоматически добавляется в workspace
- [ ] Приглашение не работает после истечения срока
- [ ] Приглашение не работает после отзыва
- [ ] Нельзя пригласить пользователя, уже состоящего в workspace

### 9.2. Безопасность

- [ ] Токены приглашений уникальны и не поддаются подбору (достаточная энтропия)
- [ ] Срок действия токена ограничен
- [ ] Принятое приглашение нельзя принять повторно
- [ ] Отозванное приглашение нельзя принять
- [ ] Только ADMIN/OWNER могут создавать/отзывать приглашения

### 9.3. Производительность

- [ ] Создание приглашения < 100ms
- [ ] Проверка токена < 50ms
- [ ] Индексы обеспечивают быстрый поиск по токену и workspace_id

---

## 10. Оценка сроков

| Этап | Задачи | Время |
|------|--------|-------|
| **Этап 1** | Миграция таблицы | 1 день |
| **Этап 2** | Базовая функциональность | 3 дня |
| **Этап 3** | API создания | 2 дня |
| **Этап 4** | API управления | 2 дня |
| **Этап 5** | Публичные эндпоинты | 2 дня |
| **Этап 6** | Интеграция с auth | 2 дня |
| **Этап 7** | Интеграция с ролями | 2 дня |
| **Этап 8** | Email уведомления | 2 дня |
| **Этап 9** | Фоновые задачи | 1 день |
| **Этап 10** | Тестирование | 2 дня |
| **Резерв** | Непредвиденное | 2 дня |
| **ИТОГО** | | **19 рабочих дней** (~4 недели) |

---

## 11. Резюме

### Ключевое решение: Новые пользователи создаются

Приглашение работает как **пред-регистрация**. Если пользователя нет в системе:
- Он получает email со ссылкой
- Переходит по ссылке
- Видит страницу регистрации с уже заполненным email
- После регистрации автоматически попадает в workspace

Это стандартный подход, который:
- Минимизирует трение для нового пользователя
- Не требует ручного создания аккаунтов администратором
- Обеспечивает бесшовный опыт

### Интеграция с существующей системой

Всё, что мы делаем, опирается на уже реализованную систему ролей:
- При создании приглашения используем те же `roleIds` и `permissions`
- При принятии вызываем те же методы `PermissionService`
- Статусы приглашений не пересекаются с бизнес-данными

### Следующие шаги

После реализации приглашений система станет полностью самодостаточной:
- Есть управление пользователями (добавление/удаление)
- Есть управление ролями и правами
- Есть процесс онбординга новых участников

---

**Документ подготовлен:** Март 2026  
**Версия:** 1.0  
**Статус:** Готов к реализации
