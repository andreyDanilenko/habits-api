# 📚 ПОЛНАЯ ШПАРГАЛКА ПО SQL

## 🎯 ПОРЯДОК ВЫПОЛНЕНИЯ SQL-ЗАПРОСА

**❌ КАК МЫ ПИШЕМ (порядок команд):**
```sql
SELECT [DISTINCT | ALL] /* поля таблиц */
FROM /* из каких таблиц */
WHERE /* условие на ограничение строк */
GROUP BY /* условие группировки */
HAVING /* условие на ограничение строк после группировки */
ORDER BY /* порядок сортировки */ [ASC | DESC]
LIMIT /* ограничение кол-во записей */
```

**✅ КАК СЕРВЕР ДУМАЕТ (порядок выполнения):**
```
1. FROM / JOIN           ← выбор таблиц и соединение
2. WHERE                 ← фильтрация строк
3. GROUP BY              ← группировка
4. HAVING                ← фильтрация групп
5. SELECT                ← вычисление выражений
6. ORDER BY              ← сортировка
7. LIMIT / OFFSET        ← ограничение количества
```

---

## 🔤 ОСНОВНЫЕ КОМАНДЫ И КЛЮЧЕВЫЕ СЛОВА

| Команда | Описание | Пример |
|--------|---------|--------|
| **SELECT** | приказ СУБД выбрать что-то | `SELECT * FROM book;` |
| **AS** | "в качестве", псевдоним | `SELECT title AS Название FROM book;` |
| **DISTINCT** | уникальные значения | `SELECT DISTINCT author FROM book;` |
| **FROM** | из каких таблиц | `FROM book JOIN author` |
| **WHERE** | условие на ограничение строк | `WHERE price > 500` |
| **GROUP BY** | группировка данных | `GROUP BY author` |
| **HAVING** | фильтр после группировки | `HAVING SUM(price) > 1000` |
| **ORDER BY** | сортировка | `ORDER BY price DESC` |
| **LIMIT** | ограничение количества строк | `LIMIT 10` |
| **JOIN** | присоединить таблицу | `book JOIN author ON book.author_id = author.id` |
| **UNION** | слияние таблиц (вертикально) | `SELECT title FROM book1 UNION SELECT title FROM book2` |
| **UNION ALL** | слияние с дубликатами | `... UNION ALL ...` |
| **INTERSECT** | пересечение множеств | `... INTERSECT ...` |
| **EXCEPT** | разность множеств | `... EXCEPT ...` |
| **UPDATE** | изменение записей | `UPDATE book SET price = price * 1.1` |
| **DELETE** | удаление записей | `DELETE FROM book WHERE amount = 0` |
| **INSERT** | занести новые записи | `INSERT INTO book(title, author) VALUES ('Идиот', 'Достоевский Ф.М.')` |
| **CREATE** | создать таблицу/индекс/вью | `CREATE TABLE author (id INT PRIMARY KEY)` |
| **ALTER** | изменить таблицу | `ALTER TABLE book ADD COLUMN pages INT` |
| **DROP** | удалить таблицу | `DROP TABLE IF EXISTS temp` |
| **SET** | установить значение | `SET price = 0.7 * price` |
| **VALUES** | значения для INSERT | `VALUES (1, 'Булгаков')` |
| **PRIMARY KEY** | первичный ключ | `id INT PRIMARY KEY` |
| **FOREIGN KEY** | внешний ключ | `FOREIGN KEY (author_id) REFERENCES author(id)` |
| **AUTO_INCREMENT** | автоинкремент | `id INT PRIMARY KEY AUTO_INCREMENT` |

---

## 🎭 УСЛОВНЫЕ КОНСТРУКЦИИ

### IF() — функция
```sql
IF(логическое_выражение, выражение_1, выражение_2)
-- Все три параметра обязательны

SELECT 
    title, 
    amount, 
    price, 
    IF(amount < 4, price * 0.5, price * 0.7) AS sale
FROM book;

-- Вложенные IF
SELECT 
    author,
    title,
    IF(author = "Булгаков М.А.", price * 1.1,
        IF(author = "Есенин С.А.", price * 1.05, price * 1)
    ) AS new_price
FROM book;
```

### CASE — универсальная конструкция
```sql
-- Формат 1: поиск по значению
CASE значение
    WHEN сравниваемое_значение_1 THEN возвращаемое_значение_1
    WHEN сравниваемое_значение_2 THEN возвращаемое_значение_2
    WHEN сравниваемое_значение_n THEN возвращаемое_значение_n
    [ELSE возвращаемое_значение_по_умолчанию]
END

-- Формат 2: поиск по условиям
CASE
    WHEN условие_1 THEN результат_1
    WHEN условие_2 THEN результат_2
    ELSE результат_по_умолчанию
END

-- Пример 1: повышение цены в зависимости от автора
SELECT 
    author,
    title,
    CASE author
        WHEN "Булгаков М.А." THEN price * 1.1
        WHEN "Есенин С.А." THEN price * 1.05
        ELSE price
    END AS new_price
FROM book;

-- Пример 2: категоризация по цене
SELECT 
    title,
    price,
    CASE 
        WHEN price < 300 THEN 'Дешевая'
        WHEN price BETWEEN 300 AND 600 THEN 'Средняя'
        WHEN price > 600 THEN 'Дорогая'
        ELSE 'Не определена'
    END AS category
FROM book;

-- Пример 3: CASE в WHERE
SELECT *
FROM patients
WHERE TRUE
    AND 1 = (CASE 
                WHEN allergies = 'Penicillin' AND city = 'Burlington' 
                THEN 1 
                ELSE 0 
             END);
```

### COALESCE() — первый не-NULL
```sql
COALESCE(x, y, z, ...)  -- возвращает первый аргумент, который не NULL

SELECT COALESCE(NULL, NULL, 1, 2, NULL, 3)  -- 1
SELECT COALESCE(phone, email, 'нет контактов') FROM users;
```

### IFNULL() / ISNULL() — замена NULL
```sql
IFNULL(выражение, результат)      -- MySQL, SQLite
ISNULL(выражение, результат)      -- MS SQL Server

SELECT name_author, IFNULL(SUM(amount), 0) AS total_books
FROM author LEFT JOIN book ON author.author_id = book.author_id
GROUP BY name_author;
```

---

## 🔍 ОПЕРАТОРЫ ФИЛЬТРАЦИИ WHERE

| Оператор | Описание | Пример |
|---------|---------|--------|
| **=** | равно | `WHERE author = 'Булгаков М.А.'` |
| **!=, <>** | не равно | `WHERE price != 500` |
| **>** | больше | `WHERE price > 500` |
| **<** | меньше | `WHERE price < 500` |
| **>=** | больше или равно | `WHERE price >= 500` |
| **<=** | меньше или равно | `WHERE price <= 500` |
| **AND (&&)** | логическое И | `WHERE price > 500 AND amount < 10` |
| **OR (||)** | логическое ИЛИ | `WHERE author = 'Булгаков' OR author = 'Есенин'` |
| **NOT** | логическое НЕ | `WHERE NOT author = 'Булгаков'` |
| **BETWEEN** | в диапазоне (включая границы) | `WHERE price BETWEEN 500 AND 700` |
| **IN** | в списке | `WHERE amount IN (2, 3, 5, 7)` |
| **NOT IN** | не в списке | `WHERE amount NOT IN (0, 1)` |
| **LIKE** | по шаблону | `WHERE title LIKE '%война%'` |
| **NOT LIKE** | не соответствует шаблону | `WHERE title NOT LIKE '% %'` |
| **IS NULL** | равно NULL | `WHERE author IS NULL` |
| **IS NOT NULL** | не равно NULL | `WHERE author IS NOT NULL` |
| **EXISTS** | существует подзапрос | `WHERE EXISTS (SELECT 1 FROM book WHERE author_id = a.id)` |
| **ANY / SOME** | любой из | `WHERE price > ANY (SELECT price FROM book WHERE author = 'Булгаков')` |
| **ALL** | все из | `WHERE price > ALL (SELECT price FROM book WHERE author = 'Булгаков')` |

### LIKE — шаблоны поиска
```
'%a'      - оканчивается на 'a'
'a%'      - начинается с 'a'
'%abc%'   - содержит 'abc'
'a_'      - 'a' и ровно 1 любой символ
'a__'     - 'a' и ровно 2 любых символа
'a%b'     - начинается на 'a', заканчивается на 'b'
'_% _%'   - минимум 2 слова (пробел внутри, до и после минимум 1 символ)
'% % %'   - содержит 2 пробела (3 слова)
'[а-я]%'  - начинается с русской буквы
'[^0-9]%' - не начинается с цифры
```

### REGEXP — регулярные выражения
```sql
-- Функция REGEXP возвращает TRUE, если строка соответствует регулярному выражению
WHERE ProductName REGEXP 'Phone'              -- содержит "Phone"
WHERE ProductName REGEXP '^Phone'             -- начинается с "Phone"
WHERE ProductName REGEXP 'Phone$'             -- заканчивается на "Phone"
WHERE ProductName REGEXP 'iPhone [78]'        -- iPhone 7 или iPhone 8
WHERE ProductName REGEXP 'iPhone [6-8]'       -- iPhone 6,7,8
WHERE ProductName REGEXP 'Phone|Galaxy'       -- содержит Phone или Galaxy
WHERE name_genre REGEXP '[[:<:]]роман[[:>:]]' -- слово "роман" целиком
```

---

## 📊 АГРЕГАТНЫЕ ФУНКЦИИ

| Функция | Описание | Пример |
|--------|---------|--------|
| **COUNT(*)** | количество строк | `SELECT COUNT(*) FROM book;` |
| **COUNT(column)** | количество не-NULL значений | `SELECT COUNT(author) FROM book;` |
| **COUNT(DISTINCT column)** | количество уникальных | `SELECT COUNT(DISTINCT author) FROM book;` |
| **SUM(column)** | сумма | `SELECT SUM(price * amount) FROM book;` |
| **AVG(column)** | среднее | `SELECT AVG(price) FROM book;` |
| **MIN(column)** | минимум | `SELECT MIN(price) FROM book;` |
| **MAX(column)** | максимум | `SELECT MAX(price) FROM book;` |
| **GROUP_CONCAT()** | конкатенация строк (MySQL) | `SELECT author, GROUP_CONCAT(title) FROM book GROUP BY author;` |
| **STRING_AGG()** | конкатенация строк (PostgreSQL) | `SELECT author, STRING_AGG(title, ', ') FROM book GROUP BY author;` |
| **ARRAY_AGG()** | собрать в массив | `SELECT author, ARRAY_AGG(title) FROM book GROUP BY author;` |
| **FILTER** | фильтр для агрегации | `AVG(price) FILTER (WHERE category = 'роман')` |

### Примеры агрегации
```sql
-- Простая группировка
SELECT 
    author,
    COUNT(*) AS book_count,
    SUM(amount) AS total_copies,
    AVG(price) AS avg_price,
    MIN(price) AS min_price,
    MAX(price) AS max_price,
    SUM(price * amount) AS total_revenue
FROM book
GROUP BY author
HAVING COUNT(*) >= 2;

-- GROUP_CONCAT (MySQL)
SELECT 
    author,
    COUNT(*) AS book_count,
    GROUP_CONCAT(title SEPARATOR '; ') AS books
FROM book
GROUP BY author;

-- STRING_AGG (PostgreSQL)
SELECT 
    country,
    STRING_AGG(name, ', ' ORDER BY name) AS ships_list
FROM Ships s
JOIN Classes c ON s.class = c.class
GROUP BY country;

-- ARRAY_AGG (PostgreSQL)
SELECT 
    author,
    ARRAY_AGG(title) AS titles_array,
    ARRAY_LENGTH(ARRAY_AGG(title), 1) AS book_count
FROM book
GROUP BY author;

-- FILTER (PostgreSQL)
SELECT 
    AVG(price) AS avg_all,
    AVG(price) FILTER (WHERE amount > 5) AS avg_high_stock,
    AVG(price) FILTER (WHERE amount <= 5) AS avg_low_stock
FROM book;

-- SUM + CASE (кросс-таблица)
SELECT 
    author,
    SUM(CASE WHEN genre = 'роман' THEN amount ELSE 0 END) AS roman_sales,
    SUM(CASE WHEN genre = 'поэзия' THEN amount ELSE 0 END) AS poetry_sales,
    SUM(CASE WHEN genre = 'фантастика' THEN amount ELSE 0 END) AS sci_fi_sales,
    SUM(amount) AS total_sales
FROM book
GROUP BY author;
```

---

## 🔗 JOIN — ОБЪЕДИНЕНИЕ ТАБЛИЦ

```sql
-- INNER JOIN (только совпадающие записи)
SELECT title, name_author
FROM author
INNER JOIN book ON author.author_id = book.author_id;

-- LEFT JOIN (все из левой + совпадающие из правой)
SELECT name_author, title
FROM author
LEFT JOIN book ON author.author_id = book.author_id;

-- RIGHT JOIN (все из правой + совпадающие из левой)
SELECT name_author, title
FROM author
RIGHT JOIN book ON author.author_id = book.author_id;

-- FULL JOIN / FULL OUTER JOIN (все строки из обеих таблиц)
SELECT name_author, title
FROM author
FULL JOIN book ON author.author_id = book.author_id;

-- CROSS JOIN (декартово произведение)
SELECT name_author, title
FROM author
CROSS JOIN book;

-- SELF JOIN (присоединение таблицы к самой себе)
SELECT e1.name AS employee, e2.name AS manager
FROM employees e1
LEFT JOIN employees e2 ON e1.manager_id = e2.id;

-- NATURAL JOIN (по одноименным столбцам - НЕ РЕКОМЕНДУЕТСЯ)
SELECT * FROM author NATURAL JOIN book;

-- JOIN с USING (если столбцы называются одинаково)
SELECT title, name_author
FROM author
JOIN book USING(author_id);

-- Множественные JOIN
SELECT 
    b.title,
    a.name_author,
    g.name_genre
FROM book b
JOIN author a ON b.author_id = a.author_id
JOIN genre g ON b.genre_id = g.genre_id
WHERE g.name_genre = 'Роман'
ORDER BY b.title;
```

---

## 🧮 ОКОННЫЕ ФУНКЦИИ (WINDOW FUNCTIONS)

### Ранжирующие функции
```sql
ROW_NUMBER() OVER()                     -- просто нумерация строк 1,2,3,4...
RANK() OVER()                          -- с пропусками: 1,1,3,3,5...
DENSE_RANK() OVER()                   -- без пропусков: 1,1,2,2,3...
NTILE(n) OVER()                       -- разбиение на n групп
PERCENT_RANK() OVER()                 -- относительный ранг (0-1)
CUME_DIST() OVER()                   -- интегральное распределение

-- Пример
SELECT 
    name,
    subject,
    grade,
    ROW_NUMBER() OVER(PARTITION BY name ORDER BY grade DESC) AS row_num,
    RANK() OVER(PARTITION BY name ORDER BY grade DESC) AS rank,
    DENSE_RANK() OVER(PARTITION BY name ORDER BY grade DESC) AS dense_rank,
    NTILE(2) OVER(PARTITION BY name ORDER BY grade DESC) AS ntile_2,
    PERCENT_RANK() OVER(PARTITION BY name ORDER BY grade DESC) AS percent_rank,
    CUME_DIST() OVER(PARTITION BY name ORDER BY grade DESC) AS cume_dist
FROM student_grades;
```

