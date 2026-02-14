# Habits Repository

Репозиторий для работы с привычками: CRUD, выполнение, календарь, статистика. Использует версионирование привычек для исторического календаря.

---

## 1. Структура пакета

| Файл | Назначение |
|------|------------|
| `repository.go` | Основной репозиторий — CRUD привычек, календарь, делегирование в подкомпоненты |
| `version_repository.go` | Управление версиями привычек (`habit_versions`) |
| `completion_repository.go` | Выполнения привычек (`habit_completions`) |
| `stats_calculator.go` | Расчёт streaks и статистики |
| `scanner.go` | Сканирование SQL-результатов в модели |
| `utils.go` | Утилиты: нормализация дат, конвертация времени |

---

## 2. Основной Repository (`repository.go`)

### 2.1 Конструктор

- **`NewRepository(db *sql.DB)`** — создаёт `Repository` с инициализированными `VersionRepository`, `CompletionRepository`, `StatsCalculator`.

### 2.2 Методы

- **`List(ctx, workspaceID, targetDate)`**  
  - Возвращает все привычки воркспейса.  
  - Если `targetDate != nil` — делегирует в `GetHabitsForDate`.  
  - Иначе — выборка из `habits` по `workspace_id`, сортировка по `preferred_time`, `created_at`.

- **`GetHabitsForDate(ctx, workspaceID, targetDate)`**  
  - Основной источник: `habit_versions` — привычки, активные на дату (расписание, `valid_from`/`valid_to`).  
  - Fallback для «сегодня» и будущего: `habits`, если нет версий.  
  - Для прошлых дат fallback не используется.

- **`Create(ctx, dto, userID, workspaceID)`**  
  - Вставка в `habits`.  
  - Типы расписания: `recurring` (дни недели, 0–6) или `one_time` (одна дата).  
  - Дефолты: `target_days` по количеству дней, `daily_goal = 1`, `color = #3B82F6`.  
  - Создаёт стартовую версию в `habit_versions` с `valid_from = дата создания`.

- **`Get(ctx, id, userID)`**  
  - Одна привычка по `id` и `user_id` из `habits`.

- **`Update(ctx, id, userID, dto)`**  
  - Частичное обновление по непустым полям DTO.  
  - При изменении полей, влияющих на историю: закрывает текущую версию (`valid_to = сегодня`), создаёт новую с `valid_from = завтра`.  
  - Если версий не было — создаёт backfill от `created_at` до сегодня.

- **`Delete(ctx, id, userID)`**  
  - Транзакция: закрывает версию (`valid_to`), удаляет запись из `habits`.

- **`Complete(ctx, habitID, userID, date, notes, rating, completionTime)`**  
  - Делегирует в `CompletionRepository.Create`.

- **`Toggle(ctx, habitID, userID, date)`**  
  - Добавляет или удаляет completion на дату. Делегирует в `CompletionRepository.Toggle`.

- **`GetStats(ctx, habitID, userID)`**  
  - `CompletedDays`, `TotalDays`, `CompletionRate`, `CurrentStreak`, `LongestStreak`.  
  - Использует `CompletionRepository` и `StatsCalculator`.

- **`GetCompletions(ctx, habitID, userID, startDate, endDate)`**  
  - Completions одной привычки за период.

- **`GetAllCompletions(ctx, userID, workspaceID, startDate, endDate)`**  
  - Completions всего воркспейса за период.

- **`GetCalendar(ctx, userID, workspaceID, startDate, endDate)`**  
  - Календарь: для каждой даты список привычек с флагом `completed`.  
  - Берёт привычки из `GetHabitsForDate`, completion-флаги — из `GetCompletionMap`.  
  - Для прошлых дат добавляет «сироты» (есть completion, но привычка не попала в день) по данным из `VersionRepository.GetForDate`.

---

## 3. VersionRepository (`version_repository.go`)

Управление таблицей `habit_versions` для исторического календаря.

### 3.1 Методы

- **`Create(ctx, habitID, userID, workspaceID, ...)`**  
  - Вставка версии с полями: `title`, `description`, `color`, `icon`, `target_days`, `daily_goal`, `preferred_time`, `category`, `schedule_type`, `recurring_days`, `one_time_date`, `is_active`, `valid_from`.  
  - Конвертирует `preferredTime` (morning/afternoon/evening) в время БД.

- **`GetForDate(ctx, habitID, workspaceID, targetDate)`**  
  - Возвращает `(id, title, color, ok)` версии на дату.  
  - Условие: `targetDate` в `[valid_from, COALESCE(valid_to, targetDate)]`.

