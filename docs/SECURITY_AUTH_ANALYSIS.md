# Анализ безопасности: SQL-инъекции и брутфорс

## 1. SQL-инъекции

### ✅ Что уже сделано правильно

**User Repository** (`internal/repository/user/repository.go`):
- Все запросы используют **параметризованные запросы** (`$1`, `$2`, ...)
- Пользовательский ввод (email, id) передаётся как аргументы, не конкатенируется в SQL

```go
// Пример безопасного запроса
query := `SELECT ... FROM users WHERE email = $1 AND status = 'ACTIVE'`
err := r.db.QueryRowContext(ctx, query, email).Scan(...)
```

**CRM Repositories** (deal, company, contact):
- Динамические условия строятся через `fmt.Sprintf` только для **плейсхолдеров** (`$%d`), значения идут в `args`
- `mapSortByToColumn()` и `sortOrder()` — **whitelist**: только разрешённые значения попадают в SQL

```go
// Безопасно: col берётся из whitelist, sortOrder — "ASC" или "DESC"
order = "ORDER BY " + col + " " + sortOrder(opts.SortOrder)
```

### ⚠️ Правила на будущее

1. **Никогда** не конкатенировать пользовательский ввод в SQL:
   ```go
   // ПЛОХО
   query := "SELECT * FROM users WHERE email = '" + email + "'"
   ```

2. **Всегда** использовать плейсхолдеры:
   ```go
   // ХОРОШО
   query := "SELECT * FROM users WHERE email = $1"
   r.db.QueryRowContext(ctx, query, email)
   ```

3. Для **динамических имён колонок/таблиц** — только whitelist:
   ```go
   allowed := map[string]string{"name": "name", "createdAt": "created_at"}
   if col, ok := allowed[userInput]; ok {
       order = "ORDER BY " + col
   }
   ```

---

## 2. Защита от брутфорса

### Текущее состояние

- ❌ Нет rate limiting на `/auth/login` и `/auth/register`
- ❌ Нет блокировки после N неудачных попыток
- ✅ Пароли хешируются через bcrypt
- ✅ Одинаковое сообщение при неверном email/пароле (нет утечки "пользователь не найден")

### Рекомендуемые меры

| Мера | Описание |
|------|----------|
| Rate limiting по IP | Ограничить число запросов на login/register с одного IP (например, 10/мин) |
| Account lockout | После 5–10 неудачных попыток — временная блокировка аккаунта |
| Задержка при неудаче | Небольшая задержка перед ответом (усложняет перебор) |
| CAPTCHA | После нескольких неудач — требовать CAPTCHA (опционально) |

---

## 3. User enumeration (timing attack)

При несуществующем email ответ возвращается быстрее (нет вызова bcrypt).
При существующем — дольше (сравнение хеша).

**Рекомендация**: всегда выполнять сравнение пароля с «заглушкой», если пользователь не найден:

```go
// Вместо немедленного return при user == nil
// выполнить password.Check с dummy hash, чтобы время ответа было одинаковым
```

---

## 4. Реализованные меры

| Мера | Файл | Описание |
|------|------|----------|
| Rate limiting | `internal/middleware/rate_limit.go` | 10 попыток/мин на IP для `/auth/*` |
| Timing attack fix | `internal/service/auth/service.go` | При несуществующем user — сравнение с dummy hash |
| Dummy hash | `pkg/password/password.go` | `DummyHash()` для constant-time проверки |

### Настройка rate limit

В `internal/di/container.go`:
```go
authRateLimiter := middleware.NewAuthRateLimiter(10, time.Minute)
authGroup.Use(authRateLimiter.Middleware(c.Responder))
```

Можно вынести 10 и time.Minute в конфиг.