### Агрегирующие оконные функции
```sql
SELECT 
    name,
    subject,
    grade,
    SUM(grade) OVER(PARTITION BY name) AS sum_by_name,
    AVG(grade) OVER(PARTITION BY name) AS avg_by_name,
    COUNT(grade) OVER(PARTITION BY name) AS count_by_name,
    MIN(grade) OVER(PARTITION BY name) AS min_by_name,
    MAX(grade) OVER(PARTITION BY name) AS max_by_name,
    SUM(grade) OVER(ORDER BY grade) AS running_total,
    AVG(grade) OVER(ORDER BY grade ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING) AS moving_avg
FROM student_grades;
```

### Функции смещения
```sql
LAG(column, offset, default) OVER(...)      -- значение из предыдущей строки
LEAD(column, offset, default) OVER(...)     -- значение из следующей строки
FIRST_VALUE(column) OVER(...)               -- первое значение в окне
LAST_VALUE(column) OVER(...)               -- последнее значение в окне
NTH_VALUE(column, n) OVER(...)            -- n-ое значение в окне

-- Примеры
SELECT 
    code,
    LAG(code) OVER(ORDER BY code) AS prev_code,
    LAG(code, 1, -999) OVER(ORDER BY code) AS prev_code_default,
    LAG(code, 2) OVER(ORDER BY code) AS prev_code_2,
    LEAD(code) OVER(ORDER BY code) AS next_code,
    FIRST_VALUE(code) OVER(ORDER BY code) AS first_code,
    LAST_VALUE(code) OVER(ORDER BY code ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) AS last_code,
    NTH_VALUE(code, 3) OVER(ORDER BY code) AS third_code
FROM products;
```

### ROWS vs RANGE — границы окна
```sql
-- ROWS — работа с физическими строками
ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW     -- все строки до текущей + текущая
ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING             -- 1 до, текущая, 1 после
ROWS BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING     -- текущая и все после
ROWS UNBOUNDED PRECEDING                             -- сокращение для ... AND CURRENT ROW

-- RANGE — работа с диапазоном значений (ORDER BY)
RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW    -- все значения <= текущему
RANGE BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING    -- все значения >= текущему

-- Пример
SELECT 
    date,
    revenue,
    SUM(revenue) OVER(ORDER BY date ROWS UNBOUNDED PRECEDING) AS running_total_rows,
    SUM(revenue) OVER(ORDER BY date RANGE UNBOUNDED PRECEDING) AS running_total_range,
    AVG(revenue) OVER(ORDER BY date ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING) AS moving_avg_3,
    SUM(revenue) OVER(ORDER BY date ROWS BETWEEN CURRENT ROW AND 2 FOLLOWING) AS forward_sum_3
FROM daily_sales;
```

### Процентили и статистика
```sql
-- PERCENTILE_CONT — непрерывный процентиль (интерполяция)
-- PERCENTILE_DISC — дискретный процентиль (существующее значение)

SELECT 
    department,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY salary) AS median_cont,
    PERCENTILE_DISC(0.5) WITHIN GROUP (ORDER BY salary) AS median_disc,
    PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY salary) AS q1,
    PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY salary) AS q3
FROM employees
GROUP BY department;
```

---

## 📅 РАБОТА С ДАТАМИ И ВРЕМЕНЕМ

### Функции даты и времени

| Функция | Описание | Пример |
|--------|---------|--------|
| **NOW(), CURRENT_TIMESTAMP** | текущая дата и время | `SELECT NOW();` |
| **CURRENT_DATE, CURDATE()** | текущая дата | `SELECT CURRENT_DATE;` |
| **CURRENT_TIME, CURTIME()** | текущее время | `SELECT CURRENT_TIME;` |
| **DATE()** | только дата | `SELECT DATE(created_at);` |
| **TIME()** | только время | `SELECT TIME(created_at);` |
| **YEAR()** | год | `YEAR('2024-01-15') → 2024` |
| **MONTH()** | месяц (1-12) | `MONTH('2024-01-15') → 1` |
| **DAY()** | день месяца | `DAY('2024-01-15') → 15` |
| **DAYOFMONTH()** | день месяца | аналогично DAY() |
| **DAYOFWEEK()** | день недели (1-7) | `DAYOFWEEK('2024-01-15')` |
| **DAYOFYEAR()** | день года (1-366) | `DAYOFYEAR('2024-01-15') → 15` |
| **WEEK()** | номер недели | `WEEK('2024-01-15')` |
| **MONTHNAME()** | название месяца | `MONTHNAME('2024-01-15') → 'January'` |
| **DAYNAME()** | название дня | `DAYNAME('2024-01-15') → 'Monday'` |
| **QUARTER()** | квартал (1-4) | `QUARTER('2024-01-15') → 1` |
| **HOUR()** | часы | `HOUR('15:30:45') → 15` |
| **MINUTE()** | минуты | `MINUTE('15:30:45') → 30` |
| **SECOND()** | секунды | `SECOND('15:30:45') → 45` |

### Разница между датами
```sql
-- MySQL, MS SQL Server
DATEDIFF(date1, date2)                     -- количество дней
DATEDIFF('2024-01-20', '2024-01-15') → 5

-- MySQL
TIMEDIFF(time1, time2)                    -- разница во времени
TIMESTAMPDIFF(unit, start, end)           -- разница в указанных единицах

-- PostgreSQL
AGE(date1, date2)                         -- интервал
EXTRACT(EPOCH FROM (date1 - date2)) / 86400 -- разница в днях

-- Пример: разница в минутах с учетом перехода через сутки
SUM((DATEDIFF(minute, time_out, time_in) + 1440) % 1440) AS minutes

-- Или через CASE
CASE
    WHEN DATEDIFF(mi, time_out, time_in) < 0 
        THEN 1440 + DATEDIFF(mi, time_out, time_in)
    ELSE DATEDIFF(mi, time_out, time_in)
END AS minutes
```

### Арифметика с датами
```sql
-- MySQL
DATE_ADD(date, INTERVAL value unit)        -- добавить интервал
DATE_SUB(date, INTERVAL value unit)        -- вычесть интервал

DATE_ADD('2024-01-15', INTERVAL 7 DAY)     -- 2024-01-22
DATE_ADD('2024-01-15', INTERVAL 1 MONTH)   -- 2024-02-15
DATE_ADD('2024-01-15', INTERVAL 1 YEAR)    -- 2025-01-15

-- PostgreSQL
date + integer                            -- добавить дни
date + INTERVAL '1 day'                  -- добавить интервал
date - INTERVAL '1 month'               -- вычесть интервал

-- MS SQL Server
DATEADD(unit, value, date)               -- добавить интервал
DATEADD(day, 7, '2024-01-15')           -- 2024-01-22
```

### Форматирование дат
```sql
-- MySQL
DATE_FORMAT(date, format)               -- форматирование даты
DATE_FORMAT('2024-01-15', '%d.%m.%Y')   -- 15.01.2024
DATE_FORMAT('2024-01-15', '%W, %M %D, %Y') -- Monday, January 15th, 2024

-- Форматы: %Y - 4-знач год, %y - 2-знач, %m - месяц (01-12), %d - день, 
-- %H - час (00-23), %i - минуты, %s - секунды, %W - день недели, %M - месяц

-- PostgreSQL
TO_CHAR(date, format)                   -- форматирование даты
TO_CHAR('2024-01-15', 'DD.MM.YYYY')     -- 15.01.2024
TO_CHAR('2024-01-15', 'FMMonth DD, YYYY') -- January 15, 2024

-- MS SQL Server
CONVERT(varchar, date, format)          -- конвертация с форматом
CONVERT(varchar, '2024-01-15', 3)       -- 15/01/2024
CONVERT(varchar, '2024-01-15', 104)     -- 15.01.2024

-- Форматы: 1/101 - mm/dd/yyyy, 3/103 - dd/mm/yyyy, 
-- 4/104 - dd.mm.yyyy, 10/110 - mm-dd-yyyy
```

### Извлечение частей даты
```sql
-- PostgreSQL: DATE_PART, EXTRACT
DATE_PART('year', TIMESTAMP '2024-01-15 20:31:05')      -- 2024.00
DATE_PART('month', TIMESTAMP '2024-01-15')              -- 1.00
DATE_PART('day', TIMESTAMP '2024-01-15')                -- 15.00
DATE_PART('hour', TIMESTAMP '2024-01-15 20:31:05')      -- 20.00
DATE_PART('minute', TIMESTAMP '2024-01-15 20:31:05')    -- 31.00
DATE_PART('second', TIMESTAMP '2024-01-15 20:31:05')    -- 5.00
DATE_PART('dow', TIMESTAMP '2024-01-15')               -- день недели (0-6)
DATE_PART('isodow', TIMESTAMP '2024-01-15')            -- день недели (1-7)
DATE_PART('quarter', TIMESTAMP '2024-01-15')           -- квартал (1-4)
DATE_PART('week', TIMESTAMP '2024-01-15')              -- неделя ISO

-- То же через EXTRACT
EXTRACT(YEAR FROM TIMESTAMP '2024-01-15')              -- 2024
EXTRACT(MONTH FROM TIMESTAMP '2024-01-15')             -- 1
EXTRACT(DAY FROM TIMESTAMP '2024-01-15')               -- 15
EXTRACT(EPOCH FROM TIMESTAMP '2024-01-15')             -- секунд с 1970-01-01
```

### Unix время
```sql
-- Unix timestamp — секунды с 1970-01-01
UNIX_TIMESTAMP()                       -- текущее Unix-время (MySQL)
UNIX_TIMESTAMP('2024-01-15')          -- конвертация даты в Unix-время

FROM_UNIXTIME(1705276800)             -- Unix → дата: 2024-01-15 00:00:00
FROM_UNIXTIME(1598291490)            -- 2020-08-24 17:51:30

-- Формула перевода: 1970-01-01 + time_unix / 86400
-- PostgreSQL: TO_TIMESTAMP(1705276800)

-- Конвертация секунд в формат времени
SEC_TO_TIME(288)                     -- 00:04:48 (MySQL)
MAKETIME(0, 4, 48)                  -- 00:04:48
```

### Первый/последний день месяца
```sql
-- MS SQL Server
EOMONTH(date)                        -- последний день месяца
EOMONTH('2024-01-15') → 2024-01-31
EOMONTH('2024-01-15', -1)           -- последний день предыдущего месяца
EOMONTH('2024-01-15', 1)            -- последний день следующего месяца

-- Первый день месяца
DATEADD(DAY, 1, EOMONTH('2024-01-15', -1))  -- 2024-01-01

-- MySQL
LAST_DAY(date)                       -- последний день месяца
DATE_FORMAT(date, '%Y-%m-01')        -- первый день месяца

-- PostgreSQL
DATE_TRUNC('month', date) + INTERVAL '1 month' - INTERVAL '1 day'  -- последний день
DATE_TRUNC('month', date)           -- первый день месяца
```

### DATE_TRUNC — усечение даты/времени (PostgreSQL)
```sql
-- Усекает дату/время до указанной точности
DATE_TRUNC('year', TIMESTAMP '2024-01-15 08:55:30')   -- 2024-01-01 00:00:00
DATE_TRUNC('month', TIMESTAMP '2024-01-15 08:55:30')  -- 2024-01-01 00:00:00
DATE_TRUNC('day', TIMESTAMP '2024-01-15 08:55:30')    -- 2024-01-15 00:00:00
DATE_TRUNC('hour', TIMESTAMP '2024-01-15 08:55:30')   -- 2024-01-15 08:00:00
DATE_TRUNC('minute', TIMESTAMP '2024-01-15 08:55:30') -- 2024-01-15 08:55:00
```

---

## 🔢 МАТЕМАТИЧЕСКИЕ ФУНКЦИИ

| Функция | Описание | Пример |
|--------|---------|--------|
| **ROUND(x, k)** | округление до k знаков | `ROUND(4.361, 2) → 4.36` |
| **CEILING(x), CEIL(x)** | округление вверх | `CEILING(4.2) → 5, CEIL(-5.8) → -5` |
| **FLOOR(x)** | округление вниз | `FLOOR(4.9) → 4, FLOOR(-5.8) → -6` |
| **TRUNCATE(x, k)** | усечение (без округления) | `TRUNCATE(4.361, 2) → 4.36` |
| **ABS(x)** | модуль числа | `ABS(-5) → 5` |
| **POWER(x, y)** | возведение в степень | `POWER(3, 4) → 81.0` |
| **SQRT(x)** | квадратный корень | `SQRT(16) → 4.0` |
| **EXP(x)** | экспонента (e^x) | `EXP(1) → 2.718...` |
| **LOG(x)** | натуральный логарифм | `LOG(2.718) → 1.0` |
| **LOG10(x)** | десятичный логарифм | `LOG10(100) → 2.0` |
| **PI()** | число π | `PI() → 3.1415926...` |
| **RADIANS(x)** | градусы → радианы | `RADIANS(180) → 3.14159...` |
| **DEGREES(x)** | радианы → градусы | `DEGREES(3.14159) → 180` |
| **SIN(x), COS(x), TAN(x)** | тригонометрические | `SIN(PI()/2) → 1` |
| **ASIN(x), ACOS(x), ATAN(x)** | обратные тригонометрические | `ASIN(1) → 1.5708` |
| **RAND(), RANDOM()** | случайное число (0-1) | `RAND() → 0.12345` |
| **SIGN(x)** | знак числа (-1,0,1) | `SIGN(-5) → -1` |
| **MOD(x, y)** | остаток от деления | `MOD(10, 3) → 1` |
| **POWER(SQRT(ABS()),3)** | комбинирование | `POWER(SQRT(ABS(-16)), 3) → 64` |

---

## 📝 РАБОТА СО СТРОКАМИ

### Основные строковые функции