- **`ClosePrevious(ctx, habitID, userID, workspaceID, validTo)`**  
  - Устанавливает `valid_to` для открытой версии (`valid_to IS NULL`).  
  - Возвращает количество обновлённых строк.

---

## 4. CompletionRepository (`completion_repository.go`)

Работа с `habit_completions`.

### 4.1 Методы

- **`Create(ctx, habitID, userID, date, notes, rating, completionTime)`**  
  - Вставка completion. `workspace_id` берётся из `habits`.  
  - Даты нормализуются через `NormalizeDate`.

- **`Toggle(ctx, habitID, userID, date)`**  
  - Если completion есть — удаляет, возвращает `(false, existingCompletion)`.  
  - Иначе создаёт, возвращает `(true, newCompletion)`.

- **`GetByHabitAndDateRange(ctx, habitID, userID, startDate, endDate)`**  
  - Completions привычки за период, сортировка по `date DESC`, `time DESC`.

- **`GetCompletionDates(ctx, habitID, userID)`**  
  - Уникальные даты выполнений (для streaks).

- **`CountByHabit(ctx, habitID, userID)`**  
  - Количество выполнений привычки.

- **`GetCompletionMap(ctx, userID, workspaceID, startDate, endDate)`**  
  - `map[dateKey][habitID]bool` для быстрой проверки completion в календаре.

- **`GetAllByWorkspaceAndDateRange(ctx, userID, workspaceID, startDate, endDate)`**  
  - Completions воркспейса за период.

---

## 5. StatsCalculator (`stats_calculator.go`)

Расчёт streaks с учётом расписания привычки.

### 5.1 Структуры

- **`HabitScheduleInfo`**  
  - `ScheduleType`, `RecurringDays`, `OneTimeDate`, `CreatedAtUTC`.

### 5.2 Методы

- **`CalculateStreaks(completionDates, info)`**  
  - **CurrentStreak**: от сегодня назад по активным дням, пока есть выполнения подряд.  
  - **LongestStreak**: максимальная серия дней с выполнениями среди активных дней.  
  - «Активный день»: входит в расписание (recurring — день недели в `RecurringDays`; one_time — дата совпадает с `OneTimeDate`) и не раньше `CreatedAtUTC`.

---

## 6. Утилиты (`utils.go`)

- **`NormalizeDate(t)`**  
  - Приводит дату к началу дня в UTC.

- **`ConvertPreferredTimeToTime(preferredTime)`**  
  - `"morning"` → `"08:00:00"`, `"afternoon"` → `"14:00:00"`, `"evening"` → `"20:00:00"`.  
  - Если передано `HH:MM` / `HH:MM:SS` — возвращает как есть.

- **`ConvertTimeToPreferredTime(timeStr)`**  
  - Обратная конвертация: `08:00:00` → `"morning"` и т.д.

- **`ConvertRecurringDays(daysArray)`**  
  - `pq.Int32Array` → `[]int`.

---

## 7. Scanner (`scanner.go`)

- **`scanHabits(rows)`**  
  - Сканирует `*sql.Rows` в `[]model.Habit`.  
  - Заполняет `PreferredTime`, `Category`, `OneTimeDate`, `RecurringDays`, `CreatedAt`, `UpdatedAt`.

- **`scanCompletions(rows)`**  
  - Сканирует в `[]model.HabitCompletion`.  
  - Обрабатывает nullable `rating`, `time`.

---

## 8. Таблицы БД

| Таблица | Описание |
|---------|----------|
| `habits` | Текущее состояние привычки (основная запись) |
| `habit_versions` | Версии привычки во времени (`valid_from`, `valid_to`) |
| `habit_completions` | Выполнения (habit_id, user_id, date, notes, rating, time) |

---

## 9. Логика версионирования

1. При **создании** — одна версия с `valid_from = дата создания`, `valid_to = NULL`.
2. При **обновлении** — старые версии закрываются (`valid_to = сегодня`), новая — с `valid_from = завтра`. Для привычек без версий — backfill.
3. При **удалении** — закрывается текущая версия, удаляется запись из `habits`.
4. Календарь использует `habit_versions` как основной источник: для даты D показываются привычки, у которых есть версия с `valid_from <= D <= valid_to` и подходящим расписанием.

---

## 10. Типы расписания

- **`recurring`**  
  - `recurring_days`: дни недели 0–6 (0=Вс, 6=Сб).  
  - Пустой массив → все дни.  
  - `target_days` по умолчанию = число дней в `recurring_days`.

- **`one_time`**  
  - `one_time_date`: конкретная дата.  
  - `target_days = 1`.
