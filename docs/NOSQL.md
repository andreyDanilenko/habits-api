# 📚 ПОЛНАЯ ШПАРГАЛКА ПО NoSQL

## 🎯 ЧТО ТАКОЕ NoSQL?

**NoSQL (Not Only SQL)** — базы данных, которые не используют реляционную модель и SQL как основной язык запросов. Созданы для решения задач, где SQL-базы неэффективны:

| Характеристика | SQL | NoSQL |
|---------------|-----|-------|
| **Схема данных** | Фиксированная (schema-on-write) | Гибкая (schema-on-read) |
| **Масштабирование** | Вертикальное (мощнее сервер) | Горизонтальное (больше серверов) |
| **Транзакции** | ACID | BASE (Basically Available, Soft state, Eventual consistency) |
| **Модель данных** | Таблицы, строки, столбцы | Ключ-значение, документы, колонки, графы |
| **Язык запросов** | SQL (стандартизирован) | API, свой язык (разный у каждой БД) |
| **Целостность** | Высокая (foreign keys, constraints) | Низкая (все на стороне приложения) |

---

## 🗂️ ТИПЫ NoSQL БАЗ ДАННЫХ

```
┌─────────────────────────────────────────────────────────────────┐
│                         NoSQL                                   │
├────────────┬───────────────┬──────────────┬───────────────────┤
│ КЛЮЧ-      │ ДОКУМЕНТНЫЕ   │ КОЛОНОЧНЫЕ   │ ГРАФОВЫЕ         │
│ ЗНАЧЕНИЕ   │               │              │                  │
├────────────┼───────────────┼──────────────┼───────────────────┤
│ Redis      │ MongoDB      │ Cassandra    │ Neo4j            │
│ Memcached  │ CouchDB      │ HBase        │ Amazon Neptune   │
│ Riak       │ Firebase     │ Google BigTable│ ArangoDB       │
│ Amazon ElastiCache│ Cosmos DB│ ScyllaDB   │ OrientDB        │
└────────────┴───────────────┴──────────────┴───────────────────┘
```

---

# 🟥 1. REDIS — КЛЮЧ-ЗНАЧЕНИЕ (IN-MEMORY)

## 📦 ОСНОВНЫЕ КОМАНДЫ

### 🔑 Работа с ключами
```bash
SET key value                    # установить значение
SET key value EX 10             # с истечением через 10 секунд
SET key value NX                # только если ключа не существует
SET key value XX                # только если ключ существует
GET key                         # получить значение
DEL key                         # удалить ключ
EXISTS key                      # проверить существование (1/0)
EXPIRE key 60                   # установить время жизни (сек)
TTL key                         # узнать время жизни
PERSIST key                     # убрать время жизни
KEYS pattern                    # найти ключи (не для продакшена!)
SCAN cursor                     # итератор по ключам
TYPE key                        # тип значения
RENAME key newkey              # переименовать
RANDOMKEY                       # случайный ключ
```

### 🔢 Строки (Strings)
```bash
SET name "Ivan"                 # строка
GET name                        # "Ivan"
APPEND name " Petrov"           # добавить в конец
STRLEN name                     # длина строки

INCR counter                    # +1 (число)
INCRBY counter 10              # +10
DECR counter                    # -1
DECRBY counter 5               # -5
INCRBYFLOAT price 1.5          # +1.5

SETEX key 60 "value"           # SET + EXPIRE
SETNX key "value"              # SET if Not eXists
MSET key1 val1 key2 val2       # множественная установка
MGET key1 key2                 # множественное получение
GETSET key newvalue           # получить старое, установить новое
```

### 📋 Списки (Lists) — упорядоченные, дубликаты
```bash
LPUSH users "ivan"             # добавить в начало
RPUSH users "petr"             # добавить в конец
LPOP users                     # удалить и получить первый
RPOP users                     # удалить и получить последний
LLEN users                     # длина списка
LRANGE users 0 -1             # все элементы
LINDEX users 0                # элемент по индексу
LSET users 0 "new_name"       # установить по индексу
LINSERT users BEFORE "ivan" "masha"  # вставить перед
LREM users 2 "ivan"           # удалить 2 вхождения
LTRIM users 0 2              # обрезать список
RPOPLPUSH list1 list2        # из list1 в list2
```

### 🎯 Множества (Sets) — неупорядоченные, уникальные
```bash
SADD roles "admin"             # добавить
SREM roles "admin"            # удалить
SMEMBERS roles                # все элементы
SISMEMBER roles "admin"       # проверить наличие (1/0)
SCARD roles                   # количество элементов
SPOP roles                    # удалить случайный
SRANDMEMBER roles 2          # 2 случайных без удаления

SINTER set1 set2             # пересечение
SUNION set1 set2             # объединение
SDIFF set1 set2             # разность
SINTERSTORE newset set1 set2 # сохранить пересечение
SMOVE src dest member        # переместить
```

