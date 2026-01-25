package model

type Habit struct {
	ID            string   `json:"id" db:"id"`
	Title         string   `json:"title" db:"title"`
	Description   string   `json:"description,omitempty" db:"description"`
	Color         string   `json:"color" db:"color"`
	Icon          string   `json:"icon,omitempty" db:"icon"`
	TargetDays    int      `json:"targetDays,omitempty" db:"target_days"`
	DailyGoal     int      `json:"dailyGoal,omitempty" db:"daily_goal"`
	PreferredTime string   `json:"preferredTime,omitempty" db:"preferred_time"`
	Category      string   `json:"category,omitempty" db:"category"`
	ScheduleType  string   `json:"scheduleType" db:"schedule_type"`           // "recurring" or "one_time"
	RecurringDays []int    `json:"recurringDays,omitempty" db:"recurring_days"` // Array of weekdays: 0=Sunday, 1=Monday, ..., 6=Saturday
	OneTimeDate   string   `json:"oneTimeDate,omitempty" db:"one_time_date"`   // Date for one-time habits
	IsActive      bool     `json:"isActive" db:"is_active"`
	UserID        string   `json:"userId" db:"user_id"`
	WorkspaceID   string   `json:"workspaceId" db:"workspace_id"`
	CreatedAt     string   `json:"createdAt" db:"created_at"`
	UpdatedAt     string   `json:"updatedAt" db:"updated_at"`
}

type HabitCompletion struct {
	ID        string `json:"id" db:"id"`
	HabitID   string `json:"habitId" db:"habit_id"`
	UserID    string `json:"userId" db:"user_id"`
	Date      string `json:"date" db:"date"`
	Notes     string `json:"notes,omitempty" db:"notes"`
	Rating    int    `json:"rating,omitempty" db:"rating"`
	Time      string `json:"time,omitempty" db:"time"`
	CreatedAt string `json:"createdAt" db:"created_at"`
}

type CreateHabitDto struct {
	Title         string   `json:"title" binding:"required"`
	Description   string   `json:"description,omitempty"`
	Color         string   `json:"color,omitempty"`
	Icon          string   `json:"icon,omitempty"`
	TargetDays    int      `json:"targetDays,omitempty"`
	DailyGoal     int      `json:"dailyGoal,omitempty"`
	PreferredTime string   `json:"preferredTime,omitempty"`
	Category      string   `json:"category,omitempty"`
	ScheduleType  string   `json:"scheduleType" binding:"required,oneof=recurring one_time"` // "recurring" or "one_time"
	RecurringDays []int    `json:"recurringDays,omitempty"`                                   // For recurring: array of weekdays (0-6)
	OneTimeDate   string   `json:"oneTimeDate,omitempty"`                                     // For one_time: specific date (YYYY-MM-DD)
	IsActive      *bool    `json:"isActive,omitempty"`                                       // Optional, defaults to true
}

type UpdateHabitDto struct {
	Title         *string  `json:"title,omitempty"`
	Description   *string  `json:"description,omitempty"`
	Color         *string  `json:"color,omitempty"`
	Icon          *string  `json:"icon,omitempty"`
	TargetDays    *int     `json:"targetDays,omitempty"`
	DailyGoal     *int     `json:"dailyGoal,omitempty"`
	PreferredTime *string  `json:"preferredTime,omitempty"`
	Category      *string  `json:"category,omitempty"`
	ScheduleType  *string  `json:"scheduleType,omitempty" binding:"omitempty,oneof=recurring one_time"`
	RecurringDays *[]int   `json:"recurringDays,omitempty"`
	OneTimeDate   *string  `json:"oneTimeDate,omitempty"`
	IsActive      *bool    `json:"isActive,omitempty"`
}

type HabitStats struct {
	HabitID        string  `json:"habitId"`
	CompletedDays  int     `json:"completedDays"`
	TotalDays      int     `json:"totalDays"`
	CompletionRate float64 `json:"completionRate"`
	CurrentStreak  int     `json:"currentStreak"`
	LongestStreak  int     `json:"longestStreak"`
}

type CalendarDay struct {
	Date   string `json:"date"`
	Habits []struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Completed bool   `json:"completed"`
		Color     string `json:"color"`
	} `json:"habits"`
}

type CalendarResponse struct {
	Days []CalendarDay `json:"days"`
}

type ToggleResponse struct {
	Completed  bool             `json:"completed"`
	Completion *HabitCompletion `json:"completion,omitempty"`
}

// HabitHistory - история изменений привычки
type HabitHistory struct {
	ID        string                 `json:"id" db:"id"`
	HabitID   string                 `json:"habitId" db:"habit_id"`
	UserID    string                 `json:"userId" db:"user_id"`
	Action    string                 `json:"action" db:"action"` // CREATED, UPDATED, DELETED, COMPLETED
	Changes   map[string]interface{} `json:"changes,omitempty" db:"changes"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt string                 `json:"createdAt" db:"created_at"`
}

// Activity - активность пользователя для виджета RecentActivity
type Activity struct {
	ID          string `json:"id" db:"id"`
	UserID      string `json:"userId" db:"user_id"`
	WorkspaceID string `json:"workspaceId" db:"workspace_id"`
	Type        string `json:"type" db:"type"` // HABIT_CREATED, HABIT_UPDATED, HABIT_DELETED, HABIT_COMPLETED
	EntityType  string `json:"entityType" db:"entity_type"` // habit, completion, workspace
	EntityID    string `json:"entityId" db:"entity_id"`
	Title       string `json:"title" db:"title"`
	Emoji       string `json:"emoji,omitempty" db:"emoji"`
	CreatedAt   string `json:"createdAt" db:"created_at"`
}
