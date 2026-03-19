# Seed: задачи (модуль tasks)

Сид создаёт тестовые задачи в указанном workspace для проверки модуля задач.

## Запуск

```bash
cd backend
WORKSPACE_ID=<uuid> USER_ID=<uuid> go run ./cmd/seed_tasks
```

### Пример (UUID из seed_users_roles)

```bash
# Workspace "Демо", пользователь owner
WORKSPACE_ID=b2222222-2222-2222-2222-222222222201 USER_ID=a1111111-1111-1111-1111-111111111101 go run ./cmd/seed_tasks
```

## Переменные окружения

| Переменная    | Обязательная | Описание                                      |
|---------------|--------------|-----------------------------------------------|
| WORKSPACE_ID  | Да           | UUID workspace, в который добавляются задачи  |
| USER_ID       | Да           | UUID пользователя (created_by, должен быть в workspace) |
| ASSIGNEE_ID   | Нет          | UUID исполнителя по умолчанию (если не задан — USER_ID) |

## Созданные данные

- **15 задач** разных типов (task, bug, feature, meeting, call, email, lunch)
- Разные приоритеты (low, medium, high, critical) и статусы (pending, in_progress, completed, cancelled)
- **3 подзадачи** у задачи «Настроить CI/CD пайплайн»
- **1 связь** blocked_by (баг блокируется задачей CI/CD)
- **2 комментария** к первой задаче
- Теги: devops, frontend, crm, важно и др.

## Требования

1. Запустить `seed_users_roles` до seed_tasks
2. USER_ID должен быть участником workspace (user_workspaces)
3. Модуль tasks должен быть активен в workspace (core-модуль, включается автоматически)