### 🧩 Упорядоченные множества (Sorted Sets) — с весами
```bash
ZADD leaderboard 100 "ivan"   # добавить с весом
ZADD leaderboard 200 "petr"
ZRANGE leaderboard 0 -1       # по возрастанию веса
ZREVRANGE leaderboard 0 -1    # по убыванию веса
ZRANGEBYSCORE leaderboard 100 200  # по диапазону весов
ZSCORE leaderboard "ivan"     # вес элемента
ZRANK leaderboard "ivan"      # позиция (с 0)
ZREVRANK leaderboard "ivan"   # позиция с конца
ZCARD leaderboard            # количество
ZCOUNT leaderboard 100 200   # количество в диапазоне
ZREM leaderboard "ivan"      # удалить
ZINCRBY leaderboard 50 "ivan" # увеличить вес
ZUNIONSTORE out 2 z1 z2     # объединение
ZINTERSTORE out 2 z1 z2     # пересечение
```

### 🌸 Хэши (Hashes) — объекты
```bash
HSET user:1000 name "Ivan" age 30  # установить поля
HGET user:1000 name                # получить поле
HGETALL user:1000                 # все поля и значения
HMGET user:1000 name age          # несколько полей
HMSET user:1000 city "Moscow"     # несколько полей (устарело)
HDEL user:1000 age               # удалить поле
HEXISTS user:1000 name           # проверить поле
HKEYS user:1000                 # все ключи полей
HVALS user:1000                # все значения
HLEN user:1000                # количество полей
HINCRBY user:1000 age 1      # инкремент числа
HINCRBYFLOAT user:1000 score 1.5
HSETNX user:1000 phone "123" # если поле не существует
```

### 🗺️ Геопространственные индексы (Geo)
```bash
GEOADD cities 37.62 55.75 "moscow"  # долгота, широта, название
GEODIST cities "moscow" "spb" km    # расстояние
GEOPOS cities "moscow"             # координаты
GEOHASH cities "moscow"           # geohash
GEORADIUS cities 37.62 55.75 100 km  # точки в радиусе
GEORADIUSBYMEMBER cities "moscow" 100 km  # радиус от члена
```

### 📊 Битовые карты (Bitmaps)
```bash
SETBIT user:login 100 1           # установить бит на позиции 100
GETBIT user:login 100            # получить бит
BITCOUNT user:login             # количество единиц
BITOP AND result key1 key2      # побитовые операции
BITPOS user:login 1             # позиция первого 1
```

### 📈 HyperLogLog — уникальные подсчеты
```bash
PFADD visitors "ip1" "ip2"       # добавить элементы
PFCOUNT visitors                # приблизительное количество уникальных
PFMERGE dest source1 source2    # объединить
```

### 🔐 Pub/Sub — публикация/подписка
```bash
SUBSCRIBE channel               # подписаться на канал
PUBLISH channel "message"       # опубликовать сообщение
PSUBSCRIBE news*               # подписаться по паттерну
UNSUBSCRIBE channel            # отписаться
```

### 📦 Транзакции
```bash
MULTI                          # начало транзакции
SET key1 value1
SET key2 value2
EXEC                          # выполнить
DISCARD                       # отменить
WATCH key                     # следить за изменением
UNWATCH                       # перестать следить
```

### 📤 Lua-скрипты
```bash
EVAL "return redis.call('SET', KEYS[1], ARGV[1])" 1 key value
SCRIPT LOAD "return redis.call('SET', KEYS[1], ARGV[1])"
EVALSHA sha1 1 key value
SCRIPT EXISTS sha1
SCRIPT FLUSH
```

### 🧹 Администрирование
```bash
INFO                          # информация о сервере
INFO memory                  # информация о памяти
CONFIG GET *                # получить конфигурацию
CONFIG SET maxmemory 1gb   # установить конфигурацию
CLIENT LIST                 # список клиентов
CLIENT KILL ip:port        # убить клиента
DBSIZE                     # количество ключей
FLUSHDB                    # очистить текущую БД
FLUSHALL                   # очистить все БД
SAVE                       # сохранить на диск
BGSAVE                     # сохранить в фоне
LASTSAVE                   # время последнего сохранения
SHUTDOWN                   # остановить сервер
```

### 🚀 Примеры комплексных сценариев

```bash
# Кэширование с TTL
SET user:1000:profile '{"name":"Ivan","age":30}' EX 3600

# Счетчик просмотров
INCR video:123:views

# Рейтинг (топ-10)
ZINCRBY ratings 1 movie:123
ZREVRANGE ratings 0 9 WITHSCORES

# Сессии пользователей
SET session:abc123 '{"user_id":1000,"ip":"192.168.1.1"}' EX 86400

# Очередь задач
LPUSH tasks "email:user1000"
BRPOP tasks 0  # блокирующее чтение

# Уникальные посетители за день
PFADD stats:2024-01-15:visitors "ip1" "ip2" "ip1"

# Гео-поиск ближайших объектов
GEOADD restaurants 37.62 55.75 "cafe1"
GEORADIUS restaurants 37.62 55.75 1 km

# Блокировка (distributed lock)
SET lock:resource "process123" NX EX 10
DEL lock:resource
```

---

# 🟩 2. MONGODB — ДОКУМЕНТНАЯ БД

## 📦 ОСНОВНЫЕ ПОНЯТИЯ

| SQL | MongoDB |
|-----|---------|
| Database | Database |
| Table | Collection |
| Row | Document |
| Column | Field |
| Index | Index |
| JOIN | $lookup, embedded documents |
| FOREIGN KEY | Manual references / DBRef |
| TRANSACTION | ACID transactions (4.0+) |

## 🎯 CRUD ОПЕРАЦИИ

