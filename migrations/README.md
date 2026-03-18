# Миграции

## Текущие миграции (разработка)

Используются инкрементальные миграции `000001` … `000030` + `constraints/`.

```bash
go run ./cmd/migrate/main.go up
```

---

## ⚠️ ВАЖНО: синхронизация с clean_baseline

**При любых изменениях в текущих миграциях** (`*.up.sql`, `constraints/*.sql`) необходимо **актуализировать** папку `clean_baseline/`.

`clean_baseline/` — миграции для **потенциального продакшена** (чистая БД с нуля). Должны соответствовать финальному состоянию схемы после применения всех текущих миграций.

### Чеклист при изменении миграций

- [ ] Внести изменения в соответствующий `*.up.sql`
- [ ] Обновить `docs/ALL_MIGRATIONS_UP.sql` (если меняется порядок/содержимое)
- [ ] **Актуализировать** `clean_baseline/` — соответствующий файл по сущности (см. `clean_baseline/README.md`)

### Соответствие: текущие миграции → clean_baseline

| Текущие | clean_baseline |
|---------|----------------|
| 000001 | 001_infra_request_logs |
| 000002 | 002_auth_users |
| 000003 + 000010 | 007_habits |
| 000004 + 000011 | 008_habit_completions |
| 000005 | 003_core_workspaces |
| 000006 | 004_core_user_workspaces |
| 000007 | 005_auth_user_preferences |
| 000008 | 009_habit_history |
| 000009 | 011_activities |
| 000011 | 010_habit_versions |
| 000012 + 000013 | 006_modules_system |
| 000014 | 012–014 shared, 015 journal |
| 000015 | 015_journal |
| 000016 | 016–017 crm |
| 000017 | 020_crm_activities |
| 000018 | 021_projects |
| 000019–000021 | 006 (modules), 022 |
| 000022 | 022_permissions |
| 000023 | 023_auth_registration |
| 000025 | 024_auth_invitations |
| 000026 | 023 (invite_token) |
| 000027 | 025_notifications |
| 000028 + 000030 | 026_tasks |
| 000029 + 000031 | 027_task_comments |
| 000032 | 006 (modules trial, tasks core) |
