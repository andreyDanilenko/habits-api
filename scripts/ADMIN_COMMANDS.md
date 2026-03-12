# Быстрые команды для администрирования пользователей

Подключение к БД (подставь свои значения из .env):

```bash
# Локально / Docker
psql -h localhost -U postgres -d auth_service

# Или через docker-compose
docker exec -it <postgres_container> psql -U postgres -d auth_service
```

---

## 1. Посмотреть всех пользователей

```sql
SELECT id, email, name, role, status, created_at FROM users ORDER BY email;
```

---

## 2. Сменить роль пользователя

```sql
-- Сделать пользователя админом (по email)
UPDATE users SET role = 'ADMIN', updated_at = NOW() WHERE email = 'user@example.com';

-- Вернуть в обычного пользователя
UPDATE users SET role = 'USER', updated_at = NOW() WHERE email = 'user@example.com';
```

---

## 3. Удалить пользователя (soft delete)

```sql
UPDATE users SET status = 'DELETED', updated_at = NOW() WHERE email = 'user@example.com';
```

Восстановить:

```sql
UPDATE users SET status = 'ACTIVE', updated_at = NOW() WHERE email = 'user@example.com';
```

---

## 4. Сменить пароль пользователю

Пароль хранится как bcrypt-хеш. Сначала сгенерируй хеш:

```bash
cd backend
go run ./scripts/hash-password 'MyNewPassword123!'
# или: make hash-password PWD='MyNewPassword123!'
```

Скопируй вывод (длинная строка вида `$2a$10$...`) и выполни:

```sql
UPDATE users 
SET password = '$2a$10$ТВОЙ_ХЕШ_ИЗ_КОМАНДЫ_ВЫШЕ', updated_at = NOW() 
WHERE email = 'user@example.com';
```

**Важно:** пароль должен соответствовать формату: мин 8 символов, буквы, цифры, спецсимволы (@$!%*#?&).

---

## 5. Забанить / разбанить

```sql
-- Забанить
UPDATE users SET status = 'BANNED', updated_at = NOW() WHERE email = 'user@example.com';

-- Разбанить
UPDATE users SET status = 'ACTIVE', updated_at = NOW() WHERE email = 'user@example.com';
```

---

## Одной строкой (примеры)

```bash
# Сменить роль
psql -h localhost -U postgres -d auth_service -c "UPDATE users SET role = 'ADMIN' WHERE email = 'admin@test.com';"

# Soft delete
psql -h localhost -U postgres -d auth_service -c "UPDATE users SET status = 'DELETED' WHERE email = 'old@test.com';"

# Сменить пароль (сгенерируй хеш и подставь в SQL)
# cd backend && go run ./scripts/hash-password 'NewPass123!'
# psql ... -c "UPDATE users SET password = '<хеш>' WHERE email = 'user@test.com';"
```