### 📄 CREATE — создание
```javascript
// Вставка одного документа
db.users.insertOne({
    name: "Иван Петров",
    email: "ivan@example.com",
    age: 30,
    tags: ["developer", "admin"],
    address: {
        city: "Москва",
        street: "Тверская",
        zip: "101000"
    },
    createdAt: new Date()
})

// Вставка нескольких документов
db.users.insertMany([
    { name: "Анна", age: 25 },
    { name: "Петр", age: 35, email: "petr@example.com" }
])

// insert (устаревший, но работает)
db.users.insert({ name: "Ольга" })
```

### 📖 READ — чтение
```javascript
// Все документы
db.users.find()
db.users.find().pretty()

// С условием
db.users.find({ age: 30 })
db.users.find({ name: "Иван Петров" })

// Операторы сравнения
db.users.find({ age: { $gt: 25 } })           // > 25
db.users.find({ age: { $gte: 30 } })          // >= 30
db.users.find({ age: { $lt: 40 } })           // < 40
db.users.find({ age: { $lte: 35 } })          // <= 35
db.users.find({ age: { $ne: 30 } })           // != 30
db.users.find({ age: { $in: [25, 30, 35] } }) // в списке
db.users.find({ age: { $nin: [20, 40] } })    // не в списке

// Логические операторы
db.users.find({ $and: [{ age: 30 }, { city: "Москва" }] })
db.users.find({ $or: [{ age: 25 }, { age: 35 }] })
db.users.find({ $not: { age: 30 } })
db.users.find({ $nor: [{ age: 30 }, { city: "СПб" }] })

// Работа с массивами
db.users.find({ tags: "admin" })                       // содержит
db.users.find({ tags: { $all: ["admin", "dev"] } })    // содержит все
db.users.find({ tags: { $size: 2 } })                  // длина массива
db.users.find({ "tags.0": "admin" })                   // по индексу

// Вложенные объекты
db.users.find({ "address.city": "Москва" })
db.users.find({ address: { city: "Москва", street: "Тверская" } })

// Регулярные выражения
db.users.find({ name: { $regex: /^Иван/ } })
db.users.find({ name: { $regex: "петр", $options: "i" } }) // i - регистронезав.

// Существование полей
db.users.find({ email: { $exists: true } })
db.users.find({ phone: { $exists: false } })

// Тип поля
db.users.find({ age: { $type: "int" } })
db.users.find({ age: { $type: "double" } })

// Проекция (какие поля вернуть)
db.users.find({}, { name: 1, email: 1, _id: 0 })      // 1 - включить, 0 - исключить
db.users.find({}, { address: 0, tags: 0 })

// Сортировка
db.users.find().sort({ age: 1 })     // 1 - по возрастанию
db.users.find().sort({ age: -1 })    // -1 - по убыванию
db.users.find().sort({ age: 1, name: -1 })

// Лимит и пропуск
db.users.find().limit(10)
db.users.find().skip(20).limit(10)

// Один документ
db.users.findOne({ email: "ivan@example.com" })
db.users.findById("507f1f77bcf86cd799439011")  // по _id
```

### 📝 UPDATE — обновление
```javascript
// Обновление одного документа
db.users.updateOne(
    { email: "ivan@example.com" },
    { $set: { age: 31, updatedAt: new Date() } }
)

// Обновление нескольких документов
db.users.updateMany(
    { city: "Москва" },
    { $set: { region: "Центр" } }
)

// Замена документа целиком
db.users.replaceOne(
    { email: "ivan@example.com" },
    { name: "Иван Иванов", email: "ivan@example.com", age: 32 }
)

// upsert - обновить или вставить
db.users.updateOne(
    { email: "new@example.com" },
    { $set: { name: "Новый", createdAt: new Date() } },
    { upsert: true }
)

// Операторы обновления
$set        // установить значение
$unset      // удалить поле
$inc        // инкремент (+=)
$mul        // умножение
$rename     // переименовать поле
$push       // добавить в массив
$pushAll    // добавить несколько в массив (устарел)
$pull       // удалить из массива по значению
$pullAll    // удалить несколько из массива
$pop        // удалить первый/последний из массива
$addToSet   // добавить в массив, если нет
$each       // с $push, $addToSet
$slice      // ограничить массив
$sort       // сортировка массива
$position   // вставить на позицию
$bit        // побитовые операции
$min        // установить, если меньше текущего
$max        // установить, если больше текущего
$currentDate// установить текущую дату

// Примеры
db.users.updateOne(
    { email: "ivan@example.com" },
    { 
        $inc: { age: 1 },
        $push: { tags: "senior" },
        $addToSet: { roles: "manager" },
        $currentDate: { lastModified: true }
    }
)

db.users.updateMany(
    { age: { $lt: 18 } },
    { $set: { status: "minor" } }
)
```

### 🗑️ DELETE — удаление
```javascript
// Удаление одного
db.users.deleteOne({ email: "old@example.com" })

// Удаление нескольких
db.users.deleteMany({ age: { $lt: 18 } })

// Удалить все документы (но не коллекцию)
db.users.deleteMany({})

// Удаление коллекции
db.users.drop()

// Удаление базы данных
db.dropDatabase()
```

## 📊 АГРЕГАЦИИ (Aggregation Pipeline)