| Функция | Описание | Пример |
|--------|---------|--------|
| **CONCAT(s1, s2, ...)** | объединение строк | `CONCAT('ab', 'cd') → 'abcd'` |
| **CONCAT_WS(sep, s1, s2)** | объединение с разделителем | `CONCAT_WS(', ', 'a', 'b') → 'a, b'` |
| **SUBSTRING(s, pos, len)** | подстрока | `SUBSTRING('abcdef', 2, 3) → 'bcd'` |
| **SUBSTR(s, pos, len)** | аналог SUBSTRING | `SUBSTR('abcdef', -3, 2) → 'de'` |
| **LEFT(s, n)** | первые n символов | `LEFT('abcdef', 3) → 'abc'` |
| **RIGHT(s, n)** | последние n символов | `RIGHT('abcdef', 3) → 'def'` |
| **LENGTH(s)** | длина строки (байты) | `LENGTH('abc') → 3` |
| **CHAR_LENGTH(s)** | длина строки (символы) | `CHAR_LENGTH('абв') → 3` |
| **UPPER(s)** | верхний регистр | `UPPER('abc') → 'ABC'` |
| **LOWER(s)** | нижний регистр | `LOWER('ABC') → 'abc'` |
| **TRIM(s)** | удалить пробелы с концов | `TRIM(' abc ') → 'abc'` |
| **LTRIM(s)** | удалить пробелы слева | `LTRIM(' abc') → 'abc'` |
| **RTRIM(s)** | удалить пробелы справа | `RTRIM('abc ') → 'abc'` |
| **REPLACE(s, from, to)** | замена подстроки | `REPLACE('abc', 'b', 'x') → 'axc'` |
| **INSTR(s, substr)** | позиция подстроки | `INSTR('abcdef', 'cde') → 3` |
| **POSITION(substr IN s)** | позиция подстроки | `POSITION('cde' IN 'abcdef') → 3` |
| **LOCATE(substr, s)** | позиция подстроки | `LOCATE('cde', 'abcdef') → 3` |
| **REVERSE(s)** | переворот строки | `REVERSE('abc') → 'cba'` |
| **REPEAT(s, n)** | повторение строки | `REPEAT('ab', 3) → 'ababab'` |
| **LPAD(s, len, pad)** | заполнение слева | `LPAD('5', 3, '0') → '005'` |
| **RPAD(s, len, pad)** | заполнение справа | `RPAD('5', 3, '0') → '500'` |
| **FORMAT(n, k)** | формат числа | `FORMAT(1234.5, 2) → '1,234.50'` |
| **ASCII(c)** | ASCII код символа | `ASCII('A') → 65` |
| **CHAR(n)** | символ по ASCII коду | `CHAR(65) → 'A'` |
| **STRCMP(s1, s2)** | сравнение строк | `STRCMP('a', 'b') → -1` |
| **FIELD(s, s1, s2, ...)** | позиция в списке | `FIELD('b', 'a', 'b', 'c') → 2` |

### Продвинутая работа со строками

```sql
-- Разделение строки (PostgreSQL)
SELECT UNNEST(STRING_TO_ARRAY('a,b,c', ','));  -- 'a', 'b', 'c' как строки
SELECT REGEXP_SPLIT_TO_TABLE('a1b2c3', '\d');   -- 'a', 'b', 'c'

-- Сборка в строку (PostgreSQL)
SELECT STRING_AGG(name, ', ' ORDER BY name) FROM authors;
SELECT ARRAY_TO_STRING(ARRAY['a', 'b', 'c'], '; ') → 'a; b; c'

-- Разделение строки (MySQL)
SELECT SUBSTRING_INDEX('a,b,c', ',', 2) → 'a,b'
SELECT SUBSTRING_INDEX(SUBSTRING_INDEX('a,b,c', ',', 2), ',', -1) → 'b'

-- Форматирование с printf (SQLite)
SELECT printf('%s %i %s', 'a', 123, NULL);  -- 'a 123 ' (NULL → пустая строка)
SELECT printf('%s %i %i', 'a', 123, NULL);  -- 'a 123 0' (NULL → 0)

-- Поиск и замена
SELECT REPLACE('The quick brown fox', ' ', '-');  -- 'The-quick-brown-fox'
SELECT TRANSLATE('12345', '135', '246');          -- '22444' (побуквенная замена)

-- Регулярные выражения
SELECT REGEXP_REPLACE('abc123def', '[0-9]', '');  -- 'abcdef'
SELECT REGEXP_SUBSTR('abc123def', '[0-9]+');      -- '123'
```

---

## 🗄️ DDL — ОПРЕДЕЛЕНИЕ ДАННЫХ

### CREATE TABLE — создание таблицы
```sql
CREATE TABLE [IF NOT EXISTS] author (
    author_id INT PRIMARY KEY AUTO_INCREMENT,
    name_author VARCHAR(50) NOT NULL,
    birth_year INT,
    country VARCHAR(50) DEFAULT 'Россия',
    biography TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_name (name_author)
) ENGINE = InnoDB, CHARSET = utf8mb4;

CREATE TABLE book (
    book_id INT PRIMARY KEY AUTO_INCREMENT,
    title VARCHAR(100) NOT NULL,
    author_id INT NOT NULL,
    genre_id INT,
    price DECIMAL(8,2) CHECK (price > 0),
    amount INT DEFAULT 0,
    pages INT,
    isbn VARCHAR(13) UNIQUE,
    FOREIGN KEY (author_id) REFERENCES author(author_id) 
        ON DELETE CASCADE 
        ON UPDATE CASCADE,
    FOREIGN KEY (genre_id) REFERENCES genre(genre_id)
        ON DELETE SET NULL,
    INDEX idx_title (title),
    INDEX idx_author (author_id),
    CONSTRAINT price_positive CHECK (price >= 0),
    CONSTRAINT amount_nonnegative CHECK (amount >= 0)
);

-- Создание таблицы на основе SELECT
CREATE TABLE book_copy AS
SELECT book_id, title, author_id, price
FROM book
WHERE amount > 0;

CREATE TABLE ordering AS
SELECT author, title, 5 AS amount
FROM book
WHERE amount < 4;
```

### ALTER TABLE — изменение таблицы
```sql
-- Добавление столбца
ALTER TABLE book 
ADD COLUMN pages INT NOT NULL DEFAULT 0,
ADD COLUMN description TEXT,
ADD COLUMN publisher VARCHAR(50) AFTER title;

-- Изменение столбца
ALTER TABLE book 
MODIFY COLUMN price DECIMAL(10,2) NOT NULL,
MODIFY COLUMN title VARCHAR(200),
CHANGE COLUMN pages page_count INT;  -- переименование

-- Переименование столбца
ALTER TABLE book 
RENAME COLUMN title TO book_title;

-- Удаление столбца
ALTER TABLE book 
DROP COLUMN old_column,
DROP COLUMN temporary;

-- Добавление ограничений
ALTER TABLE book 
ADD PRIMARY KEY (book_id),
ADD UNIQUE INDEX idx_isbn (isbn),
ADD FOREIGN KEY (author_id) REFERENCES author(author_id),
ADD CONSTRAINT price_positive CHECK (price > 0),
ADD INDEX idx_title_amount (title, amount);

-- Удаление ограничений
ALTER TABLE book 
DROP PRIMARY KEY,
DROP FOREIGN KEY fk_author,
DROP INDEX idx_isbn,
DROP CONSTRAINT price_positive;

-- Добавление внешнего ключа с именем
ALTER TABLE book
ADD CONSTRAINT fk_book_author 
FOREIGN KEY (author_id) REFERENCES author(author_id)
ON DELETE CASCADE;

-- Удаление внешнего ключа
ALTER TABLE book 
DROP FOREIGN KEY fk_book_author;

-- Переименование таблицы
ALTER TABLE book 
RENAME TO books;

ALTER TABLE books 
RENAME TO book;
```

### DROP TABLE — удаление таблицы
```sql
DROP TABLE IF EXISTS temp_table;
DROP TABLE author, book CASCADE;
```

### TRUNCATE TABLE — очистка таблицы
```sql
TRUNCATE TABLE temp_log;  -- быстрее DELETE, сбрасывает AUTO_INCREMENT
```

### CREATE INDEX — создание индексов
```sql
-- Обычный индекс
CREATE INDEX idx_author_name ON author(name_author);
CREATE INDEX idx_book_price ON book(price);
CREATE INDEX idx_book_author_genre ON book(author_id, genre_id);

-- Уникальный индекс
CREATE UNIQUE INDEX idx_book_isbn ON book(isbn);
CREATE UNIQUE INDEX idx_author_name_unique ON author(name_author);

-- Полнотекстовый индекс (MySQL)
CREATE FULLTEXT INDEX idx_book_title_desc ON book(title, description);

-- Частичный индекс (PostgreSQL, SQLite)
CREATE INDEX idx_partial ON book(price) WHERE amount > 0;

-- Индекс на выражение (PostgreSQL, SQLite)
CREATE INDEX idx_expression ON book((price * amount));
CREATE INDEX idx_lower_title ON book((LOWER(title)));

-- Удаление индекса
DROP INDEX idx_author_name ON author;
DROP INDEX idx_book_price ON book;
```

### VIEW — представления
```sql
-- Создание представления
CREATE VIEW view_available_books AS
SELECT 
    b.book_id,
    b.title,
    a.name_author,
    b.price,
    b.amount
FROM book b
JOIN author a ON b.author_id = a.author_id
WHERE b.amount > 0;

-- С представлением можно работать как с таблицей
SELECT * FROM view_available_books WHERE price < 500;

-- Обновляемое представление
CREATE VIEW view_book_prices AS
SELECT book_id, title, price, price * 1.1 AS new_price
FROM book;

-- Замена представления
CREATE OR REPLACE VIEW view_available_books AS
SELECT b.book_id, b.title, a.name_author, b.price, b.amount, g.name_genre
FROM book b
JOIN author a ON b.author_id = a.author_id
LEFT JOIN genre g ON b.genre_id = g.genre_id
WHERE b.amount > 0;

-- Удаление представления
DROP VIEW IF EXISTS view_available_books;

-- Информация о представлении
DESCRIBE view_available_books;
SHOW CREATE VIEW view_available_books;
```

### MATERIALIZED VIEW — материализованные представления (PostgreSQL)
```sql
-- Создание материализованного представления (хранит данные физически)
CREATE MATERIALIZED VIEW mv_book_stats AS
SELECT 
    author_id,
    COUNT(*) AS book_count,
    AVG(price) AS avg_price,
    SUM(amount) AS total_copies
FROM book
GROUP BY author_id;

-- Обновление данных
REFRESH MATERIALIZED VIEW mv_book_stats;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_book_stats;  -- без блокировки

-- Удаление
DROP MATERIALIZED VIEW mv_book_stats;
```

---

## 📋 DML — МАНИПУЛЯЦИЯ ДАННЫМИ

### INSERT — добавление записей
```sql
-- Вставка одной записи
INSERT INTO author (name_author, birth_year, country)
VALUES ('Булгаков М.А.', 1891, 'Россия');

-- Вставка нескольких записей
INSERT INTO author (name_author, birth_year, country) VALUES 
    ('Достоевский Ф.М.', 1821, 'Россия'),
    ('Есенин С.А.', 1895, 'Россия'),
    ('Пастернак Б.Л.', 1890, 'Россия');

-- Вставка с SELECT
INSERT INTO book_archive (book_id, title, author_id, price, amount, deleted_at)
SELECT book_id, title, author_id, price, amount, NOW()
FROM book
WHERE amount = 0;

-- INSERT IGNORE (MySQL) — игнорировать ошибки дубликатов
INSERT IGNORE INTO author (author_id, name_author) VALUES (1, 'Толстой Л.Н.');

-- INSERT ... ON DUPLICATE KEY UPDATE (MySQL)
INSERT INTO author (author_id, name_author) VALUES (1, 'Толстой Л.Н.')
ON DUPLICATE KEY UPDATE name_author = VALUES(name_author);

-- UPSERT (PostgreSQL)
INSERT INTO author (author_id, name_author) VALUES (1, 'Толстой Л.Н.')
ON CONFLICT (author_id) DO UPDATE 
SET name_author = EXCLUDED.name_author;

-- INSERT ... RETURNING (PostgreSQL, SQLite)
INSERT INTO author (name_author) VALUES ('Чехов А.П.')
RETURNING author_id, name_author;
```

### UPDATE — обновление записей
```sql
-- Обновление всех записей
UPDATE book SET price = price * 1.1;

-- Обновление с условием
UPDATE book 
SET price = price * 0.9 
WHERE amount < 5 AND author_id = 1;

-- Обновление с JOIN (MySQL)
UPDATE book b
JOIN author a ON b.author_id = a.author_id
SET b.price = b.price * 1.2
WHERE a.name_author = 'Булгаков М.А.';

-- Обновление с FROM (PostgreSQL, SQL Server)
UPDATE book
SET price = price * 1.1
FROM author
WHERE book.author_id = author.author_id
AND author.name_author = 'Булгаков М.А.';

-- UPDATE с подзапросом
UPDATE book
SET price = (
    SELECT AVG(price) * 1.2
    FROM book b2
    WHERE b2.author_id = book.author_id
)
WHERE amount > 0;

-- UPDATE ... RETURNING (PostgreSQL)
UPDATE book
SET amount = amount + 10
WHERE book_id = 1
RETURNING book_id, title, amount;
```

### DELETE — удаление записей
```sql
-- Удаление всех записей (медленно, с логированием)
DELETE FROM book;

-- Быстрая очистка таблицы (без логирования)
TRUNCATE TABLE book;

-- Удаление с условием
DELETE FROM book WHERE amount = 0 AND price < 100;

-- DELETE с JOIN (MySQL)
DELETE b
FROM book b
JOIN author a ON b.author_id = a.author_id
WHERE a.name_author = 'Булгаков М.А.'
AND b.price < 100;

-- DELETE с USING (PostgreSQL)
DELETE FROM book
USING author
WHERE book.author_id = author.author_id
AND author.name_author = 'Булгаков М.А.';

-- DELETE ... RETURNING (PostgreSQL)
DELETE FROM book
WHERE amount = 0
RETURNING book_id, title;
```

### REPLACE — замена записей (MySQL)
```sql
-- REPLACE = DELETE + INSERT (если запись существует)
REPLACE INTO author (author_id, name_author)
VALUES (1, 'Толстой Л.Н.');
```

---

## 🎯 ОПЕРАТОРЫ МНОЖЕСТВ

```sql
-- UNION — объединение без дубликатов
SELECT title FROM book_2023
UNION
SELECT title FROM book_2024;

-- UNION ALL — объединение с дубликатами (быстрее)
SELECT author FROM book_2023
UNION ALL
SELECT author FROM book_2024;

-- INTERSECT — пересечение (только в обеих)
SELECT author_id FROM book
INTERSECT
SELECT author_id FROM book_archive;

-- EXCEPT / MINUS — разность (в первой, но не во второй)
SELECT author_id FROM book
EXCEPT
SELECT author_id FROM book_archive;
```

---

## 💎 ПОДЗАПРОСЫ

### Скалярный подзапрос (возвращает одно значение)
```sql
SELECT 
    title,
    price,
    (SELECT AVG(price) FROM book) AS avg_price,
    price - (SELECT AVG(price) FROM book) AS diff_from_avg
FROM book;
```

### Подзапрос в WHERE
```sql
-- Сравнение с одним значением
SELECT title, price
FROM book
WHERE price > (SELECT AVG(price) FROM book);

-- IN
SELECT title, author_id
FROM book
WHERE author_id IN (
    SELECT author_id
    FROM author
    WHERE birth_year > 1850
);

-- EXISTS
SELECT name_author
FROM author a
WHERE EXISTS (
    SELECT 1
    FROM book b
    WHERE b.author_id = a.author_id
    AND b.price > 500
);

-- ANY / SOME
SELECT title, price
FROM book
WHERE price > ANY (
    SELECT price
    FROM book
    WHERE author_id = 1
);

-- ALL
SELECT title, price
FROM book
WHERE price > ALL (
    SELECT price
    FROM book
    WHERE author_id = 1
);
```

### Подзапрос в FROM
```sql
SELECT 
    author_name,
    book_count,
    total_revenue
FROM (
    SELECT 
        a.name_author AS author_name,
        COUNT(b.book_id) AS book_count,
        SUM(b.price * b.amount) AS total_revenue
    FROM author a
    LEFT JOIN book b ON a.author_id = b.author_id
    GROUP BY a.author_id
) AS author_stats
WHERE book_count > 1;
```

