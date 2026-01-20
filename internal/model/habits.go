package model

type Habit struct {
	ID            string `json:"id" db:"id"`
	Title         string `json:"title" db:"title"`
	Description   string `json:"description,omitempty" db:"description"`
	Color         string `json:"color" db:"color"`
	Icon          string `json:"icon,omitempty" db:"icon"`
	TargetDays    int    `json:"targetDays,omitempty" db:"target_days"`
	DailyGoal     int    `json:"dailyGoal,omitempty" db:"daily_goal"`
	PreferredTime string `json:"preferredTime,omitempty" db:"preferred_time"`
	Category      string `json:"category,omitempty" db:"category"`
	UserID        string `json:"userId" db:"user_id"`
	WorkspaceID   string `json:"workspaceId" db:"workspace_id"`
	CreatedAt     string `json:"createdAt" db:"created_at"`
	UpdatedAt     string `json:"updatedAt" db:"updated_at"`
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
	Title         string `json:"title" binding:"required"`
	Description   string `json:"description,omitempty"`
	Color         string `json:"color,omitempty"`
	Icon          string `json:"icon,omitempty"`
	TargetDays    int    `json:"targetDays,omitempty"`
	DailyGoal     int    `json:"dailyGoal,omitempty"`
	PreferredTime string `json:"preferredTime,omitempty"`
	Category      string `json:"category,omitempty"`
}

type UpdateHabitDto struct {
	Title         string `json:"title,omitempty"`
	Description   string `json:"description,omitempty"`
	Color         string `json:"color,omitempty"`
	Icon          string `json:"icon,omitempty"`
	TargetDays    int    `json:"targetDays,omitempty"`
	DailyGoal     int    `json:"dailyGoal,omitempty"`
	PreferredTime string `json:"preferredTime,omitempty"`
	Category      string `json:"category,omitempty"`
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