```javascript
db.orders.aggregate([
    // Stage 1: фильтрация
    { $match: { status: "completed", date: { $gte: ISODate("2024-01-01") } } },
    
    // Stage 2: группировка
    { $group: {
        _id: { $dateToString: { format: "%Y-%m-%d", date: "$date" } },
        totalOrders: { $sum: 1 },
        totalRevenue: { $sum: "$amount" },
        avgOrder: { $avg: "$amount" },
        minOrder: { $min: "$amount" },
        maxOrder: { $max: "$amount" }
    }},
    
    // Stage 3: проекция
    { $project: {
        date: "$_id",
        totalOrders: 1,
        totalRevenue: { $round: ["$totalRevenue", 2] },
        avgOrder: { $round: ["$avgOrder", 2] },
        _id: 0
    }},
    
    // Stage 4: сортировка
    { $sort: { date: -1 } },
    
    // Stage 5: лимит
    { $limit: 30 }
])

// Операторы агрегации
$group: {
    _id: "$field",           // поле для группировки
    count: { $sum: 1 },      // счетчик
    sum: { $sum: "$price" }, // сумма
    avg: { $avg: "$price" }, // среднее
    min: { $min: "$price" }, // минимум
    max: { $max: "$price" }, // максимум
    first: { $first: "$price" }, // первое
    last: { $last: "$price" },   // последнее
    push: { $push: "$name" },    // собрать в массив
    addToSet: { $addToSet: "$category" } // уникальные в массив
}

// Другие stage-операторы
$project        // изменить форму документа
$match          // фильтрация
$limit          // ограничить
$skip           // пропустить
$sort           // сортировка
$unwind         // развернуть массив
$lookup         // LEFT JOIN с другой коллекцией
$lookup: {
    from: "users",
    localField: "userId",
    foreignField: "_id",
    as: "user"
}

$lookup: {      // JOIN с пайплайном
    from: "orders",
    let: { userId: "$_id" },
    pipeline: [
        { $match: { $expr: { $eq: ["$userId", "$$userId"] } } },
        { $limit: 5 }
    ],
    as: "recentOrders"
}

$lookup: {
    from: "books",
    localField: "author_id",
    foreignField: "author_id",
    as: "books"
}

$addFields      // добавить поля
$set            // установить поля (псевдоним $addFields)
$replaceRoot    // заменить корневой документ
$merge          // записать результат в коллекцию
$out            // записать результат в новую коллекцию
$facet          // несколько пайплайнов параллельно
$bucket         // группировка в сегменты
$bucketAuto     // автоматическая группировка
$graphLookup    // рекурсивный поиск по графу
$unionWith      // объединение с другой коллекцией
$count          // подсчет документов

// Пример: JOIN коллекций
db.orders.aggregate([
    {
        $lookup: {
            from: "users",
            localField: "userId",
            foreignField: "_id",
            as: "user"
        }
    },
    { $unwind: "$user" },
    {
        $project: {
            orderId: "$_id",
            userName: "$user.name",
            amount: 1,
            date: 1
        }
    }
])

// Пример: развернуть массив
db.products.aggregate([
    { $unwind: "$categories" },
    { $group: {
        _id: "$categories",
        products: { $push: "$name" }
    }}
])

// Пример: условная агрегация
db.sales.aggregate([
    { $group: {
        _id: "$region",
        total: { $sum: "$amount" },
        highValue: { 
            $sum: { 
                $cond: { 
                    if: { $gt: ["$amount", 1000] }, 
                    then: "$amount", 
                    else: 0 
                }
            }
        }
    }}
])
```

## 📇 ИНДЕКСЫ

```javascript
// Создание индексов
db.users.createIndex({ email: 1 })                 // одиночный, по возрастанию
db.users.createIndex({ email: -1 })                // по убыванию
db.users.createIndex({ name: 1, age: -1 })        // составной
db.users.createIndex({ email: 1 }, { unique: true }) // уникальный
db.users.createIndex({ location: "2dsphere" })     // геопространственный
db.users.createIndex({ tags: 1 })                  // по массиву
db.users.createIndex({ description: "text" })      // текстовый
db.users.createIndex({ "$**": "text" })            // текстовый по всем полям
db.users.createIndex({ createdAt: 1 }, { expireAfterSeconds: 86400 }) // TTL

// Частичные индексы
db.users.createIndex(
    { email: 1 },
    { 
        unique: true, 
        partialFilterExpression: { email: { $exists: true } }
    }
)

// Разреженный индекс (включает только документы с полем)
db.users.createIndex({ phone: 1 }, { sparse: true })

// Хешированный индекс (для шардинга)
db.users.createIndex({ _id: "hashed" })

// Получение информации
db.users.getIndexes()
db.users.totalIndexSize()
db.users.dropIndex("email_1")
db.users.dropIndexes()
```

## 📤 АДМИНИСТРИРОВАНИЕ

```javascript
// Информация о БД
show dbs
use mydb
db
show collections
db.getCollectionNames()
db.stats()
db.serverStatus()

// Создание пользователей
db.createUser({
    user: "admin",
    pwd: "password",
    roles: ["root", { role: "readWrite", db: "mydb" }]
})

// Бэкап и восстановление
// mongodump --db mydb --out /backup/
// mongorestore --db mydb /backup/mydb

// Мониторинг
db.currentOp()
db.killOp(opid)
db.setProfilingLevel(2)  // включить профилирование
db.system.profile.find().sort({ ts: -1 }).limit(5)
```

