// seed_tasks создаёт тестовые задачи для указанного workspace.
//
// Использование:
//
//	WORKSPACE_ID=<uuid> USER_ID=<uuid> go run ./cmd/seed_tasks
//
// Пример (UUID из seed_users_roles):
//
//	WORKSPACE_ID=b2222222-2222-2222-2222-222222222201 USER_ID=a1111111-1111-1111-1111-111111111101 go run ./cmd/seed_tasks
//
// Опционально ASSIGNEE_ID — иначе используется USER_ID для всех задач.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"backend/internal/config"
	"backend/internal/database"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

func main() {
	workspaceID := os.Getenv("WORKSPACE_ID")
	userID := os.Getenv("USER_ID")
	assigneeID := os.Getenv("ASSIGNEE_ID")
	if workspaceID == "" || userID == "" {
		log.Fatal("WORKSPACE_ID and USER_ID environment variables are required. Example: WORKSPACE_ID=b2222222-2222-2222-2222-222222222201 USER_ID=a1111111-1111-1111-1111-111111111101 go run ./cmd/seed_tasks")
	}
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		log.Fatalf("Invalid WORKSPACE_ID: %v", err)
	}
	uID, err := uuid.Parse(userID)
	if err != nil {
		log.Fatalf("Invalid USER_ID: %v", err)
	}
	assignee := uID
	if assigneeID != "" {
		if a, err := uuid.Parse(assigneeID); err == nil {
			assignee = a
		}
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Проверка: workspace существует и у пользователя есть доступ
	var exists int
	err = db.QueryRowContext(ctx, `SELECT 1 FROM user_workspaces WHERE workspace_id = $1 AND user_id = $2`, wsID, uID).Scan(&exists)
	if err != nil || exists != 1 {
		log.Fatalf("Workspace %s not found or user %s has no access. Run seed_users_roles first.", wsID, uID)
	}

	now := time.Now()
	dueFmt := "2006-01-02T15:04:05Z"

	tasksData := []struct {
		title       string
		description string
		type_       string
		priority    string
		status      string
		dueDate     time.Time
		tags        []string
		spentMin    int
		spentSec    int
		completed   bool
	}{
		{"Настроить CI/CD пайплайн", "Настроить GitHub Actions для автотестов и деплоя", "task", "high", "in_progress", now.AddDate(0, 0, 3), []string{"devops", "важно"}, 0, 0, false},
		{"Исправить баг в форме логина", "При пустом пароле показывается 500 вместо валидации", "bug", "critical", "pending", now.AddDate(0, 0, 1), []string{"frontend"}, 0, 0, false},
		{"Добавить экспорт в Excel", "Экспорт списка контактов в xlsx", "feature", "medium", "pending", now.AddDate(0, 0, 7), []string{"crm"}, 0, 0, false},
		{"Синхронизация с 1С", "Обсудить API и формат обмена", "meeting", "high", "pending", now.AddDate(0, 0, 2), []string{"интеграция"}, 0, 0, false},
		{"Позвонить клиенту", "Уточнить сроки по договору", "call", "medium", "completed", now.AddDate(0, 0, -1), []string{}, 15, 30, true},
		{"Ревью PR #142", "Проверить изменения в модуле задач", "task", "medium", "in_progress", now.AddDate(0, 0, 1), []string{"code-review"}, 0, 0, false},
		{"Обед с командой", "Обсудить спринт", "lunch", "low", "pending", now.AddDate(0, 0, 5), []string{}, 0, 0, false},
		{"Отправить отчёт по спринту", "Еженедельный отчёт стейкхолдерам", "email", "medium", "pending", now.AddDate(0, 0, 2), []string{"отчётность"}, 0, 0, false},
		{"Рефакторинг AuthService", "Вынести логику в отдельные пакеты", "task", "low", "pending", now.AddDate(0, 0, 14), []string{"backend"}, 0, 0, false},
		{"Документация API", "Обновить Swagger для новых эндпоинтов задач", "task", "medium", "completed", now.AddDate(0, 0, -2), []string{"docs"}, 45, 0, true},
		{"Настроить мониторинг", "Prometheus + Grafana для API", "feature", "high", "pending", now.AddDate(0, 0, 10), []string{"devops", "мониторинг"}, 0, 0, false},
		{"Исправить утечку памяти", "В компоненте Kanban при большом списке", "bug", "high", "in_progress", now.AddDate(0, 0, 1), []string{"frontend", "performance"}, 0, 0, false},
		{"Демо для инвестора", "Подготовить презентацию и сценарий", "meeting", "critical", "pending", now.AddDate(0, 0, 5), []string{"демо"}, 0, 0, false},
		{"Миграция на Vue 3.5", "Обновить зависимости и проверить совместимость", "task", "low", "cancelled", now.AddDate(0, 0, 30), []string{"frontend"}, 0, 0, false},
		{"Ежедневный стендап", "Синхрон в 10:00", "call", "low", "pending", now.AddDate(0, 0, 0), []string{"agile"}, 0, 0, false},
	}

	taskIDs := make([]uuid.UUID, len(tasksData))
	for i, t := range tasksData {
		taskIDs[i] = uuid.New()
		dueStr := t.dueDate.Format(dueFmt)
		completedAt := interface{}(nil)
		completedBy := interface{}(nil)
		if t.completed {
			completedAt = now.AddDate(0, 0, -1).Format(dueFmt)
			completedBy = uID
		}
		tagsVal := t.tags
		if tagsVal == nil {
			tagsVal = []string{}
		}
		_, err = db.ExecContext(ctx, `
			INSERT INTO tasks (id, workspace_id, title, description, type, priority, status, due_date,
				assignee_id, created_by, created_at, updated_at, spent_minutes, spent_seconds, tags, completed_at, completed_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8::timestamp, $9, $10, $11, $11, $12, $13, $14, $15::timestamp, $16)`,
			taskIDs[i], wsID, t.title, t.description, t.type_, t.priority, t.status, dueStr,
			assignee, uID, now, t.spentMin, t.spentSec, pq.Array(tagsVal), completedAt, completedBy)
		if err != nil {
			log.Fatalf("Insert task %d: %v", i+1, err)
		}
	}

	// Подзадачи для первой задачи
	subTasks := []struct {
		title    string
		status   string
		completed bool
	}{
		{"Создать workflow в GitHub", "completed", true},
		{"Добавить тесты", "in_progress", false},
		{"Настроить деплой на staging", "pending", false},
	}
	for _, st := range subTasks {
		subID := uuid.New()
		dueStr := now.AddDate(0, 0, 5).Format(dueFmt)
		completedAt := interface{}(nil)
		completedBy := interface{}(nil)
		if st.completed {
			completedAt = now.Format(dueFmt)
			completedBy = uID
		}
		_, err = db.ExecContext(ctx, `
			INSERT INTO tasks (id, workspace_id, title, type, priority, status, due_date, parent_id,
				assignee_id, created_by, created_at, updated_at, tags, completed_at, completed_by)
			VALUES ($1, $2, $3, 'task', 'medium', $4, $5::timestamp, $6, $7, $8, $9, $9, '{}', $10::timestamp, $11)`,
			subID, wsID, st.title, st.status, dueStr, taskIDs[0], assignee, uID, now, completedAt, completedBy)
		if err != nil {
			log.Fatalf("Insert subtask: %v", err)
		}
	}

	// Связь блокировки: задача 2 блокируется задачей 1
	_, err = db.ExecContext(ctx, `
		INSERT INTO task_task_links (id, task_id, linked_task_id, link_type, created_at)
		VALUES (gen_random_uuid(), $1, $2, 'blocked_by', NOW())`,
		taskIDs[1], taskIDs[0])
	if err != nil {
		log.Printf("Warning: task_task_link: %v", err)
	}

	// Комментарии к первой задаче
	comments := []string{
		"Начал настройку. Нужно определиться с окружениями.",
		"Staging готов, ждём ревью.",
	}
	for _, body := range comments {
		_, err = db.ExecContext(ctx, `
			INSERT INTO task_comments (id, task_id, body, created_by, created_at)
			VALUES (gen_random_uuid(), $1, $2, $3, NOW())`,
			taskIDs[0], body, uID)
		if err != nil {
			log.Printf("Warning: task_comment: %v", err)
		}
	}

	log.Printf("Seed tasks: created %d tasks, %d subtasks, 1 link, %d comments",
		len(tasksData), len(subTasks), len(comments))
	log.Printf("Workspace: %s | Assignee: %s", wsID, assignee)
	fmt.Println("Done. Open Tasks module to see the data.")
}
