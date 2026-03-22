# Документация подсистем (SYSTEM)

Один вход: **что читать в первую очередь** и где лежит история проектирования без дублей в корне папки.

---

## Актуальные документы (порядок чтения)

| Тема | Файл | Зачем |
|------|------|--------|
| **Права, роли, API, фронт, Casbin — полная операционная модель** | [PERMISSIONS/README_PERMISSIONS_ROLES.md](./PERMISSIONS/README_PERMISSIONS_ROLES.md) | Главный справочник по реализованной системе RBAC. |
| **RBAC + видимость строк (data scope), `role_object_scopes`** | [ROLES_PERMISSIONS_AND_DATA_SCOPE.md](./ROLES_PERMISSIONS_AND_DATA_SCOPE.md) | Второй слой доступа поверх Casbin; не дублирует README, а дополняет. |
| **Краткая архитектура бэкенда ролей** | [ROLES_BACKEND_ARCHITECTURE.md](./ROLES_BACKEND_ARCHITECTURE.md) | Сжатый обзор таблиц и Casbin; детали — в README выше и в DATA_SCOPE. |
| **Глобальная роль vs роль в workspace** | [ROLES_GLOBAL_VS_WORKSPACE.md](./ROLES_GLOBAL_VS_WORKSPACE.md) | `users.role` (ADMIN/USER) и `systemRole` в workspace. |
| **Угрозы и меры (роли, наследование, индивидуальные права)** | [ROLES_SECURITY_CONSIDERATIONS.md](./ROLES_SECURITY_CONSIDERATIONS.md) | Риски и рекомендуемые проверки в коде/API. |
| **Спецификация бэкенда (контракт, middleware, масштабирование)** | [PERMISSIONS/SPEC_ROLE_BACK.md](./PERMISSIONS/SPEC_ROLE_BACK.md) | Техническая спецификация; перекрёстные ссылки на архивные постановки. |
| **Итог внедрения RBAC по этапам (март 2026)** | [PERMISSION_REPORT.PART_TOTAL.md](./PERMISSION_REPORT.PART_TOTAL.md) | Сводный отчёт: миграции, DI, сервисы, что осталось включить. |
| **SQL-инъекции, брутфорс (общий раздел безопасности)** | [SECURITY_AUTH_ANALYSIS.md](./SECURITY_AUTH_ANALYSIS.md) | Не про роли; про паттерны в репозиториях. |
| **План модуля активности** | [ACTIVITY_MODULE_PLAN.md](./ACTIVITY_MODULE_PLAN.md) | Продуктовый/технический план. |
| **План микрофронтенда** | [FRONTEND_MICROFRONTEND_PLAN.md](./FRONTEND_MICROFRONTEND_PLAN.md) | Архитектурный план фронта. |

---

## Архив (`_archive/`)

Черновики, поэтапные отчёты и длинные анализы **сохранены** — см. [_archive/README.md](./_archive/README.md). На них ссылаются `PERMISSION_REPORT.PART_TOTAL.md` и `SPEC_ROLE_BACK.md` там, где нужна первоисточная детализация.

---

## Что убрали из «корня» SYSTEM

Раньше в одном каталоге лежали и **актуальные** гайды, и **v1/v2/v3 постановки**, и **PART_1/2/3**, и чеклист **TOTAL_STATE_ROLE** — из-за этого было неочевидно, что является источником истины. Сейчас в корне остались только документы из таблицы выше; остальное перенесено в `_archive/` без удаления текста.