---

# 🟦 3. CASSANDRA — КОЛОНОЧНАЯ БД

## 📦 ОСНОВНЫЕ ПОНЯТИЯ

```
Keyspace → Table → Partition Key → Clustering Columns → Columns
(база)   (таблица) (обязательный)  (сортировка)        (данные)
```

## 🎯 CQL (Cassandra Query Language)

### 📄 CREATE — создание
```sql
-- Создание keyspace (базы)
CREATE KEYSPACE IF NOT EXISTS shop 
WITH REPLICATION = { 
    'class': 'SimpleStrategy', 
    'replication_factor': 3 
};

CREATE KEYSPACE IF NOT EXISTS analytics 
WITH REPLICATION = { 
    'class': 'NetworkTopologyStrategy', 
    'datacenter1': 3, 
    'datacenter2': 2 
};

USE shop;

-- Создание таблицы
CREATE TABLE users (
    user_id UUID PRIMARY KEY,
    email text,
    name text,
    age int,
    city text,
    created_at timestamp
);

-- Составной первичный ключ (partition key + clustering columns)
CREATE TABLE orders (
    user_id uuid,
    order_id uuid,
    order_date timestamp,
    total decimal,
    status text,
    items list<text>,
    PRIMARY KEY (user_id, order_date, order_id)
) WITH CLUSTERING ORDER BY (order_date DESC, order_id ASC);

-- Таблица с составным partition key
CREATE TABLE events (
    year int,
    month int,
    day int,
    event_id uuid,
    data text,
    PRIMARY KEY ((year, month, day), event_id)
);

-- Таблица со статическими полями
CREATE TABLE messages (
    thread_id uuid,
    message_id uuid,
    sender text,
    body text,
    subject text STATIC,  -- общее для всего раздела
    PRIMARY KEY (thread_id, message_id)
);
```

### 📖 READ — чтение
```sql
-- Базовые запросы
SELECT * FROM users;
SELECT name, email FROM users;
SELECT * FROM users WHERE user_id = 123e4567-e89b-12d3-a456-426614174000;
SELECT * FROM orders WHERE user_id = 123e4567-e89b-12d3-a456-426614174000;

-- Обязательно использовать partition key в WHERE!
-- ❌ НЕЛЬЗЯ: SELECT * FROM orders WHERE order_date > '2024-01-01';
-- ✅ МОЖНО:  SELECT * FROM orders WHERE user_id = ? AND order_date > '2024-01-01';

-- Разрешенные операторы: =, IN, >, >=, <, <=, CONTAINS, CONTAINS KEY
SELECT * FROM orders 
WHERE user_id = ? 
  AND order_date >= '2024-01-01' 
  AND order_date <= '2024-01-31';

SELECT * FROM users WHERE user_id IN (?, ?, ?);

SELECT * FROM products WHERE tags CONTAINS 'новинка';
SELECT * FROM products WHERE attributes CONTAINS KEY 'color';

-- Сортировка (только по clustering columns в заданном порядке)
SELECT * FROM orders 
WHERE user_id = ? 
ORDER BY order_date DESC, order_id ASC;

-- Лимит
SELECT * FROM events LIMIT 100;
SELECT * FROM events PER PARTITION LIMIT 10;  -- на раздел

-- Агрегации (ограниченно)
SELECT COUNT(*) FROM users;
SELECT MAX(order_date) FROM orders WHERE user_id = ?;
SELECT MIN(price), AVG(price) FROM products WHERE category = 'books';

-- ALLOW FILTERING (НЕ РЕКОМЕНДУЕТСЯ!)
SELECT * FROM users WHERE city = 'Москва' ALLOW FILTERING;
```

### 📝 INSERT/UPDATE — вставка/обновление
```sql
-- INSERT (upsert)
INSERT INTO users (user_id, email, name, age, city, created_at)
VALUES (
    uuid(),
    'ivan@example.com',
    'Иван Петров',
    30,
    'Москва',
    toTimestamp(now())
);

-- UPDATE (по сути то же самое)
UPDATE users 
SET age = 31, city = 'СПб' 
WHERE user_id = ?;

-- UPDATE с условием (lightweight transaction)
UPDATE users 
SET age = 31 
WHERE user_id = ? 
IF age = 30;  -- CAS (Compare-And-Swap)

-- Работа с коллекциями
UPDATE users 
SET emails = emails + {'work@example.com'}  -- добавить в set
WHERE user_id = ?;

UPDATE users 
SET phones['mobile'] = '+79991234567'  -- добавить в map
WHERE user_id = ?;

UPDATE users 
SET tags = tags - {'old'}  -- удалить из set
WHERE user_id = ?;

DELETE emails['spam'] FROM users WHERE user_id = ?;  -- удалить из map
```

### 🗑️ DELETE — удаление
```sql
-- Удалить всю строку
DELETE FROM users WHERE user_id = ?;

-- Удалить конкретную колонку
DELETE age FROM users WHERE user_id = ?;

-- Удалить элемент из коллекции
DELETE phones['old'] FROM users WHERE user_id = ?;
DELETE emails[2] FROM users WHERE user_id = ?;

-- Удалить с условием (LWТ)
DELETE FROM users WHERE user_id = ? IF EXISTS;
```

