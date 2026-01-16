# Шпаргалка по PostgreSQL

## Подключение к БД
```bash
# Локальное подключение
psql -U username -d dbname -h localhost -p 5432

# В контейнере Docker
docker exec -it postgres_container psql -U username -d dbname

# С паролем
docker exec -it postgres_container psql -U postgres -W

# Через bash в контейнере, затем psql
docker exec -it postgres_container bash
psql -U username -d dbname
```

## Таблицы
```sql
-- Список всех таблиц
\dt
-- Или
SELECT table_name FROM information_schema.tables WHERE table_schema = 'public';

-- Структура таблицы
\d table_name
-- Или
SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'table_name';
```

## Миграции (golang-migrate)

### Обычное добавление поля
```sql
-- 003_add_new_field.up.sql
ALTER TABLE table_name ADD COLUMN new_field VARCHAR(100);
```

### Проблема с schema_migrations
```sql
-- Просмотр примененных миграций
SELECT * FROM schema_migrations;

-- Удаление записи о миграции (чтобы перезапустить)
DELETE FROM schema_migrations WHERE version = 2;

-- Полный сброс
DROP TABLE IF EXISTS schema_migrations;
```

### Принудительный сброс в коде
```go
// В RunMigrations():
conn.Exec("DROP TABLE IF EXISTS schema_migrations")
```

## Утилиты
```sql
-- Текущая БД
SELECT current_database();

-- Список всех БД
\l

-- Смена БД
\c dbname

-- Показать все индексы таблицы
\di table_name
```

## Быстрые команды
```bash
# Удалить и создать БД заново (DEV!)
docker exec postgres_container psql -U postgres -c "DROP DATABASE dbname; CREATE DATABASE dbname;"

# Принудительно установить версию миграции
migrate -database "postgres://..." -path ./migrations force 2

# Копировать дамп в контейнер
docker cp backup.sql postgres_container:/tmp/
docker exec postgres_container psql -U username -d dbname -f /tmp/backup.sql
```

## Поиск и отладка
```sql
-- Есть ли таблица?
SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users');

-- Есть ли столбец?
SELECT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'avatar_url');

-- Посмотреть все подключения
SELECT * FROM pg_stat_activity;

-- Размер таблицы
SELECT pg_size_pretty(pg_total_relation_size('users'));
```

## Docker специфичное
```bash
# Посмотреть логи контейнера
docker logs postgres_container -f

# Зайти в контейнер и исследовать файлы
docker exec -it postgres_container bash
ls -la /var/lib/postgresql/data/

# Создать дамп БД из контейнера
docker exec postgres_container pg_dump -U username dbname > backup.sql

# Восстановить дамп в контейнер
cat backup.sql | docker exec -i postgres_container psql -U username -d dbname
```

**Простое правило:** Если миграция не применяется → проверь `schema_migrations` → удали запись → перезапусти приложение.