### Подзапрос в SELECT
```sql
SELECT 
    title,
    price,
    (SELECT name_author FROM author WHERE author_id = b.author_id) AS author_name,
    (SELECT COUNT(*) FROM book WHERE author_id = b.author_id) AS author_total_books
FROM book b;
```

### Кореллированные подзапросы
```sql
-- Найти книги дороже среднего по автору
SELECT 
    b1.title,
    b1.author_id,
    b1.price
FROM book b1
WHERE b1.price > (
    SELECT AVG(b2.price)
    FROM book b2
    WHERE b2.author_id = b1.author_id
);
```

---

## 🔄 CTE (Common Table Expressions)

### WITH — временные таблицы запроса
```sql
WITH author_stats AS (
    SELECT 
        author_id,
        COUNT(*) AS book_count,
        AVG(price) AS avg_price
    FROM book
    GROUP BY author_id
)
SELECT 
    a.name_author,
    s.book_count,
    s.avg_price
FROM author a
JOIN author_stats s ON a.author_id = s.author_id
WHERE s.book_count >= 2;

-- Несколько CTE
WITH 
book_stats AS (
    SELECT author_id, COUNT(*) AS cnt, AVG(price) AS avg_p
    FROM book GROUP BY author_id
),
author_births AS (
    SELECT author_id, name_author, birth_year
    FROM author WHERE birth_year IS NOT NULL
)
SELECT 
    ab.name_author,
    ab.birth_year,
    bs.cnt,
    bs.avg_p
FROM author_births ab
JOIN book_stats bs ON ab.author_id = bs.author_id
ORDER BY ab.birth_year;
```

### Рекурсивные CTE
```sql
-- Генерация чисел
WITH RECURSIVE numbers AS (
    SELECT 1 AS n
    UNION ALL
    SELECT n + 1 FROM numbers WHERE n < 10
)
SELECT * FROM numbers;

-- Генерация дат
WITH RECURSIVE dates AS (
    SELECT '2024-01-01' AS date
    UNION ALL
    SELECT DATE_ADD(date, INTERVAL 1 DAY)
    FROM dates
    WHERE date < '2024-01-31'
)
SELECT * FROM dates;

-- Иерархия сотрудников
WITH RECURSIVE org_tree AS (
    SELECT id, name, manager_id, 1 AS level
    FROM employees
    WHERE manager_id IS NULL
    
    UNION ALL
    
    SELECT 
        e.id, 
        e.name, 
        e.manager_id, 
        ot.level + 1
    FROM employees e
    JOIN org_tree ot ON e.manager_id = ot.id
)
SELECT * FROM org_tree;

-- Путь в иерархии
WITH RECURSIVE emp_path AS (
    SELECT id, name, manager_id, name AS path
    FROM employees
    WHERE manager_id IS NULL
    
    UNION ALL
    
    SELECT 
        e.id, 
        e.name, 
        e.manager_id, 
        CONCAT(ep.path, ' → ', e.name)
    FROM employees e
    JOIN emp_path ep ON e.manager_id = ep.id
)
SELECT * FROM emp_path;

-- MATERIALIZED / NOT MATERIALIZED (PostgreSQL)
WITH t AS MATERIALIZED (
    SELECT * FROM big_table WHERE condition
)
SELECT * FROM t JOIN another_table ON t.id = another_table.t_id;
```

---

## 📊 МЕТРИКИ И АНАЛИТИКА

### Основные метрики
```sql
-- ARPU (Average Revenue Per User)
SELECT 
    DATE(date) AS day,
    SUM(revenue) / COUNT(DISTINCT user_id) AS ARPU
FROM payments
GROUP BY DATE(date);

-- ARPPU (Average Revenue Per Paying User)
SELECT 
    DATE(date) AS day,
    SUM(revenue) / COUNT(DISTINCT user_id) AS ARPPU
FROM payments
WHERE revenue > 0
GROUP BY DATE(date);

-- AOV (Average Order Value)
SELECT 
    DATE(order_date) AS day,
    SUM(amount) / COUNT(*) AS AOV
FROM orders
GROUP BY DATE(order_date);

-- Retention Rate (удержание)
WITH user_activity AS (
    SELECT 
        user_id,
        DATE(created_at) AS activity_date,
        MIN(DATE(created_at)) OVER(PARTITION BY user_id) AS first_date
    FROM user_logs
    WHERE action = 'login'
)
SELECT 
    first_date AS cohort,
    DATEDIFF(activity_date, first_date) AS day_diff,
    COUNT(DISTINCT user_id) AS users
FROM user_activity
GROUP BY first_date, day_diff
ORDER BY first_date, day_diff;
```

---

## 🛠️ РАЗНЫЕ ПОЛЕЗНЫЕ ФУНКЦИИ

### Преобразование типов
```sql
-- CAST
CAST('123' AS INT)                 -- 123
CAST(123.45 AS DECIMAL(10,2))      -- 123.45
CAST('2024-01-15' AS DATE)        -- 2024-01-15

-- CONVERT (MySQL, SQL Server)
CONVERT('123', SIGNED)            -- 123
CONVERT('2024-01-15', DATE)       -- 2024-01-15
CONVERT(price, DECIMAL(10,2))     

-- Таблица приоритетов типов (высший → низший)
-- datetime, smalldatetime, float, real, decimal, money, 
-- smallmoney, int, smallint, tinyint, bit, nvarchar, nchar, varchar, char
```

### Работа с массивами (PostgreSQL)
```sql
-- Создание массива
SELECT ARRAY[1, 2, 3];
SELECT ARRAY_AGG(id) FROM users;

-- Длина массива
SELECT ARRAY_LENGTH(ARRAY[1, 2, 3], 1);  -- 3

-- Развернуть массив в строки
SELECT UNNEST(ARRAY['a', 'b', 'c']);

-- Проверка вхождения
SELECT 1 = ANY(ARRAY[1, 2, 3]);        -- true
SELECT 1 = ALL(ARRAY[1, 1, 1]);        -- true

-- Конкатенация массивов
SELECT ARRAY[1,2] || ARRAY[3,4];       -- {1,2,3,4}
```

### Работа с JSON (MySQL, PostgreSQL)
```sql
-- MySQL
SELECT JSON_OBJECT('id', 1, 'name', 'Булгаков');
SELECT JSON_ARRAY(1, 2, 3);
SELECT JSON_EXTRACT('{"a": 1, "b": 2}', '$.a');
SELECT JSON_UNQUOTE(JSON_EXTRACT(json_col, '$.name'));

-- PostgreSQL
SELECT JSONB_BUILD_OBJECT('id', 1, 'name', 'Булгаков');
SELECT TO_JSONB(users) FROM users;
SELECT jsonb_col->'name' FROM table;
SELECT jsonb_col->>'name' FROM table;  -- как текст
```

### Информационные запросы
```sql
-- Список таблиц
SHOW TABLES;                                      -- MySQL
SELECT table_name FROM information_schema.tables; -- Стандарт
\dt                                              -- PostgreSQL

-- Структура таблицы
DESCRIBE book;                                    -- MySQL
DESC book;
SHOW COLUMNS FROM book;
SELECT column_name, data_type FROM information_schema.columns 
WHERE table_name = 'book';

-- Индексы
SHOW INDEX FROM book;                            -- MySQL
SELECT * FROM pg_indexes WHERE tablename = 'book'; -- PostgreSQL

-- Создание базы данных
CREATE DATABASE habits_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Удаление базы данных
DROP DATABASE IF EXISTS habits_db;

-- Переключение базы данных
USE habits_db;                                    -- MySQL
\c habits_db;                                   -- PostgreSQL
```

---

## 🚨 ТИПИЧНЫЕ ОШИБКИ И ИХ РЕШЕНИЯ

### 1. Лишняя запятая перед FROM
```sql
-- ❌ НЕПРАВИЛЬНО
SELECT title, author, price, amount,
FROM book;

-- ✅ ПРАВИЛЬНО
SELECT title, author, price, amount
FROM book;
```

### 2. WHERE после HAVING
```sql
-- ❌ НЕПРАВИЛЬНО
GROUP BY author
HAVING SUM(price) > 1000
WHERE amount > 5;

-- ✅ ПРАВИЛЬНО
WHERE amount > 5
GROUP BY author
HAVING SUM(price) > 1000;
```

### 3. Два одинаковых алиаса
```sql
-- ❌ НЕПРАВИЛЬНО (второй перезапишет первый)
SELECT 
    IF(author = 'Булгаков', price*1.1, price) AS new_price,
    IF(author = 'Есенин', price*1.05, price) AS new_price
FROM book;

-- ✅ ПРАВИЛЬНО (один алиас с вложенным IF)
SELECT 
    IF(author = 'Булгаков', price*1.1,
        IF(author = 'Есенин', price*1.05, price)
    ) AS new_price
FROM book;

-- ✅ ПРАВИЛЬНО (CASE)
SELECT 
    CASE author
        WHEN 'Булгаков' THEN price * 1.1
        WHEN 'Есенин' THEN price * 1.05
        ELSE price
    END AS new_price
FROM book;
```

### 4. NULL в арифметических операциях
```sql
-- ❌ НЕПРАВИЛЬНО (NULL + число = NULL)
SELECT price * amount FROM book;  -- если amount = NULL, результат NULL

-- ✅ ПРАВИЛЬНО
SELECT price * IFNULL(amount, 0) FROM book;
SELECT price * COALESCE(amount, 0) FROM book;
```

### 5. Пустая строка в LIKE '% %'
```sql
-- ' ' LIKE '% %' → TRUE (строка из пробела проходит)
-- ''  LIKE '% %' → FALSE (пустая строка не проходит)

-- Для проверки "минимум 2 слова" лучше:
title LIKE '_% _%'  -- минимум 1 символ до и после пробела
```

### 6. Неправильный формат инициалов
```sql
-- Поиск авторов с форматом "Фамилия И.О."
author LIKE '% _._.'                    -- правильный формат
author LIKE '% С._.' OR author LIKE '% _.С.'  -- буква С в инициалах
```

---

## 🎓 ПРИМЕРЫ КОМПЛЕКСНЫХ ЗАПРОСОВ

### Пример 1: Отчет по продажам с ранжированием
```sql
WITH sales_stats AS (
    SELECT 
        a.name_author,
        COUNT(b.book_id) AS books_published,
        SUM(b.amount) AS total_copies_sold,
        SUM(b.price * b.amount) AS total_revenue,
        AVG(b.price) AS avg_price,
        AVG(b.amount) AS avg_copies_per_book
    FROM author a
    LEFT JOIN book b ON a.author_id = b.author_id
    GROUP BY a.author_id, a.name_author
)
SELECT 
    name_author,
    books_published,
    total_copies_sold,
    ROUND(total_revenue, 2) AS total_revenue,
    ROUND(avg_price, 2) AS avg_price,
    ROUND(avg_copies_per_book, 1) AS avg_copies,
    RANK() OVER(ORDER BY total_revenue DESC) AS revenue_rank,
    RANK() OVER(ORDER BY total_copies_sold DESC) AS popularity_rank,
    CASE 
        WHEN total_revenue > 10000 THEN 'Топ'
        WHEN total_revenue > 5000 THEN 'Средний'
        ELSE 'Низкий'
    END AS revenue_category
FROM sales_stats
WHERE books_published > 0
ORDER BY total_revenue DESC;
```

### Пример 2: Поиск книг по сложному критерию
```sql
-- Название из двух и более слов, 
-- автор с инициалами в формате "Фамилия И.О.",
-- буква 'С' в первом или втором инициале
SELECT 
    title,
    author,
    price,
    CASE 
        WHEN author LIKE '% С._.' THEN price * 1.1
        WHEN author LIKE '% _.С.' THEN price * 1.05
        ELSE price
    END AS new_price
FROM book
WHERE title LIKE '_% _%'                    -- минимум 2 слова
  AND author LIKE '% _._.'                 -- формат "Фамилия И.О."
  AND (author LIKE '% С._.' OR author LIKE '% _.С.')  -- буква С в инициалах
ORDER BY title;
```

### Пример 3: Анализ динамики продаж
```sql
WITH daily_sales AS (
    SELECT 
        DATE(order_date) AS sale_date,
        COUNT(DISTINCT order_id) AS orders_count,
        COUNT(DISTINCT customer_id) AS customers_count,
        SUM(order_amount) AS revenue,
        SUM(SUM(order_amount)) OVER(ORDER BY DATE(order_date)) AS cumulative_revenue,
        AVG(SUM(order_amount)) OVER(ORDER BY DATE(order_date) ROWS BETWEEN 6 PRECEDING AND CURRENT ROW) AS revenue_ma_7d
    FROM orders
    WHERE order_date >= DATE_SUB(CURDATE(), INTERVAL 90 DAY)
    GROUP BY DATE(order_date)
)
SELECT 
    sale_date,
    orders_count,
    customers_count,
    ROUND(revenue, 2) AS revenue,
    ROUND(cumulative_revenue, 2) AS cumulative_revenue,
    ROUND(revenue_ma_7d, 2) AS revenue_ma_7d,
    ROUND(revenue / NULLIF(customers_count, 0), 2) AS arpu,
    ROUND(revenue / NULLIF(orders_count, 0), 2) AS aov
FROM daily_sales
ORDER BY sale_date;
```

### Пример 4: Обновление с присвоением порядковых номеров
```sql
UPDATE applicant_order 
JOIN (
    SELECT 
        ROW_NUMBER() OVER (PARTITION BY program_id ORDER BY itog DESC) AS str_num,
        program_id, 
        enrollee_id, 
        itog
    FROM applicant_order
) AS t2 USING (program_id, enrollee_id)
SET applicant_order.str_id = t2.str_num;

SELECT * FROM applicant_order;
```

---

## 📚 СОКРАЩЕНИЯ И ТЕРМИНЫ

| Термин | Расшифровка | Описание |
|--------|------------|----------|
| **DDL** | Data Definition Language | Язык определения данных (CREATE, ALTER, DROP) |
| **DML** | Data Manipulation Language | Язык манипуляции данными (SELECT, INSERT, UPDATE, DELETE) |
| **DCL** | Data Control Language | Язык управления доступом (GRANT, REVOKE) |
| **TCL** | Transaction Control Language | Язык управления транзакциями (COMMIT, ROLLBACK) |
| **CTE** | Common Table Expression | Общее табличное выражение (WITH) |
| **PK** | Primary Key | Первичный ключ |
| **FK** | Foreign Key | Внешний ключ |
| **ACID** | Atomicity, Consistency, Isolation, Durability | Требования к транзакционной системе |
| **CRUD** | Create, Read, Update, Delete | Четыре базовые функции работы с данными |
| **OLTP** | Online Transaction Processing | Операционная обработка транзакций |
| **OLAP** | Online Analytical Processing | Аналитическая обработка данных |
| **ERD** | Entity-Relationship Diagram | Диаграмма "сущность-связь" |

---

*Эта шпаргалка содержит 99% того, что реально используется в работе с SQL. Сохраняй и пользуйся!* 🚀
Часть 1 не осмысленная) дополнял на ранних этапах

!!!!SQL выполняет команды не так!!! Это порядок записи кода:

SELECT [DISTINCT | ALL ] /* поля таблиц*/
FROM /* из каких таблиц*/
WHERE /* условие на ограничение строк*/
GROUP BY /*условие группировки*/
HAVING BY /*условие на ограничение строк после группировки*/
ORDER BY /*порядок сортировки*/ [ ASC | DESC]
LIMIT /* ограничение колво записей*/