### 📊 ВТОРИЧНЫЕ ИНДЕКСЫ
```sql
-- Создание вторичного индекса
CREATE INDEX ON users(city);
CREATE INDEX ON users(age);
CREATE INDEX ON users(emails);  -- на коллекцию
CREATE INDEX ON users(KEYS(phones));  -- на ключи map
CREATE INDEX ON users(VALUES(phones));  -- на значения map

-- Материализованные представления
CREATE MATERIALIZED VIEW users_by_city AS
    SELECT * FROM users
    WHERE city IS NOT NULL AND user_id IS NOT NULL
    PRIMARY KEY (city, user_id)
    WITH CLUSTERING ORDER BY (user_id ASC);
```

## 🛠️ АДМИНИСТРИРОВАНИЕ

```sql
-- Информация
DESCRIBE KEYSPACES;
DESCRIBE KEYSPACE shop;
DESCRIBE TABLES;
DESCRIBE TABLE users;

-- Изменение таблицы
ALTER TABLE users ADD phone text;
ALTER TABLE users DROP age;
ALTER TABLE users RENAME city TO town;
ALTER TABLE users WITH gc_grace_seconds = 86400;

-- Настройки
ALTER KEYSPACE shop 
WITH REPLICATION = { 
    'class': 'SimpleStrategy', 
    'replication_factor': 5 
};

-- Удаление
DROP TABLE IF EXISTS old_table;
DROP KEYSPACE IF EXISTS test;

-- Настройка TTL (время жизни)
INSERT INTO users (user_id, email, session) 
VALUES (?, ?, ?) USING TTL 3600;  -- удалится через час

UPDATE users USING TTL 86400 
SET session = ? 
WHERE user_id = ?;
```

---

# 🟪 4. NEO4j — ГРАФОВАЯ БД

## 📦 ОСНОВНЫЕ ПОНЯТИЯ

```
Nodes    → вершины (сущности)
Relationships → ребра (связи)
Properties    → свойства
Labels        → метки (типы узлов)
Types         → типы связей
```

## 🎯 CYPHER — ЯЗЫК ЗАПРОСОВ

### 📄 CREATE — создание
```cypher
// Создание узлов
CREATE (ivan:User {name: 'Иван', age: 30, city: 'Москва'})
CREATE (book:Book {title: 'Мастер и Маргарита', year: 1967})
CREATE (spb:City {name: 'Санкт-Петербург'})

// Создание со связями
CREATE (ivan:User {name: 'Иван'})-[like:LIKES {rating: 5}]->(book:Book {title: 'Мастер и Маргарита'})
CREATE (ivan)-[:FRIEND]->(petr:User {name: 'Петр'})
CREATE (ivan)-[:LIVES_IN]->(msk:City {name: 'Москва'})

// С несколькими связями сразу
CREATE p = (ivan)-[:WRITTEN_BY]->(author:Author {name: 'Булгаков'})
CREATE p = (ivan)-[:BOUGHT {date: '2024-01-15', price: 500}]->(book)

// MERGE (найти или создать)
MERGE (ivan:User {name: 'Иван'})
ON CREATE SET ivan.created = timestamp()
ON MATCH SET ivan.lastSeen = timestamp()
MERGE (ivan)-[:FRIEND]->(petr:User {name: 'Петр'})

// Создание индексов
CREATE INDEX ON :User(name)
CREATE INDEX ON :Book(title)
CREATE CONSTRAINT ON (u:User) ASSERT u.email IS UNIQUE
```

### 📖 READ — чтение
```cypher
// Поиск узлов
MATCH (u:User) RETURN u
MATCH (u:User {name: 'Иван'}) RETURN u
MATCH (u:User) WHERE u.age > 25 RETURN u.name, u.age

// Поиск связей
MATCH (u:User)-[:LIKES]->(b:Book) RETURN u.name, b.title
MATCH (u:User)-[r:LIKES]->(b:Book) WHERE r.rating >= 4 RETURN u, b, r.rating

// Различные паттерны
MATCH (u:User)-[:FRIEND]->(friend)-[:LIKES]->(book)  // друзья и их книги
WHERE u.name = 'Иван'
RETURN friend.name, book.title

MATCH (u:User)-[:LIKES]->(b:Book)<-[:LIKES]-(other)  // общие интересы
WHERE u.name = 'Иван'
RETURN other.name, collect(b.title) AS commonBooks

// Опциональные связи
MATCH (u:User)
OPTIONAL MATCH (u)-[:LIVES_IN]->(city)
RETURN u.name, city.name

// Агрегация
MATCH (u:User)-[:LIKES]->(b:Book)
RETURN b.title, count(u) AS fans, avg(r.rating) AS avgRating
ORDER BY fans DESC
LIMIT 10

// Пути и длина
MATCH path = (ivan:User {name: 'Иван'})-[:FRIEND*1..3]->(friend)
RETURN length(path) AS distance, friend.name
ORDER BY distance

// Кратчайший путь
MATCH p = shortestPath(
    (ivan:User {name: 'Иван'})-[:FRIEND*]-(petr:User {name: 'Петр'})
)
RETURN p

// Поиск рекомендаций (то, что нравится друзьям друзей)
MATCH (u:User {name: 'Иван'})-[:FRIEND*2]->(friend)
MATCH (friend)-[:LIKES]->(book)
WHERE NOT (u)-[:LIKES]->(book)
RETURN DISTINCT book.title, count(friend) AS recommendations
ORDER BY recommendations DESC

// WITH для пайплайна
MATCH (u:User)-[:LIKES]->(b:Book)
WITH u, count(b) AS booksCount
WHERE booksCount > 5
RETURN u.name, booksCount

// UNION
MATCH (u:User)-[:LIKES]->(b:Book)
RETURN b.title AS name, 'book' AS type
UNION
MATCH (u:User)-[:LIKES]->(m:Movie)
RETURN m.title AS name, 'movie' AS type
```

