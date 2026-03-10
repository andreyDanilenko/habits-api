# Auth — бэклог и чеклист

## ✅ Уже сделано

- [x] Login / Register / Logout
- [x] JWT в cookie (access_token), HttpOnly, Secure, SameSite (Lax/None по окружению)
- [x] Rate limit по IP на `/auth/*` (10 попыток/мин, in-memory)
- [x] Пароли — bcrypt, единое сообщение при неверных данных (нет user enumeration)
- [x] Constant-time проверка пароля при несуществующем user (dummy hash — защита от timing attack)
- [x] Валидация запросов (email, min password length, name)
- [x] CORS, заголовок X-CSRF-Token в Allow-Headers
- [x] Эндпоинт GET /me для текущего пользователя

## 📋 Бэклог (что доделать)

- [ ] **Refresh token** — реализовать `/auth/refresh` (сейчас 501)
- [ ] **Account lockout** — блокировка аккаунта после 5–10 неудачных попыток (таблица `login_attempts`)
- [ ] **Конфиг rate limit** — вынести лимит и окно (например `AUTH_RATE_LIMIT_PER_MIN`) в конфиг
- [ ] **Redis rate limit** — для нескольких инстансов использовать Redis вместо in-memory лимитера
- [ ] **Задержка при неудачном логине** — небольшая задержка перед ответом (усложняет перебор)
- [ ] **CAPTCHA** (опционально) — после N неудач требовать CAPTCHA на login/register

## Связанные документы

- `docs/SECURITY_AUTH_ANALYSIS.md` — анализ SQL-инъекций и брутфорса
- `docs/BACKLOG/auth-ideal-security.md` — задача «идеальная авторизация» (безопасность, DDoS, взлом)