!!!SQL думает в таком порядке :-- порядок выполнения запросов на выборку на СЕРВЕР:
 
                            FROM (выбор таблицы)
                            JOIN (комбинация с подходящими по условию данными из других таблиц)
                            WHERE (фильтрация строк)
                            GROUP BY (агрегирование данных)
                            HAVING (фильтрация агрегированных данных)
                            SELECT (возврат результирующего датасета)
                            ORDER BY (сортировка).
_________________________________________________________________________________________

SELECT * FROM book; - Выбрать все записи таблицы book

SELECT - приказ СУБД выбрать что-то

AS - "в качестве", "обзови всё что слева фразой справа"
    : левый операнд AS(в качестве) правого операнда
                /*SELECT title AS Название, amount 
                FROM book;*/

IF(логическое_выражение, выражение_1, выражение_2)
    Функция вычисляет логическое_выражение, если оно истина – 
    в поле заносится значение выражения_1, в противном случае –  значение выражения_2. 
    Все три параметра IF() являются обязательными.

    Допускается использование вложенных функций, вместо выражения_1 или выражения_2 может стоять новая функция IF.



JOIN - присоединить
                        SELECT title, name_author
                        FROM 
                            author INNER JOIN book
                            ON author.author_id = book.author_id;

HAVING COUNT(DISTINCT(....)) подсчет уникальных элементов в условии



WHERE используется ДО!!! GROUP BY, а после GROUP BY используется HAVING !!! - "Где (есть)", 
     (Если условие – истина, то строка(запись)  включается в выборку, если ложь – нет.)
    Логическое выражение после ключевого слова WHERE кроме операторов сравнения  и выражений 
    может включать  логические операции (И «and» (&&), ИЛИ «or» (||), НЕ «not») и 
    круглые скобки, изменяющие приоритеты выполнения операций.

    BETWEEN и IN -  Логическое выражение после ключевого слова WHERE 
                    WHERE может включать операторы  BETWEEN и IN. 
                Приоритет  у этих операторов такой же как у операторов сравнения, 
                то есть они выполняются раньше, чем NOT, AND, OR.
                /*SELECT title, amount 
                FROM book
                WHERE amount BETWEEN 5 AND 14;*/

                BETWEEN - позволяет отобрать данные, относящиеся к некоторому интервалу, включая его границы
                
                IN  определяет, совпадает ли значение столбца с одним из значений, содержащихся во вложенном запросе. 
                    При этом логическое выражение после WHERE получает значение истина. 
                    Оператор NOT IN выполняет обратное действие – выражение истинно, 
                    если значение столбца не содержится во вложенном запросе.

HAVING - В запросах с групповыми функциями вместо WHERE, которое размещается после оператора GROUP BY. 
                /*SELECT author,
                MIN(price) AS Минимальная_цена, 
                MAX(price) AS Максимальная_цена
                FROM book
                GROUP BY author
                HAVING SUM(price * amount) > 5000; */


WHERE фильтрует строки в таблице, HAVING фильтрует результаты группировки, 
      В рамках одного запроса может быть WHERE и HAVING, 
      просто WHERE работает ДО группировок, а HAVING - после

И where и having используются для наложения ограничений, 
но where используются при ограничении на исходные значения в строках нашей таблицы, 
а having на уже агрегированные значения.


GROUP BY (DISTINCT) - группирует данные при выборке, имеющие одинаковые значения в некотором столбце
            при этом можно использовать агрегатные функции SUM, AVG, MAX и т.д.
              /*SELECT  атрибут
                FROM таблица
                GROUP BY атрибут;*/   

DISTINCT (GROUP BY) - работает быстрее GROUP BY но с меньшим функционалом (отобрать уникальные элементы некоторого столбца)
               
                /*SELECT DISTINCT атрибут
                FROM book;*/

ORDER BY -  после которых задаются имена столбцов
    По умолчанию ORDER BY выполняет сортировку по возрастанию. 
    Чтобы управлять направлением сортировки вручную, 
    после имени столбца указывается ключевое слово ASC (по возрастанию) или DESC (по убыванию). 

    Столбцы после ключевого слова ORDER BY можно задавать:

                                            названием столбца;
                                            номером столбца указаннов в SELECT;
                                            именем столбца (указанным после AS).

UNION - слияние таблиц( формируем одну таблицу и вторую, 
        если столбцы совподают по колличеству сможем их соеденить в одну друг под другом)

LIKE - Оператор LIKE используется для сравнения строк в соответствии с шаблоном. 
    В языке SQL обзывается паттерном(шаблоном)
    Шаблон может включать обычные символы и символы-шаблоны ( % и _ ).
    
NOT LIKE, который в данном случае отберет все названия, в которых нет пробелов.

                            'abc%' - любые строки начинающееся с abc
                            'abc_' - строки начинанающееся с abc длинной 4 символа
                            '%z'   - последовательность символов на конце z
                            '%Rost%' - последовательность любая содержащая Rost
                            '% % %' текс содеожащий 2 пробела
                            WHERE name_genre LIKE"[[:<:]]роман[[:>:]]"

PRIMARY - "первичный", имеется в виду - "KEY", есть ещё один - FOREIGN, тоже - ключик. 
Эти ключи - истинная сила RELATION-ной модели базы данных, 
Relation - переводится как "отношение" чего-то к чему-то, либо как "связь" чего-то с чем-то(кого-то с кем-то - тоже).

UPDATE -- Изменение записей в таблице 
                UPDATE таблица SET поле = выражение

                UPDATE book 
                SET price = 0.7 * price;

                SELECT * FROM book;

SET - глагол - установи

INSERT - 'Занести': имя таблицы + ( cписок полей через запятую, в которые следует занести новые данные)
VALUES - что именно занести. список значений через запятую, 
которые заносятся в соответствующие поля, при этом текстовые значения заключаются в кавычки, 
числовые значения записываются без кавычек, в качестве разделителя целой и дробной части используется точка;
                /*INSERT INTO book(title, author, price, amount)
                VALUES ('Идиот', 'Достоевский Ф.М.', 460.00, 10);*/

DELETE - глагол "удали" 
                DELETE FROM таблица;  - Этот запрос удаляет все записи из указанной после FROM таблицы.
                /*DELETE FROM supply;*/

DROP - удалить таблицу
                        DROP TABLE applicant;

LIMIT - чтобы отобрать заданное количество отсортированных строк результата запроса
                       /*SELECT *
                        FROM trip
                        ORDER BY  date_first
                        LIMIT 1;*/

CASE - "В СЛУЧАЕ"

WHEN -  "КОГДА"

THEN -  "ТОГДА"
                        SELECT first_name, last_name,
                                CASE
                                WHEN TIMESTAMPDIFF(YEAR, birthday, NOW()) >= 18 THEN "Совершеннолетний"
                                ELSE "Несовершеннолетний"
                                END AS status
                                FROM Student

                            CASE значение
                                WHEN сравниваемое_значение_1 THEN возвращаемое_значение_1
                                WHEN сравниваемое_значение_2 THEN возвращаемое_значение_2
                                WHEN сравниваемое_значение_n THEN возвращаемое_значение_n
                                [ELSE возвращаемое_значение_по-умолчанию]
                            END

REGEXP          https://stepik.org/lesson/404275/step/3?auth=login&unit=393473
        -- Функция REGEXP в SQL используется для сопоставления текстовой строки 
        -- с регулярным выражением. Она возвращает TRUE, если строка соответствует 
        -- регулярному выражению, и FALSE в противном случае.

                WHERE ProductName REGEXP 'Phone': строка должна содержать "Phone", например, iPhone X, Nokia Phone N, iPhone

                WHERE ProductName REGEXP '^Phone': строка должна начинаться с "Phone", например, Phone 34, PhoneX

                WHERE ProductName REGEXP 'Phone$': строка должна заканчиваться на "Phone", например, iPhone, Nokia Phone

                WHERE ProductName REGEXP 'iPhone [78]';: строка должна содержать либо iPhone 7, либо iPhone 8

                WHERE ProductName REGEXP 'iPhone [6-8]';: строка должна содержать либо iPhone 6, либо iPhone 7, либо iPhone 8

                Например, найдем товары, названия которых содержат либо "Phone", либо "Galaxy":

                SELECT * FROM Products

                WHERE ProductName REGEXP 'Phone|Galaxy';


COALESCE()  -- это специальное выражение, которое вычисляет по порядку каждый из своих аргументов и 
            --на выходе возвращает значение первого аргумента, который был не NULL.
                        SELECT COALESCE(NULL, NULL, 1, 2, NULL, 3)
                            # 1


ISNULL(columnName, 0) -- замена NULL на 0


MONTH(дата) - чтобы выделить номер месяца из даты 
                        /*MONTH('2020-04-12') = 4
                        MONTH(date_first) месяц из столбца*/

DAY('2020-02-01') = 1
MONTH('2020-02-01') = 2
YEAR('2020-02-01') = 2020                        

MONTHNAME(дата) - возвращает название месяца на английском языке для указанной даты
                        /* MONTHNAME('2020-04-12')='April' */


DATEDIFF(дата_1, дата_2) - результатом которой является количество дней между дата_1 и дата_2. Например,

                        /* DATEDIFF('2020-04-01', '2020-03-28')=4

                        DATEDIFF('2020-05-09','2020-05-01')=8

                        DATEDIFF(date_last, date_first)*/

CEILING(x)    возвращает наименьшее целое число, большее или равное x
(округляет до целого числа в большую сторону)    
    CEILING(4.2)=5
    CEILING(-5.8)=-5

ROUND(x, k)    округляет значение x до k знаков после запятой,
если k не указано – x округляется до целого    
    ROUND(4.361)=4
    ROUND(5.86592,1)=5.9
CEIL(x)  которая выполняет округление числа вверх до ближайшего целого числа.

FLOOR(x)    возвращает наибольшее целое число, меньшее или равное x
(округляет до  целого числа в меньшую сторону)    
    FLOOR(4.2)=4
    FLOOR(-5.8)=-6

POWER(x, y)    возведение x в степень y    
    POWER(3,4)=81.0

SQRT(x)    квадратный корень из x    
    SQRT(4)=2.0
    SQRT(2)=1.41...

DEGREES(x)    конвертирует значение x из радиан в градусы    
    DEGREES(3) = 171.8...

RADIANS(x)    конвертирует значение x из градусов в радианы    
    RADIANS(180)=3.14...

ABS(x)    модуль числа x    
    ABS(-1) = 1
    ABS(1) = 1

PI()    pi = 3.1415926...

******************************************

Часть 2 осмысленная)
*********
00) Посмотреть таблицу 
                        SHOW COLUMNS FROM book;


                        
                        Query result:
                        +-----------+--------------+------+-----+---------+----------------+
                        | Field     | Type         | Null | Key | Default | Extra          |
                        +-----------+--------------+------+-----+---------+----------------+
                        | book_id   | int          | NO   | PRI | NULL    | auto_increment |
                        | title     | varchar(50)  | YES  |     | NULL    |                |
                        | author_id | int          | NO   | MUL | NULL    |                |
                        | genre_id  | int          | YES  | MUL | NULL    |                |
                        | price     | decimal(8,2) | YES  |     | NULL    |                |
                        | amount    | int          | YES  |     | NULL    |                |
                        +-----------+--------------+------+-----+---------+----------------+
0) Создание таблицы 
                        CREATE TABLE author (author_id INT PRIMARY KEY AUTO_INCREMENT, name_author VARCHAR(50));
                        SELECT * FROM author;

0.0) Наполнение таблицы
                        INSERT INTO author(name_author) 
                        VALUES ('Булгаков М.А.'), ('Достоевский Ф.М.'), ('Есенин С.А.'), ('Пастернак Б.Л.');

                        SELECT * FROM author;

Или так))) https://onecompiler.com/mysql 
-- CREATE TABLE trainingTeam(
-- trainingTeam_id INTEGER PRIMARY KEY AUTO_INCREMENT, 
-- name TEXT NOT NULL, 
-- age INTEGER, 
-- address TEXT NOT NULL
-- );
-- INSERT INTO trainingTeam(name, age, address)
-- VALUES ('Булгаков М.А.', 48, 'Москва, СССР'),
-- ('Достоевский Ф.М.', 59, 'Санкт-Петербург, Российская империя'),
-- ('Есенин С.А.', 30, 'Ленинград, СССР'),
-- ('Пастернак Б.Л.', 70, 'Переделкино, СССР'),
-- ('Искандер Ф.А.', 87, 'Переделкино, СССР');

-- SELECT * FROM trainingTeam;


1) Занести в таблицу 1  данные  из таблицы 2 - UPDATE 1,2

    Команда SET используется с UPDATE, чтобы указать, какие столбцы и значения должны быть обновлены в таблице.

                    UPDATE applicant_order JOIN (SELECT row_number() over (partition by program_id) AS str_num
                            , program_id, enrollee_id, itog
                        FROM applicant_order) AS t2 USING (program_id, enrollee_id)
                        SET applicant_order.str_id = t2.str_num;

                        SELECT * FROM applicant_order;

        Будьте осторожны при обновлении записей в таблице! 
    Обратите внимание на предложение WHERE в инструкции UPDATE. 
    Предложение WHERE указывает, какие записи должны быть обновлены. 
    Если вы опустите предложение WHERE, все записи в таблице будут обновлены!

1.1) ALTER TABLE -- изменение таблицы 

                                       ALTER TABLE client 
                                                    DROP FOREIGN KEY fk_client_source1, 
                                                    DROP COLUMN code,
                                                    DROP COLUMN source_id;
              
                     ALTER TABLE название_таблицы [WITH CHECK | WITH NOCHECK]
                                { ADD название_столбца тип_данных_столбца [атрибуты_столбца] | -- добавим в нашу таблицу новый столбец
                                      -- ADD author NVARCHAR(50) NOT NULL;
                                DROP COLUMN название_столбца | -- удалить столбец  DROP COLUMN authors
                                ALTER COLUMN название_столбца тип_данных_столбца [NULL|NOT NULL] | -- изменение столбца и его типа
                                       -- ALTER COLUMN book_category VARCHAR(200)
                                ADD [CONSTRAINT] определение_ограничения | -- добавление первичного ключа 
                                                                           -- ADD PRIMARY KEY (book_id)
                                     -- внешнего ключа 
                                     -- ADD FOREIGN KEY (author_id) REFERENCES authors(author_id)

                                     -- создание ограничения -- ADD CHECK (Age > 21) ошибка если есть иные данные 
                                     -- ALTER TABLE Customers WITH NOCHECK ADD CHECK (Age > 21) - не вызовет ошибку
                                     -- добавление имени ограничения 
                                     -- ALTER TABLE Customers ADD CONSTRAINT Check_Age_Greater_Than_Twenty_One CHECK (Age > 21);
                                     -- удаление ограничения
                                     -- ALTER TABLE Customers DROP Check_Age_Greater_Than_Twenty_One;

                                      ALTER TABLE client
                                                ADD source_id INT null,
                                                ADD CONSTRAINT fk_source_id FOREIGN KEY (source_id) REFERENCES source(id);


                                DROP [CONSTRAINT] имя_ограничения} -- ограничение  или ключ DROP FOREIGN KEY <fk_name>
                                 -- RENAME COLUMN author TO authors  -- переименование столбца
                                 -- ALTER TABLE book RENAME TO books_selectel;  -- переименование табл
                                 -- удалить таблицу DROP TABLE source;