### 📝 UPDATE — обновление
```cypher
// Обновление свойств
MATCH (u:User {name: 'Иван'})
SET u.age = 31, u.updated = timestamp()

// Удаление свойств
MATCH (u:User {name: 'Иван'})
REMOVE u.tempField

// Добавление метки
MATCH (u:User {name: 'Иван'})
SET u:Admin:Premium  // добавить метки

// Обновление связей
MATCH (u:User {name: 'Иван'})-[r:LIKES]->(b:Book {title: 'Мастер и Маргарита'})
SET r.rating = 5, r.updated = true
```

### 🗑️ DELETE — удаление
```cypher
// Удаление связи
MATCH (u:User {name: 'Иван'})-[r:LIKES]->(b:Book)
DELETE r

// Удаление узла (только без связей)
MATCH (u:User {name: 'Иван'})
DETACH DELETE u  // удалить узел и все его связи

// Удалить всё
MATCH (n)
DETACH DELETE n
```

## 📊 СЛОЖНЫЕ ЗАПРОСЫ

```cypher
// Поиск сообществ (алгоритм Лувена)
CALL algo.louvain.stream('User', 'FRIEND', {})
YIELD nodeId, community
MATCH (u:User) WHERE id(u) = nodeId
RETURN community, collect(u.name) AS members

// Центральность (PageRank)
CALL algo.pageRank.stream('User', 'FRIEND', {})
YIELD nodeId, score
MATCH (u:User) WHERE id(u) = nodeId
RETURN u.name, score
ORDER BY score DESC

// Поиск по расстоянию (гео)
MATCH (u:User)
WHERE distance(u.location, point({latitude: 55.75, longitude: 37.62})) < 10000
RETURN u.name

// Рекурсивный поиск
MATCH path = (ceo:User {title: 'CEO'})-[:MANAGES*]->(sub)
RETURN nodes(path), relationships(path)

// Статистика графа
MATCH (n)-[r]->()
RETURN labels(n) AS NodeType, type(r) AS RelationshipType, count(*) AS Count
```

---

# 📊 СРАВНЕНИЕ NoSQL БАЗ

| Характеристика | Redis | MongoDB | Cassandra | Neo4j |
|---------------|-------|---------|-----------|-------|
| **Тип** | Ключ-значение | Документная | Колоночная | Графовая |
| **Хранение** | In-memory | Диск | Диск | Диск |
| **Скорость чтения** | ⚡⚡⚡⚡⚡ | ⚡⚡⚡ | ⚡⚡⚡⚡ | ⚡⚡ |
| **Скорость записи** | ⚡⚡⚡⚡ | ⚡⚡⚡ | ⚡⚡⚡⚡⚡ | ⚡⚡ |
| **Масштабирование** | Репликация | Шардинг | Masterless | Репликация |
| **ACID** | Частично | Да (4.0+) | Нет | Да |
| **Язык запросов** | Команды | JSON-like | CQL | Cypher |
| **Когда использовать** | Кэш, сессии, счетчики, очереди | Каталоги, блоги, аналитика | IoT, логи, временные ряды | Соцсети, рекомендации, связи |
| **Когда НЕ использовать** | Сложные отношения, большие данные | Сложные транзакции | Сложные JOIN | Простые CRUD |

---

# 🎯 КРИТЕРИИ ВЫБОРА NoSQL

## ✅ Redis — когда нужно:
- Быстрый кэш
- Сессии пользователей
- Реальные рейтинги и лидерборды
- Очереди задач
- Pub/Sub
- Геопоиск
- Счетчики, лимиты

## ✅ MongoDB — когда нужно:
- Гибкая схема данных
- JSON-документы
- Быстрая разработка (стартапы)
- Каталоги товаров
- Блоги, CMS
- Журналы событий
- Location-based сервисы

## ✅ Cassandra — когда нужно:
- Огромные объемы данных (PB)
- Высокая доступность записи
- Распределенность по датацентрам
- Временные ряды (метрики, логи)
- IoT
- Нет критичности к согласованности

## ✅ Neo4j — когда нужно:
- Связанные данные
- Социальные графы
- Рекомендательные системы
- Анализ связей и влияния
- Поиск путей
- Fraud detection
- Управление зависимостями

---

# 🚀 ПАТТЕРНЫ ПРОЕКТИРОВАНИЯ

## 📦 Redis

