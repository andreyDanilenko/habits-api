package model

// Task — задача в workspace.
type Task struct {
	ID            string  `json:"id" db:"id"`
	WorkspaceID   string  `json:"workspaceId" db:"workspace_id"`
	Title         string  `json:"title" db:"title"`
	Description   string  `json:"description" db:"description"`
	Type          string  `json:"type" db:"type"`
	Priority      string  `json:"priority" db:"priority"`
	Status        string  `json:"status" db:"status"`
	DueDate       string  `json:"dueDate" db:"due_date"`
	DueTime       string  `json:"dueTime,omitempty" db:"due_time"`
	ReminderDate  string  `json:"reminderDate,omitempty" db:"reminder_date"`
	Duration      *int    `json:"duration,omitempty" db:"duration"`
	CompletedAt   string  `json:"completedAt,omitempty" db:"completed_at"`
	CompletedBy   string  `json:"completedBy,omitempty" db:"completed_by"`
	CompletionNote string `json:"completionNote,omitempty" db:"completion_note"`
	IsRecurring   bool    `json:"isRecurring" db:"is_recurring"`
	RecurringPattern []byte `json:"recurringPattern,omitempty" db:"recurring_pattern"`
	ParentID      string  `json:"parentId,omitempty" db:"parent_id"`
	AssigneeID    string  `json:"assigneeId" db:"assignee_id"`
	CreatedBy     string  `json:"createdBy" db:"created_by"`
	CreatedAt     string  `json:"createdAt" db:"created_at"`
	UpdatedAt     string  `json:"updatedAt" db:"updated_at"`
	SpentMinutes  int     `json:"spentMinutes" db:"spent_minutes"`
	SpentSeconds  int     `json:"spentSeconds" db:"spent_seconds"`
	Tags          []string `json:"tags,omitempty" db:"tags"`
	Entities      []TaskEntityLink `json:"entities,omitempty" db:"-"`
}

// TaskEntityLink — связь задачи с сущностью (crm_deal, crm_contact, crm_company).
type TaskEntityLink struct {
	EntityType string `json:"entityType" db:"entity_type"`
	EntityID   string `json:"entityId" db:"entity_id"`
	EntityName string `json:"entityName,omitempty" db:"entity_name"`
}

// CreateTaskDto — DTO для создания задачи.
type CreateTaskDto struct {
	Title        string           `json:"title" validate:"required,max=500"`
	Description  string           `json:"description"`
	Type         string           `json:"type" validate:"required,oneof=task bug feature meeting call email lunch other"`
	Priority     string           `json:"priority" validate:"required,oneof=low medium high critical"`
	Status       string           `json:"status" validate:"omitempty,oneof=pending in_progress completed cancelled"`
	DueDate      string           `json:"dueDate" validate:"required"`
	DueTime      string           `json:"dueTime"`
	ReminderDate string           `json:"reminderDate"`
	Duration     *int             `json:"duration"`
	AssigneeID   string           `json:"assigneeId" validate:"required,uuid"`
	ParentID     string           `json:"parentId"`
	Entities     []TaskEntityLink  `json:"entities"`
}

// UpdateTaskDto — DTO для обновления задачи.
type UpdateTaskDto struct {
	Title        *string          `json:"title"`
	Description  *string          `json:"description"`
	Type         *string          `json:"type"`
	Priority     *string          `json:"priority"`
	Status       *string          `json:"status"`
	DueDate      *string          `json:"dueDate"`
	DueTime      *string          `json:"dueTime"`
	ReminderDate *string          `json:"reminderDate"`
	Duration     *int             `json:"duration"`
	SpentMinutes *int             `json:"spentMinutes"`
	SpentSeconds *int             `json:"spentSeconds"`
	AssigneeID   *string          `json:"assigneeId"`
	Tags         []string         `json:"tags"`
	Entities     []TaskEntityLink `json:"entities"`
}

// CompleteTaskDto — DTO для отметки выполнения.
type CompleteTaskDto struct {
	Note string `json:"note"`
}

// TaskComment — комментарий к задаче (часть потока активности).
type TaskComment struct {
	ID        string `json:"id" db:"id"`
	TaskID    string `json:"taskId" db:"task_id"`
	ParentID  string `json:"parentId,omitempty" db:"parent_id"`
	Body      string `json:"body" db:"body"`
	CreatedBy string `json:"createdBy" db:"created_by"`
	CreatedAt string `json:"createdAt" db:"created_at"`
}

// CreateTaskCommentDto — DTO для создания комментария.
type CreateTaskCommentDto struct {
	Body     string `json:"body" validate:"required,max=10000"`
	ParentID string `json:"parentId"`
}

// UpdateTaskCommentDto — DTO для обновления комментария.
type UpdateTaskCommentDto struct {
	Body string `json:"body" validate:"required,max=10000"`
}

// TaskTaskLink — связь между задачами (blocks/blocked_by).
type TaskTaskLink struct {
	ID           string `json:"id" db:"id"`
	TaskID       string `json:"taskId" db:"task_id"`
	LinkedTaskID string `json:"linkedTaskId" db:"linked_task_id"`
	LinkType     string `json:"linkType" db:"link_type"` // "blocks" | "blocked_by"
	CreatedAt    string `json:"createdAt" db:"created_at"`
	// Денормализованные поля связанной задачи для отображения
	LinkedTitle    string `json:"linkedTitle" db:"-"`
	LinkedPriority string `json:"linkedPriority" db:"-"`
}

// CreateTaskTaskLinkDto — DTO для создания связи.
type CreateTaskTaskLinkDto struct {
	LinkedTaskID string `json:"linkedTaskId" validate:"required,uuid"`
	LinkType     string `json:"linkType" validate:"required,oneof=blocks blocked_by"`
}

// TaskAttachment — вложение к задаче.
type TaskAttachment struct {
	ID         string `json:"id" db:"id"`
	TaskID     string `json:"taskId" db:"task_id"`
	FileName   string `json:"fileName" db:"file_name"`
	FilePath   string `json:"-" db:"file_path"`
	FileSize   *int   `json:"fileSize,omitempty" db:"file_size"`
	MimeType   string `json:"mimeType,omitempty" db:"mime_type"`
	UploadedBy string `json:"uploadedBy" db:"uploaded_by"`
	CreatedAt  string `json:"createdAt" db:"created_at"`
	URL        string `json:"url" db:"-"` // URL для скачивания (формируется в handler)
}

// TaskListFilters — фильтры для списка задач.
type TaskListFilters struct {
	Status         string
	Priority       string
	Type           string // task, bug, feature, meeting, call, email, lunch, other
	AssigneeID     string
	EntityType     string
	EntityID       string
	ParentID       string // UUID — подзадачи родителя; "root" — только корневые (parent_id IS NULL)
	OverdueOnly    bool
	Search         string // поиск по title, description (ILIKE)
	Page           int
	Limit          int
}