1.2) Представления (виртуальная таблица)
                CREATE VIEW ViewUsers AS
                      SELECT id, name,
                             CONCAT(SUBSTR(email, 1, 2), '****', SUBSTR(email, -4)) AS email
                        FROM Users;

                DESCRIBE ViewUsers; -- посмотреть столбцы представления
                DROP VIEW view_name; -- удаление представления

                OR REPLACE -- при использовании этого опционального параметра в случае, если представление с таким именем уже существует, 
                           -- старое представление будет удалено, а новое создано. 
                           -- В противном случае, при попытке создать представление с существующем именем, возникнет ошибка.

                CREATE [OR REPLACE]
                VIEW имя_представления [(имена_полей_представления)]
                AS select_выражение

1.3) Материализованные представления -- Данный тип представлений один раз выполняет запрос и хранит данные.
                CREATE MATERIALIZED VIEW view_name AS
                SELECT columns
                  FROM tables;

                REFRESH MATERIALIZED VIEW view_name; -- обновления данных

2) Создать таблицу на основе данных из другой таблицы. 
            Для этого используется запрос SELECT, 
        результирующая таблица которого и будет новой таблицей базы данных. 
        При этом имена столбцов запроса становятся именами столбцов новой таблицы.
                                CREATE TABLE имя_таблицы AS
                                SELECT ...
        Пример

                                CREATE TABLE ordering AS
                                SELECT author, title, 5 AS amount
                                FROM book
                                WHERE amount < 4;
                                SELECT * FROM ordering;

3) Удалить из таблицы информацию
                                DELETE FROM fine
                                WHERE date_violation < '2020-02-01';

                                SELECT * FROM fine

4) REPLACE -- заменяет в строке_1 все вхождения строки_2 на строку_3
                                  
                                  REPLACE(строка1, строка2, строка3)
                                  UPDATE Ships
                                  SET name = REPLACE(name, ' ', '-');

                                  SELECT name, LEN(REPLACE(name, 'a', 'aa')) - LEN(name) 
                                  FROM Ships;


5) какие таблицы(все) созданы в БД:

                                SELECT table_name, engine
                                FROM information_schema.tables
                                WHERE engine = 'InnoDB';

6) Выборка
                                
                                SELECT  name_genre, title, name_author
                                FROM
                                    genre 
                                    INNER JOIN  book ON genre.genre_id = book.genre_id
                                    INNER JOIN  author ON author.author_id = book.author_id
                                    WHERE name_genre IN ('Роман') /* WHERE name_genre LIKE"% роман %"*/
                                ORDER BY title;

7) Колличество без NULL
                            SELECT name_author, COALESCE(SUM(amount), 0) AS Количество
                            FROM 
                                author LEFT JOIN book
                                on author.author_id = book.author_id   
                            GROUP BY name_author
                            HAVING Количество < 10 /*HAVING IFNULL(SUM(book.amount), 0) < 10*/ 
                            ORDER BY Количество; 
7.1) Замена NULL на 0
                            IFNULL(выражение, результат)
                                   которая возвращает результат, если выражение равно NULL, 
                                   и само выражение в противном случае.
7.2) COALESCE() --в SQL используется для замены нулевых значений на другое значение. 
                 -- Она принимает список аргументов и возвращает первое не-NULL значение из этого списка. 
                 -- Если все аргументы равны NULL, она возвращает NULL.
                 COALESCE(x, y)

7.3) ISNULL() -- Эта функция обычно используется в Microsoft SQL Server. Она проверяет, является ли значение NULL, 
              -- и возвращает 1 (True) если значение NULL, и 0 (False) в противном случае. Синтаксис:
              ISNULL(expression, replacement_value)


8) ОБьеденение запросов  UNION:

                            <запрос 1>
                            UNION [ALL]
                            <запрос 2>  

9) представить результат в один столбец, воспользуемся функцией  COALESCE:


                            SELECT COALESCE(m_pc.maker, m_printer.maker)
                             FROM
                            (SELECT DISTINCT maker FROM Product WHERE type='pc') m_pc
                            FULL JOIN
                            (SELECT DISTINCT maker FROM Product WHERE type='printer') m_printer
                            ON m_pc.maker = m_printer.maker   
                            WHERE m_pc.maker IS NULL OR m_printer.maker IS NULL; 

10) В случае пересечения и разности можно воспользоваться предикатом существования EXISTS.

    а также INTERSECT/EXCEPT если СУБД поддерживает

11) CONVERT

                            SELECT CONVERT(NUMERIC(6,2),AVG(numGuns*1.0)) AS ngAVG 
                            FROM Classes
                            WHERE type = 'bb'


                            
                    
12) Оконные функции

            -- Агрегирующие

                            select name, subject, grade,
                            sum(grade) over (partition by name) as sum_grade,
                            avg(grade) over (partition by name) as avg_grade,
                            count(grade) over (partition by name) as count_grade,
                            min(grade) over (partition by name) as min_grade,
                            max(grade) over (partition by name) as max_grade
                            from student_grades;

            -- Ранжирующие

                            select name, subject, grade,
                            row_number() over (partition by name order by grade desc),
                            rank() over (partition by name order by grade desc),
                            dense_rank() over (partition by name order by grade desc)
                            from student_grades;

                            ROW_NUMBER() over() -- просто нумерация строк;

                            RANK() over() -- ранжирование строк - при одинаковом значении строкам присваивается один номер,
                                          -- с пропуском номеров;

                            DENSE_RANK() over()-- ранжирование строк без пропуска номеров;

            -- Функции смещения

                            LAG() - выбирает строку, предшествующую текущей, если таковой нет - выдается NULL;

                            LEAD() - выбирает строку, следующую за текущей, если таковой нет - выдается NULL. 

                            LAG(code) OVER(ORDER BY code) prev_code

                            LAG(code,1,-999) OVER(ORDER BY code) prev_code --  -999 Значение этого параметра будет 
                            -- использоваться в том случае, если соответствующей строки не существует

                            LAG(code,2,-999) OVER(ORDER BY code) prev_code -- 2 какую из предыдущих (последующих) 
                            -- строк следует использовать, т.е. на сколько данная строка отстоит от текущей

                            /* Функцию можно использовать для того, чтобы сравнивать текущее значение строки с предыдущим или следующим. 
                             Имеет три параметра: столбец, значение которого необходимо вернуть, количество строк для смещения (по умолчанию 1), 
                             значение, которое необходимо вернуть если после смещения возвращается значение NULL;*/

                            FIRST_VALUE() -- с помощью функции можно получить первое значение в окне.
                            LAST_VALUE() -- с помощью функции можно получиТь последнее значение в окне. 
                            /*В качестве параметра принимает столбец, значение которого необходимо вернуть.*/                       

            -- Аналитические функции
                            -- это функции которые возвращают информацию о распределении данных и используются для статистического анализа.

                            CUME_DIST -- вычисляет интегральное распределение (относительное положение) значений в окне;
                            PERCENT_RANK -- вычисляет относительный ранг строки в окне;
                            PERCENTILE_CONT -- вычисляет процентиль на основе постоянного распределения значения столбца. 
                                            -- В качестве параметра принимает процентиль, который необходимо вычислить (например посчитать медиану);
                            PERCENTILE_DISC -- вычисляет определенный процентиль для отсортированных значений в наборе данных. 
                                            -- В качестве параметра принимает процентиль, который необходимо вычислить.
                            Важно! У функций PERCENTILE_CONT и PERCENTILE_DISC, столбец, 
                            по которому будет происходить сортировка, указывается с помощью ключевого слова WITHIN GROUP.

                            SELECT 
                                Date
                                , Medium
                                , Conversions
                                , CUME_DIST() OVER(PARTITION BY Date ORDER BY Conversions) AS 'Cume_Dist' 
                                , PERCENT_RANK() OVER(PARTITION BY Date ORDER BY Conversions) AS 'Percent_Rank' 
                                , PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY Conversions) OVER(PARTITION BY Date) AS 'Percentile_Cont' 
                                , PERCENTILE_DISC(0.5) WITHIN GROUP (ORDER BY Conversions) OVER(PARTITION BY Date) AS 'Percentile_Disc'
                                FROM Orders

12.1) ROWS или RANGE

        Инструкция ROWS -- позволяет ограничить строки в окне, указывая фиксированное количество строк, предшествующих или следующих за текущей.

                    sum(revenue) OVER(ORDER BY date, revenue rows between unbounded preceding and current row) as total_revenue
                                    
                    sum(arpu) OVER(ORDER BY date, arpu ROWS UNBOUNDED PRECEDING) as running_arpu -- текущая + предидущая

                    -- не ограничиваем диапозон строк
                    ROWS BETWEEN CURRENT ROW AND 1 FOLLOWING -- в окно попадут текущая и одна следующая запись;
                    BETWEEN UNBOUNDED PRECEDING  -- Все, что до текущей строки/диапазона
                    BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW -- Все, что до текущей строки/диапазона и само значение текущей строки
                    BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING -- Текущая строка/диапазон и все, что после нее

                    -- конкретное ограничение диапозона строк
                    BETWEEN <целое число> Preceding AND <целое число> Following -- сколько строк до и после включать (не поддерживается для RANGE)
                    BETWEEN CURRENT ROW AND <целое число> Following -- сколько строк до и после включать (не поддерживается для RANGE)
                    BETWEEN <целое число> Preceding AND CURRENT ROW -- сколько строк до и после включать (не поддерживается для RANGE)
         
        UNBOUNDED PRECEDING — указывает, что окно начинается с первой строки группы;
        UNBOUNDED FOLLOWING – с помощью данной инструкции можно указать, что окно заканчивается на последней строке группы;
        CURRENT ROW – инструкция указывает, что окно начинается или заканчивается на текущей строке;
        BETWEEN «граница окна» AND «граница окна» — указывает нижнюю и верхнюю границу окна;
        <целое число> PRECEDING – определяет число строк перед текущей строкой (не допускается в предложении RANGE).;
        <целое число> FOLLOWING — определяет число строк после текущей строки (не допускается в предложении RANGE).
 
    
        Инструкция RANGE --, в отличие от ROWS, работает не со строками, а с диапазоном строк в инструкции ORDER BY. 
            То есть под одной строкой для RANGE могут пониматься несколько физических строк одинаковых по рангу.
            Предложение RANGE может использоваться только с опциями CURRENT ROW, UNBOUNDED PRECEDING и UNBOUNDED FOLLOWING.
            
            Предложение RANGE предназначено также для ограничения набора строк. 
            В отличие от ROWS, оно работает не с физическими строками, а с диапазоном строк в предложении ORDER BY. 
            Это означает, что одинаковые по рангу строки в контексте предложения ORDER BY будут считаться как одна 
            текущая строка для функции CURRENT ROW. А в предложении ROWS текущая строка – это одна, текущая строка набора данных.


13) Выбор случайных строк в один запрос


               быстрый 
                                MAX * rand()

                                Получение одной строки:
                                SELECT t.pk_id FROM test as t,
                                (SELECT ROUND((SELECT MAX(pk_id) FROM test) *rand()) as rnd 
                                FROM test LIMIT 1) tmp
                                WHERE t.pk_id = rnd

                                Среднее время выполнения — 0.001 секунды
                                Получение 100 строк:
                                SELECT t.pk_id FROM test as t,
                                (SELECT ROUND((SELECT MAX(pk_id) FROM test) *rand()) as rnd 
                                FROM test LIMIT 100) tmp
                                WHERE t.pk_id in (rnd)
                                ORDER BY pk_id
               надежный
                                ORDER BY rand + LIMIT

                                Изображение не загружено

                                Получение одной строки:
                                SELECT pk_id FROM test ORDER BY rand() LIMIT 1

                                Среднее время выполнения в MySQL — 6.150 секунд

                                Попробуем взять 100 записей
                                SELECT pk_id FROM test ORDER BY rand() LIMIT 100

14) LEFT -- Чтобы выделить крайние левые n символов из строки используется функция LEFT(строка, n):

                                LEFT("abcde", 3) -> "abc"


15) CONCAT -- Соединение строк осуществляется с помощью функции CONCAT(строка_1, строка_2):

                                CONCAT("ab","cd") -> "abcd"

    CONCAT --Если хотя бы один из аргументов равен NULL, возвращается значение NULL.

                                    SELECT CONCAT('sql', '-', 'academy');

                                    SELECT CONCAT('sql', NULL);

    

16) FILTER --Вариант фильтрации:
                      Если после агрегирующей функции указать ключевое слово FILTER и поместить в скобках 
                      некоторое условие condition после WHERE, то агрегирующей функции на вход 
                      будут поданы только те строки, для которых условие фильтра окажется истинным.

                                        В общем виде эта конструкция выглядит так:

                                        SELECT agg_function(column) FILTER (WHERE condition)
                                        FROM table


        Например, если бы мы захотели посчитать среднюю цену только для товаров категории 'рыба', то запрос выглядел бы так:

                                                SELECT AVG(price) FILTER (WHERE category = 'рыба') AS avg_fish_price
                                                FROM table

17) UPPER -- Преобразовать все буквы указанной строки в верхний регистр.
                                        
                                        UPPER( string )

18) Unix -- время , в котором хранится количество секунд, прошедших с 1 января 1970 года. 
                       Для перевода к привычному виду DATE используется формула:

                        1970-01-01 + time_unix / 86400
          В SQL для перевода удобно использовать функцию  FROM_UNIXTIME( ).

                Например:

                FROM_UNIXTIME(1598291490) =  2020-08-24 17:51:30
                
          Для перевода количества секунд во временной формат используется функция SEC_TO_TIME(),например:

                SEC_TO_TIME(288) = 0:04:48


19) STRING_AGG -- позволяющая конкатенировать строки

                        select  maker, STRING_AGG(type, '/')
                        from (select distinct maker, type from Product) as Product
                        group by maker

                 -- Чтобы задать порядок сортировки, используется необязательное предложение WITHIN GROUP

                         SELECT country, STRING_AGG(name,',') WITHIN GROUP (ORDER BY name) ships_list 
                            FROM Ships s JOIN Classes c ON s.class=c.class
                            GROUP BY country
                            ORDER BY country;
                            
20) AGE --Вернуть разницу между двумя значениями, представленными в формате TIMESTAMP
                                    SELECT AGE('2022-12-12', '2021-11-10')

                                    Результат:
                                    397 days, 0:00:00

                                        SELECT AGE(current_date, '2021-11-10')

                                        Результат:
                                        397 days, 0:00:00

                                        SELECT AGE(current_date, '2021-11-10')::VARCHAR

                                        Результат:
                                        1 year 1 mon 2 days

20.0) MONTHNAME(дата) -- выделить название месяца из даты
                        MONTHNAME('2020-04-12')='April'
                        