```javascript
// Кэширование с TTL
const CACHE_TTL = 3600;
await redis.setex(`user:${id}`, CACHE_TTL, JSON.stringify(user));

// Счетчик просмотров
await redis.incr(`video:${id}:views`);
const views = await redis.get(`video:${id}:views`);

// Топ-10
await redis.zincrby('leaderboard:2024', 1, userId);
const top = await redis.zrevrange('leaderboard:2024', 0, 9, 'WITHSCORES');

// Очередь задач
await redis.lpush('tasks:email', JSON.stringify(task));
const task = await redis.brpop('tasks:email', 0);

// Блокировка
const lock = await redis.set(`lock:order:${orderId}`, 'locked', 'NX', 'EX', 10);
if (lock) { /* делаем работу */ await redis.del(`lock:order:${orderId}`); }

// Сессия
await redis.hmset(`session:${token}`, {
    userId, ip, userAgent, lastActivity: Date.now()
});
await redis.expire(`session:${token}`, 86400);
```

## 📦 MongoDB

```javascript
// Встраивание (embedding) — для вложенных данных
const order = {
    _id: ObjectId(),
    userId: userId,
    items: [
        { productId: 1, name: "Книга", price: 500, quantity: 2 },
        { productId: 2, name: "Ручка", price: 50, quantity: 5 }
    ],
    total: 1250
};

// Ссылки (referencing) — для отдельных коллекций
const user = { _id: ObjectId(), name: "Иван" };
const order = { 
    _id: ObjectId(), 
    userId: user._id,  // ссылка
    total: 1250 
};

// Полиморфные документы
db.products.insertMany([
    { type: "book", title: "Война и мир", author: "Толстой", pages: 1225 },
    { type: "electronics", name: "Ноутбук", brand: "Apple", warranty: 12 },
    { type: "service", name: "Подписка", period: "monthly" }
]);

// Временные ряды (Time Series)
db.measurements.insertOne({
    timestamp: new Date(),
    deviceId: 1,
    metrics: { temperature: 22.5, humidity: 45 },
    metadata: { location: "Moscow" }
});

// Геопространственные запросы
db.places.createIndex({ location: "2dsphere" });
db.places.find({
    location: {
        $near: {
            $geometry: { type: "Point", coordinates: [37.62, 55.75] },
            $maxDistance: 1000
        }
    }
});
```

## 📦 Cassandra

```javascript
// Моделирование под запросы (query-first)
// 1. Запрос: получить заказы пользователя по датам
CREATE TABLE orders_by_user (
    user_id uuid,
    order_date timestamp,
    order_id uuid,
    total decimal,
    PRIMARY KEY (user_id, order_date, order_id)
) WITH CLUSTERING ORDER BY (order_date DESC, order_id ASC);

// 2. Запрос: получить заказы по статусу
CREATE TABLE orders_by_status (
    status text,
    order_date timestamp,
    user_id uuid,
    order_id uuid,
    total decimal,
    PRIMARY KEY (status, order_date, order_id)
) WITH CLUSTERING ORDER BY (order_date DESC);

// Денормализация данных
CREATE TABLE user_recent_orders (
    user_id uuid,
    order_date timestamp,
    order_id uuid,
    order_summary text,  -- денормализованные данные
    PRIMARY KEY (user_id, order_date, order_id)
);

// Счетчики
CREATE TABLE user_stats (
    user_id uuid PRIMARY KEY,
    orders_count counter,
    total_spent counter
);
UPDATE user_stats SET orders_count = orders_count + 1 WHERE user_id = ?;
UPDATE user_stats SET total_spent = total_spent + 1250 WHERE user_id = ?;
```

## 📦 Neo4j

```javascript
// Модель социального графа
(User)-[:FRIEND]->(User)
(User)-[:LIKES]->(Movie|Book|Music)
(User)-[:REVIEWED {rating, text, date}]->(Movie)

// Рекомендательная система
MATCH (u:User {id: $userId})-[:LIKES]->(item)<-[:LIKES]-(similarUser)
MATCH (similarUser)-[:LIKES]->(recommendation)
WHERE NOT (u)-[:LIKES]->(recommendation)
RETURN recommendation, count(similarUser) AS score
ORDER BY score DESC
LIMIT 10

// Анализ влияния
MATCH path = (influencer:User)-[:INFLUENCES*1..3]->(follower)
WHERE influencer.id = $userId
RETURN follower.id, length(path) AS distance
ORDER BY distance

// Древо организаций
MATCH path = (ceo:Employee {title: 'CEO'})-[:MANAGES*]->(employee)
RETURN employee.name, length(path) AS level
ORDER BY level
```

---

# 📚 РЕСУРСЫ

## 🔴 Redis
- [Redis Commands](https://redis.io/commands)
- [Redis University](https://university.redis.com)
- [Redis in Action](https://redis.com/ebook/redis-in-action)

## 🟢 MongoDB
- [MongoDB Manual](https://docs.mongodb.com/manual)
- [MongoDB University](https://university.mongodb.com)
- [The Definitive Guide to MongoDB](https://www.oreilly.com/library/view/the-definitive-guide/9781484256822)

## 🔵 Cassandra
- [Cassandra Documentation](https://cassandra.apache.org/doc/latest)
- [DataStax Academy](https://www.datastax.com/dev)
- [Cassandra: The Definitive Guide](https://www.oreilly.com/library/view/cassandra-the-definitive/9781491933657)

## 🟣 Neo4j
- [Neo4j Documentation](https://neo4j.com/docs)
- [Neo4j GraphAcademy](https://neo4j.com/graphacademy)
- [Graph Databases](https://www.oreilly.com/library/view/graph-databases/9781491930885)

---