20.1) DATEDIFF(дата_1, дата_2) --Вернуть разницу между двумя значениями -- DATEDIFF(datepart, startdate, enddate)
                 DATEDIFF('2020-04-01', '2020-03-28')=4

                    DATEDIFF('2020-05-09','2020-05-01')=8

                    DATEDIFF(date_last, date_first)

                    SUM((DATEDIFF(minute, time_out, time_in) + 1440)%1440) AS minutes

                    или

                    case
                            when DATEDIFF(mi, time_out, time_in) < 0 then 1440 + DATEDIFF(mi, time_out, time_in)
                            else DATEDIFF(mi, time_out, time_in)
                    end as int

20.2) В PGSql нет DATEDIFF

             SELECT (EXTRACT(EPOCH FROM time_out) - EXTRACT(EPOCH FROM time_in)) * 60 AS difference_in_minutes
               FROM your_table;
               
21) DATE_TRUNC -- используется для усечения дат и времени, т.е. она работает аналогично округлению ROUND, 
только для типов данных TIMESTAMP и INTERVAL.

Синтаксис у неё такой же, как и у DATE_PART:
                                    
                                         SELECT DATE_TRUNC(part, column)

На месте part в кавычках указывается, 
до какой точности следует округлять переданное значение времени:  'year', 'month', 'day', 'hour'

                                        SELECT DATE_TRUNC('month', TIMESTAMP '2022-01-12 08:55:30')

                                        Результат:
                                        01/01/22 00:00

                                        SELECT DATE_TRUNC('day', TIMESTAMP '2022-01-12 08:55:30')

                                        Результат:
                                        12/01/22 00:00    

                                        SELECT DATE_TRUNC('hour', TIMESTAMP '2022-01-12 08:55:30')

                                        Результат:
                                        12/01/22 08:00    

22.0) Оставить только дату без времени 1962-10-20 00:00:00.000  в 1962-10-20
       mssql   SELECT CONVERT(date, аргумент) AS DateOnly;
               SELECT CAST(аргумент AS date) AS DateOnly;

        остальные  SELECT DATE(аргумент) AS DateOnly;

22) DATE_PART --Для извлечения части даты (год, месяц, день, час и т.д.) 


                                           SELECT DATE_PART(part, column)


            На месте part необходимо в кавычках указать ту часть, 
            которую нужно извлечь: 'year', 'month', 'day', 'hour' и т.д. 
            На месте column следует указать нужную колонку либо конкретную дату или время. 


                                            SELECT DATE_PART('year', DATE '2022-01-12')

                                            Результат:
                                            2022.00


                                            SELECT DATE_PART('month', DATE '2022-01-12')

                                            Результат:
                                            1.00


                                            SELECT DATE_PART('day', DATE '2022-01-12')

                                            Результат:
                                            12.00


                                            SELECT DATE_PART('hour', TIMESTAMP '2022-01-12 20:31:05')

                                            Результат:
                                            20.00


                                            SELECT DATE_PART('minute', TIMESTAMP '2022-01-12 20:31:05')

                                            Результат:
                                            31.00


        Выше в качестве примера мы указали конкретную дату. На её месте могла быть, например, колонка с датами dates.
        Тогда запрос выглядел бы так:
                                                    SELECT DATE_PART('day', dates)
                        
        century    Использует григорианский календарь, где первый век начинается с '0001-01-01 00:00:00 AD'
        day    день месяца (1 to 31)
        decade    Год делится на 10
        dow    день в неделю (0=Sunday, 1=Monday, 2=Tuesday, ... 6=Saturday)
        doy    день недели в году (1 = первый день года, 365/366 = последний день года, в зависимости от того, високосный ли это год)
        epoch    Количество секунд с '1970-01-01 00:00:00 UTC', если значение даты. Количество секунд в интервале, если значение интервал.
        hour    час (0 to 23)
        isodow    день недели (1=Monday, 2=Tuesday, 3=Wednesday, ... 7=Sunday)
        isoyear    ISO 8601 (где год начинается в понедельник недели, содержащей 4 января)
        microseconds    Секунды (и доли секунды), умноженные на 1 000 000
        millennium    значение тысячелетия
        milliseconds    Секунды (и доли секунды), умноженные на 1000
        minute    минута (0 to 59)
        month    номер месяца для месяца (от 1 до 12), если значение даты. Количество месяцев (от 0 до 11), если значение интервала
        quarter    квартал (с 1 по 4)
        second    секунды (и доли секунды)
        timezone    Смещение часового пояса от UTC, выраженное в секундах
        timezone_hour    Часовая часть смещения часового пояса от UTC
        timezone_minute    Минутная часть смещения часового пояса от UTC
        week    Номер недели в году, основанный на ISO 8601 (где год начинается в понедельник недели, содержащей 4 января)
        the year    год как 4 цифры

22.0) Последний день месяца 
                           EOMONTH(date) AS lastD
22.00) Первый день месяца 
                           DATEADD(DAY,1,EOMONTH(date,-1)) as firstD

22.1) EXTRACT -- извлечение из даты части (год, месяц, день...)

                                          EXTRACT(field FROM source)

   https://learndb.ru/articles/article/122  
   

Функция extract получает из значений даты/времени поля, такие как год или час. 
Здесь source — значение типа timestamp, time или interval. 
(Выражения типа date приводятся к типу timestamp, так что допускается и этот тип.) 
Указанное поле представляет собой идентификатор, по которому из источника выбирается заданное поле.


23) DATEADD (datepart, number, date) -- возвращает значение типа datetime, которое получается добавлением 
                                     -- к дате date количества интервалов типа datepart, равного number (целое число)

     Datepart    Допустимые сокращения
        Year     — год    yy, yyyy
        Quarter  — квартал    qq, q
        Month    — месяц    mm, m
        Dayofyear — день года    dy, y
        Day      — день    dd, d
        Week     — неделя    wk, ww
        Hour     — час    hh
        Minute   — минута    mi, n
        Second   — секунда    ss, s
        Millisecond - миллисекунда    ms                               


24) TO_CHAR() -- Преобразование даты в строку 
                                            SELECT TO_CHAR(SYSDATE, 'FMMonth DD, YYYY') FROM DUAL;
                                            
                                            --Результат:   Август 9, 2014
                                            
                                            SELECT TO_CHAR(SYSDATE, 'FMMON DDth, YYYY') FROM DUAL;
                                            
                                            --Результат:   АВГ 9TH, 2014
                                            
                                            SELECT TO_CHAR(SYSDATE, 'FMMon ddth, YYYY') FROM DUAL;
                                            
                                            --Результат:   Авг 9th, 2014

                                            SELECT TO_CHAR(sysdate, 'yyyy/mm/dd') FROM DUAL;
                                            
                                            --Результат:   2014/08/28
                                            
                                            SELECT TO_CHAR(sysdate, 'yyyy.mm.dd') FROM DUAL;
                                            
                                            --Результат:   2014.08.28
                                            
                                            SELECT TO_CHAR(sysdate, 'Month DD, YYYY') FROM DUAL;
                                            
                                            --Результат:   Август   28, 2014

                Параметр    Пояснение
                YEAR    Год.
                YYYY    4-значный год.
                YYY
                YY
                Y    Последние 3, 2 или 1 цифра(ы) года.
                IYY
                IY
                I    Последние 3, 2 или 1 цифра(ы) года ISO.
                IYYY    4-значный год в соответствии со стандартом ISO.
                Q    Квартал года (1, 2, 3, 4; JAN-MAR = 1).
                MM    Месяц (01-12; JAN = 01).
                MON    Сокращенное название месяца.
                MONTH    Название месяца, дополненное пробелами длиной до 9 символов.
                RM    Римская цифра RM (I-XII; JAN = I).
                WW    Неделя года (1-53), где неделя 1 начинается в первый день года и продолжается до седьмого дня года.
                W    Неделя месяца (1-5), где неделя 1 начинается в первый день месяца и заканчивается седьмым.
                IW    Неделя года (1-52 или 1-53) на основе стандарта ISO.
                D    День недели (1-7).
                DAY    Название дня.
                DD    День месяца (1-31).
                DDD    День года (1-366).
                DY    Сокращенное название дня.
                J    юлианский день; количество дней с 1 января 4712 г. до н.э.
                HH    Час дня (1-12).
                HH12    Час дня (1-12).
                HH24    Час дня (0-23).
                MI    Минуты (0-59).
                SS    Секунды (0-59).
                SSSSS    Секунды после полуночи (0-86399).
                FF    Дробные секунды.


24.1) NOW()-- полезная функция, которая позволяет получать текущую дату и время

                                SELECT NOW()

                                    Результат:
                                    17/12/22 19:32

24.2) FROM_UNIXTIME( )  -- для переводва времении в секундах Unix-время 
                      FROM_UNIXTIME(1598291490) =  2020-08-24 17:51:30

                      -- ну или формула
                      1970-01-01 + time_unix / 86400

24.3) SEC_TO_TIME()  -- Для перевода количества секунд во временной формат
                    
                        SEC_TO_TIME(288) = 0:04:48

25) unnest -- Функция unnest предназначена для разворачивания массивов и превращения их в набор строк:

                                   SELECT unnest(ARRAY['one','two','three'])

                                        Результат:
                                        one
                                        two
                                        three

25.1) array_agg -- это продвинутая агрегирующая функция, которая собирает все значения в указанном столбце в единый список (ARRAY). 
                По сути array_agg — это операция, обратная unnest, 
                её синтаксис ничем не отличается от синтаксиса остальных агрегирующих функций:
                                    SELECT column_1, array_agg(column_2) AS new_array
                                      FROM table
                                    GROUP BY column_1
25.2) array_length -- длина массива Для расчёта числа товаров в заказах воспользуйтесь функцией array_length

                                    SELECT array_length(ARRAY[0, 1, 2], 1);

25.3) array_to_string(array[], ';') -- позволяет преобразовать массив в строку: первым параметром указывается массив, 
                                    -- вторым — удобный нам разделитель в одинарных кавычках (апострофах). 
                                    -- В качестве разделителя можно использовать
                                    -- Табуляция \t — к примеру, позволит при вставки ячейки в EXCEL без усилий разбить 
                                    -- значения на столбцы (использовать так: array_to_string(array[], E'\t') )
                                    -- Перевод строки \n — разложит значения массива по строкам в одной ячейке 
                                    -- (использовать так: array_to_string(array[], E'\n') — объясню ниже почему)

26) UNION -- объединяет записи из двух запросов в один общий результат (объединение множеств).

        При этом по умолчанию эти операции исключают из результата строки-дубликаты. 
        Чтобы дубликаты не исключались из результата, необходимо после имени операции указать ключевое слово ALL
                                        SELECT column_1, column_2
                                        FROM table_1
                                        UNION ALL  
                                        SELECT column_1, column_2
                                        FROM table_2
26.1)    EXCEPT  - вычетание , возвращает все записи, которые есть в первом запросе, но отсутствуют во втором (разница множеств).

26.2)    INTERSECT  - пересечение, возвращает все записи, которые есть и в первом, и во втором запросе (пересечение множеств).

27)  Использование SUM и CASE WHEN вместе 
                                        select 
                                            sum(case when allergies = 'Penicillin' and city = 'Burlington' then 1 else 0 end) as allergies_burl
                                        , sum(case when allergies = 'Penicillin' and city = 'Oakville' then 1 else 0 end)   as allergies_oak
                                        
                                        from patients
28)  CASE WHEN в WHERE

                                    select
                                    * 
                                    FROM patients
                                    WHERE TRUE
                                    and 1 = (case when allergies = 'Penicillin' and city = 'Burlington' then 1 else 0 end)

29) обьеденяем два столбца в один массив

                                  SELECT string_to_array(CONCAT(name_1,', ', name_2), ', ') AS pair

30) Метрики
    1. ARPU (Average Revenue Per User) — средняя выручка на одного пользователя за определённый период.
    2. ARPPU (Average Revenue Per Paying User) — средняя выручка на одного платящего пользователя за определённый период.
    3. AOV (Average Order Value) — средний чек, или отношение выручки за определённый период к общему количеству заказов за это же время.
    4. CAC (Customer Acquisition Cost), которая отражает затраты на привлечение одного покупателя.
    5. ROI (Return on Investment), в маркетинге её часто применяют для подсчёта окупаемости рекламных кампаний
    6. Retention rate — коэффициент удержания клиентов. Он показывает долю пользователей, которые вернулись в приложение спустя N дней
                    
31) CAST CONVERT --Таблица приоритетов (чем выше, тем больший приоритет):
 SQL Server может выполнять неявные преобразования от типа с меньшим приоритетом к типу с большим приоритетом

                datetime
                smalldatetime
                float
                real
                decimal
                money
                smallmoney
                int
                smallint
                tinyint
                bit
                nvarchar
                nchar
                varchar
                char

    Ккогда необходимо выполнить преобразования от типов с высшим приоритетом к типам с низшим приоритетом,
     то надо выполнять явное приведение типов.
                CAST(выражение AS тип_данных)
                    CAST(CreatedAt AS nvarchar) + '; total: ' + CAST(Price * ProductCount AS nvarchar) 

                CONVERT(тип_данных, выражение [, стиль])
                    CONVERT(nvarchar, CreatedAt, 3), 
                    CONVERT(nvarchar, Price * ProductCount, 1)
                стиль форматирования данных:
                            0 или 100 - формат даты "Mon dd yyyy hh:miAM/PM" (значение по умолчанию)

                            1 или 101 - формат даты "mm/dd/yyyy"

                            3 или 103 - формат даты "dd/mm/yyyy"

                            7 или 107 - формат даты "Mon dd, yyyy hh:miAM/PM"

                            8 или 108 - формат даты "hh:mi:ss"

                            10 или 110 - формат даты "mm-dd-yyyy"

                            14 или 114 - формат даты "hh:mi:ss:mmmm" (24-часовой формат времени)

                            Некоторые значения для форматирования данных типа money в строку:

                            0 - в дробной части числа остаются только две цифры (по умолчанию)

                            1 - в дробной части числа остаются только две цифры, а для разделения разрядов применяется запятая

                            2 - в дробной части числа остаются только четыре цифры
            
32) DDL -- data definition language - язык определения данных.
         Сюда входят те команды, которые нужны для изменения структуры БД. (CREATE, ALTER, DROP, RENAME и др.)
    
    DML -- (data manipulation language). 
            Это операции типа SELECT, INSERT, UPDATE, DELETE.

    DCL -- Data Control Language операторы для администрирования БД 
           GRANT -- предоставляет пользователю или группе разрешения на определённые операции с объектом;
           REVOKE -- отзывает выданные разрешения;
           DENY -- задаёт запрет, имеющий приоритет над разрешением
    
    TCL -- Transaction Control Language операторы для работы с транзакциями
            BEGIN TRANSACTION – служит для определения начала транзакции;
            COMMIT TRANSACTION – применяет транзакцию;
            ROLLBACK TRANSACTION – откатывает все изменения, сделанные в контексте текущей транзакции;
            SAVE TRANSACTION – устанавливает промежуточную точку сохранения внутри транзакции.

33) CREATE INDEX -- создаем индексы

                        CREATE INDEX idx1 ON Booking(flight_id);
                        CREATE INDEX idx2 ON Flight(planet_id);
                        ----
                        SELECT COUNT(1)
                        FROM Flight F JOIN Spacecraft S ON F.spacecraft_id = S.id
                        JOIN Planet P ON F.planet_id = P.id
                        JOIN Booking B ON F.id = B.flight_id
                        WHERE S.capacity<_capacity AND S.class=1
                        AND P.name=_planet_name;

34) -- создаем функцию

                        SELECT Price.planet_id, 
                                Price.spacecraft_class, 
                                Price.price * GetPaxCount(Price.planet_id, Price.spacecraft_class) AS takings 
                          FROM Price;


                        -- Вспомогательная функция, считающая количество пассажиров, летевших 
                        -- на планету _planet_id в звездолете класса _class

                        CREATE OR REPLACE FUNCTION GetPaxCount(_planet_id INT, _class INT) RETURNS BIGINT AS $$
                        SELECT COUNT(Pax.id)
                        FROM Planet P 
                        JOIN Flight F     ON P.id=F.planet_id
                        JOIN Booking B    ON B.flight_id = F.id
                        JOIN Spacecraft S ON F.spacecraft_id = S.id
                        JOIN Pax          ON B.pax_id = Pax.id
                        WHERE S.class = _class AND P.id = _planet_id;
                        $$ LANGUAGE SQL;

35) Генерация строк

                      WITH t1 (RowNumber) AS (
                                                SELECT 1 AS RowNumber
                                                UNION ALL
                                                SELECT a.RowNumber + 1    AS RowNumber
                                                FROM t1 a
                                                WHERE a.RowNumber < 7)


36) Чтобы проверить, есть ли ключевое слово в заголовке шага, можно использовать функцию:

                     INSTR(string_1, string_2)
    -- которая возвращает позицию первого вхождения string_2 в string_1. 
    -- Если вхождения нет - результат функции 0.

36)Возможности SQLite, которые вы могли пропустить

Частичные индексы (Partial Indexes) -- При построении индекса можно указать условие попадания строки в индекс, к примеру, 
                                    -- одна из колонок не пустая, а другая равна заданному значению.

create index idx_partial on tab1(a, b) where a is not null and b = 5;
select * from tab1 where a is not null and b = 5; --> search table tab1 using index

Индексы на выражение (Indexes On Expressions) -- Если в запросах к таблице часто используется выражение, 
                                              -- то можно построить индекс по нему. Однако следует иметь 
                                              -- в виду, что пока оптимизатор не очень гибок и перестановка столбцов 
                                              -- в выражении приведет к отказу от использования индекса.

create index idx_expression on tab1(a + b);
select * from tab1 where a + b > 10; --> search table tab1 using index ...
select * from tab1 where b + a > 10; --> scan table

Вычисляемые колонки (Generated Columns) -- Если данные столбца представляют собой результат вычисления выражения 
                                        -- по другим столбцам, то можно создать виртуальный столбец. 
                                        -- Есть два вида: VIRTUAL (вычисляется каждый раз при 
                                        -- чтении таблицы и не занимает места) и STORED (вычисляется при записи 
                                        -- данных в таблицу и место занимает).
                                        -- Разумеется записывать данные в такие столбцы напрямую нельзя.

create table tab1 (
    a integer primary key,
    b int,
    c text,
    d int generated always as (a * abs(b)) virtual,
    e text generated always as (substr(c, b, b + 1)) stored
);

R-Tree индекс -- Индекс предназначен для быстрого поиска в диапазоне значений/вложенности объектов,
              --  т.е. задачи типичной для гео-систем, когда объекты-прямоугольники заданы своей 
              -- позицией и размером и требуется найти все объекты, которые пересекаются с текущим.
              --  Данный индекс реализован в виде виртуальной таблицы (см. ниже) и это индекс только 
              -- по своей сути. Для поддержки R-Tree индекса
              --  требуется собрать SQLite с флагом SQLITE_ENABLE_RTREE (по умолчанию не установлен).

create virtual table idx_rtree using rtree (
    id,              -- ключ
    minx, maxx,      -- мин и макc x координаты
    miny, maxy,      -- мин и макc y координаты
    data             -- дополнительные данные  
);  

insert into idx_rtree values (1, -80.7749, -80.7747, 35.3776, 35.3778); 
insert into idx_rtree values (2, -81.0, -79.6, 35.0, 36.2);

select id from idx_rtree 
where minx >= -81.08 and maxx <= -80.58 and miny >= 35.00  and maxy <= 35.44;

Возвращаемые значения (Returning) -- С версии 3.35.0 можно вернуть значения из операторов update, insert и delete.

insert into t (a, b) values ('x', 'y') returning id;
update t set a = a * 2 where b < 10 returning a, b as c;

Переименование и удаление колонки -- В SQLite слабо поддерживает изменения в структуре таблиц, так, 
                                  -- после создания таблицы, нельзя изменить ограничение (constraint) 
                                  -- или положение колонки. С версии 3.25.0 можно переименовать столбец, 
                                -- но не изменить его тип.

alter table tbl1 rename column a to b;

С версии 3.35.0 можно удалить столбец.

alter table tbl1 drop column a;

Для других операций всё также предлагается создать таблицу с нужной структурой, 
перелить туда данные, удалить старую и переименовать новую.

Добавить строку, иначе обновить (Upsert) -- Используя класс on conflict оператора insert, 
                                         -- можно добавить новую строку,
                                         --  а при уже имеющейся с таким же значением по ключу, обновить.

create table vocabulary (word text primary key, count int default 1);
insert into vocabulary (word) values ('jovial') 
  on conflict (word) do update set count = count + 1;

Оператор Update from -- Если строка должна быть обновлена на основе данных другой таблицы, 
                     -- то ранее приходилось использовать вложенный запрос для каждого 
                     -- столбца или with. С версии 3.33.0 оператор update расширен 
                     -- ключевым словом from и теперь можно делать так

update inventory
   set quantity = quantity - daily.amt
  from (select sum(quantity) as amt, itemid from sales group by 2) as daily
 where inventory.itemid = daily.itemid;

CTE запросы, класс with (Common Table Expression) -- Класс with может использоваться как 
                                                -- временное представление для запроса. 
                                                -- В версии 3.34.0 заявлена возможность 
                                                -- использования with внутри with.

with tab2 as (select * from tab1 where a > 10), 
  tab3 as (select * from tab2 inner join ...)
select * from tab3;

-- Явное указание имен для столбцов
with tab2 (a, b) as (select col1, col2 from tab1 where ...)
select a, b from tab2;

С добавлением ключевого слова recursive, with можно использовать для запросов,
 где требуется оперировать связанными данными.

-- Генерация значений
with recursive cnt(x) as (
  values(1) union all select x + 1 from cnt where x < 1000
)
select x from cnt;

-- Нахождения дочерних элементов или родителя в таблице с иерархией
-- Узлы ниже по иерархии
with recursive tc (id) as (
    select id from tab1 where id = 10    
    union 
    select tab1.id from tab1, tc where tab1.parent_id = tc.id
)
select * from tc;

-- Узелы верхнего уровня для выбранных дочерних
with recursive tc (id, parent_id) as (
    select id, parent_id from tab1 where id in (12, 21)
    union 
    select tc.parent_id, tab1.parent_id 
    from tab1, tc where tab1.id = tc.parent_id
)
select distinct id from tc where parent_id is null order by 1;

-- Формирования отступов при выводе, напр. для структуры отделов
create table org(name text primary key, boss text references org);
insert into org values ('Alice', null), 
  ('Bob', 'Alice'), ('Cindy', 'Alice'), ('Dave', 'Bob'), 
  ('Emma', 'Bob'), ('Fred', 'Cindy'), ('Gail', 'Cindy');

with recursive
  under_alice (name, level) as (
    values('Alice', 0)
    union all
    select org.name, under_alice.level + 1
      from org join under_alice on org.boss = under_alice.name
     order by 2
  )
select substr('..........', 1, level * 3) || name from under_alice;

В версии 3.35.0 введено ключевое слово MATERIALIZED, 
при котором планировщик для результатов CTE-запроса создаст «эфемерную» таблицу.
 По умолчанию оно есть в запросе, и для отключения такого поведения необходимо использовать NOT MATERIALIZED.

with tab2 (a, b) as materialized (
    select col1, col2 from tab1 where ...
)
select a, b from tab2;

Оконные функции (Window Functions)
С версии 3.25.0 в SQLite доступны оконные функции, 
также иногда называемые аналитическими, позволяющие проводить вычисления над частью данных (окном).

-- Номер строки в результате
create table tab1 (x integer primary key, y text);
insert into tab1 values (1, 'aaa'), (2, 'ccc'), (3, 'bbb');
select x, y, row_number() over (order by y) as row_number from tab1 order by x;

-- Таблица используется для следующих примеров
create table tab1 (a integer primary key, b, c);
insert into tab1 values (1, 'A', 'one'),
  (2, 'B', 'two'), (3, 'C', 'three'), (4, 'D', 'one'), 
  (5, 'E', 'two'), (6, 'F', 'three'), (7, 'G', 'one');

-- Доступ к предыдущей и следующей записи в окне
select a, b, group_concat(b, '.') over (order by a rows between 1 preceding and 1 following) as prev_curr_next from tab1;

-- Значения в окне (группе, определяемой колонкой c)  от текущей строки до конца окна
select c, a, b, group_concat(b, '.') over (partition by c order by a range between current row and unbounded following) as curr_end from tab1 order by c, a;

-- Пропуск строк в окне по условию
select c, a, b, group_concat(b, '.') filter (where c <> 'two') over (order by a) as exceptTwo from t1 order by a;

Утилиты SQLite -- Помимо CLI sqlite3 доступны еще две утилиты. 
Первая — sqldiff, позволяет сравнивать базы (или отдельную таблицу) не только по структуре, но и по данным. 
Вторая — sqlite3_analizer используется для вывода информации о том, как эффективно используется место 
таблицами и индексами в файле базы данных. Аналогичную информацию можно получить из виртуальной 
таблицы dbstat (требует флаг SQLITE_ENABLE_DBSTAT_VTAB при компиляции SQLite).

С версии 3.22.0 CLI sqlite3 содержит (экспериментальную) команду .expert, которая может подсказать 
какой индекс стоит добавить для вводимого запроса.

Создание резервной копии Vacuum Into
С версии 3.27.0 команда vacuum расширена ключевым словом into, позволяющим создать 
копию базы без её остановки прямо из SQL. Является простой альтернативой Backup API.

vacuum into 'D:/backup/' || strftime('%Y-%m-%d', 'now') || '.sqlite';

Функция printf -- Функция является аналогом С-функции. При этом NULL-значения интерпретируются 
               -- как пустая строка для %s и 0 для плейсхолдера числа.

select 'a' || ' 123 ' || null; --> null
select printf('%s %i %s', 'a', 123, null); --> 123 a
select printf('%s %i %i', 'a', 123, null); --> 123 a 0

Кортежи столбцов (Row values)

delete from tab where (a, b) = (1, 2); -- то же самое что a = 1 and b = 2
update tab set c = 3 where (a, b) = (1, 2);
update tab set c = 3 where (a, b) in (select c, d from tab2);
select * from tab where (year, month, day) between (2015, 9, 12) and (2016, 9, 12);

Время и дата -- В SQLite нет типов Date и Time. 
             -- Хотя и можно создать таблицу с колонками таких типов, это будет аналогично созданию колонок 
             -- без указания типа, поэтому данные в таких колонках хранятся как текст.
            -- Это удобно при просмотре данных, однако имеет ряд недостатков: неэффективный поиск, 
            -- если нет индекса, данные занимают много места, отсутсвует временная зона.
            -- Для избежания этого можно хранить данные как unix-время, т.е. число секунд, прошедших с полуночи 01.01.1970.

select strftime('%Y-%m-%d %H:%M', 'now'); --> UTC время
select strftime('%Y-%m-%d %H:%M', 'now', 'localtime'); --> местное время
select strftime('%s', 'now'); -- текущее Unix-время 
select strftime('%s', 'now', '+2 day'); --> текущее unix-время плюс два дня
-- Конвертация unix-времени в локальное для пользователя - 21-11-2020 15:25:14
select strftime('%d-%m-%Y %H:%M:%S', 1605961514, 'unixepoch', 'localtime')

Json
С версии 3.9.0 в SQLite можно работать с json (требуется либо флаг SQLITE_ENABLE_JSON1 при компиляции или загруженное расширение).
 Данные json хранятся как текст. Результат функций — также текст.

select json_array(1, 2, 3); --> [1,2,3] (строка)
select json_array_length(json_array(1, 2, 3)); --> 3
select json_array_length('[1,2,3]'); --> 3
select json_object('a', json_array(2, 5), 'b', 10); --> {"a":[2,5],"b":10} (строка)
select json_extract('{"a":[2,5],"b":10}', '$.a[0]');  --> 2
select json_insert('{"a":[2,5]}', '$.c', 10); --> {"a":[2,5],"c":10} (строка)
select value from json_each(json_array(2, 5)); --> 2 строки 2, 5
select json_group_array(value) from json_each(json_array(2, 5)); --> [2,5] (строка)
А используя вычисляемые колонки и индексы по ним, эти запросы можно ускорить.

Полнотекстовый поиск
Как и json, полнотекстовый поиск требует задания флага SQLITE_ENABLE_FTS5 при компиляции или загрузки расширения. 
Для работы с поиском, сперва создается виртуальная таблица с индексируемыми полями, 
а и потом туда загружаются данные, используя обычный insert. Следует иметь в виду, что для своей работы
 расширение создает дополнительные таблицы и созданная виртуальная таблица использует их данные.

create virtual table emails using fts5(sender, body);
SELECT * FROM emails WHERE emails = 'fts5'; -- sender или body содержит fts5

Расширения
Возможности SQLite могут быть добавлены через загружаемые модули. Некоторые из них уже были упомянуты выше — json1 и fts.

Расширения могут использоваться как для добавления пользовательских функций
 (не только скалярных, как, например, crc32, но и агрегирующих или даже оконных), 
 так и виртуальных таблиц. Виртуальные таблицы — это таблицы, которые присутствуют в базе, 
 но их данные обрабатываются расширением, при этом, в зависимости от реализации, некоторые из них требуют создания

create virtual table temp.tab1 using csv(filename='thefile.csv');
select * from tab1;

Другие же, так называемые table-valued, могут использоваться сразу

select value from generate_series(5, 100, 5);
.
Часть виртуальных таблиц перечислена здесь.

Одно расширение может реализовать как функции, так и виртуальные таблицы. 
Например, json1 содержит 13 скалярных и 2 агрегирующие функции и 
две виртуальные таблицы json_each и json_tree.
 Чтобы написать свою функцию достаточно иметь базовые знания С и разобрать
  код расширений из репозитария SQLite. Реализация своих виртуальных 
  таблиц несколько сложнее (видимо поэтому их мало). Тут можно рекомендовать не сильно 
  устаревшую книгу Using SQLite by Jay A. Kreibich, статью Michael Owens,
   шаблон из репозитария и код generate_series, как table-valued функции.

Помимо этого, расширения могут реализовать специфичные для операционной системы вещи, 
такие как файловая система, обеспечивающие портируемость. Подробности можно узнать здесь.

Разное

Используйте ' (одинарная кавычка) для строковых констант и " (двойная кавычка) для имен столбцов и таблиц.
Чтобы получить информацию по таблице tab1 можно использовать

-- В main схеме
select * from pragma_table_info('tab1');
-- В temp схеме или подключенной (attach) базе
select * from pragma_table_info('tab1') where schema = 'temp'
  