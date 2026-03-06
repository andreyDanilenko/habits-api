// seed_users_roles создаёт тестовых пользователей с разными ролями в workspace
// для тестирования мультитенаси и разных ролей без реализации приглашений.
//
// Использование:
//
//	go run ./cmd/seed_users_roles
//
// Переменные окружения:
//   SKIP_EXISTING=1 — не перезаписывать существующих пользователей/workspace
//   RESET=1         — удалить seed workspaces перед созданием
//   RESET_FULL=1    — удалить seed users + workspaces, затем создать заново (осторожно: удалит все workspace этих пользователей)
//
// Все пользователи имеют пароль: Password123!

// {userOwner, "owner@demo.local", "Владелец", "USER"},
// {userAdmin, "admin@demo.local", "Администратор", "USER"},
// {userMember, "member@demo.local", "Сотрудник", "USER"},
// {userGuest, "guest@demo.local", "Гость", "USER"},
// {userMulti, "multi@demo.local", "Мультитенант", "USER"},
//
// Источник ролей участника:
//   - user_workspaces (членство, role: OWNER/ADMIN/MEMBER/GUEST)
//   - user_role_assignments (права для Casbin, GetUserRoles, GetMyPermissions → systemRole)
//     Оба должны быть синхронизированы. Миграция 000024 исправляет расхождения.
package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"backend/internal/config"
	"backend/internal/database"
	"backend/pkg/password"

	"github.com/google/uuid"
)

const defaultPassword = "Password123!"

var (
	userOwner  = uuid.MustParse("a1111111-1111-1111-1111-111111111101")
	userAdmin  = uuid.MustParse("a1111111-1111-1111-1111-111111111102")
	userMember = uuid.MustParse("a1111111-1111-1111-1111-111111111103")
	userGuest  = uuid.MustParse("a1111111-1111-1111-1111-111111111104")
	userMulti  = uuid.MustParse("a1111111-1111-1111-1111-111111111105")

	wsDemo     = uuid.MustParse("b2222222-2222-2222-2222-222222222201")
	wsPersonal = uuid.MustParse("b2222222-2222-2222-2222-222222222202")
)

func main() {
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
	skipExisting := os.Getenv("SKIP_EXISTING") == "1"
	resetFirst := os.Getenv("RESET") == "1"
	resetFull := os.Getenv("RESET_FULL") == "1"

	if resetFull {
		if err := resetSeedDataFull(ctx, db); err != nil {
			log.Fatalf("Reset full: %v", err)
		}
		log.Printf("Seed users and workspaces reset done")
	} else if resetFirst {
		if err := resetSeedData(ctx, db); err != nil {
			log.Fatalf("Reset seed data: %v", err)
		}
		log.Printf("Seed workspaces reset done")
	}

	hashedPassword, err := password.Hash(defaultPassword)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	users := []struct {
		id    uuid.UUID
		email string
		name  string
		role  string
	}{
		{userOwner, "owner@demo.local", "Владелец", "USER"},
		{userAdmin, "admin@demo.local", "Администратор", "USER"},
		{userMember, "member@demo.local", "Сотрудник", "USER"},
		{userGuest, "guest@demo.local", "Гость", "USER"},
		{userMulti, "multi@demo.local", "Мультитенант", "USER"},
	}

	for _, u := range users {
		if err := ensureUser(ctx, db, u.id, u.email, u.name, u.role, hashedPassword, skipExisting); err != nil {
			log.Fatalf("User %s: %v", u.email, err)
		}
		log.Printf("User %s (%s) ready", u.email, u.name)
	}

	// Workspaces: "Демо" (owner=owner), "Личный" (owner=multi)
	if err := ensureWorkspace(ctx, db, wsDemo, "Демо", "Демо workspace для тестов ролей", userOwner, skipExisting); err != nil {
		log.Fatalf("Workspace Демо: %v", err)
	}
	if err := ensureWorkspace(ctx, db, wsPersonal, "Личный", "Личный workspace мультитенанта", userMulti, skipExisting); err != nil {
		log.Fatalf("Workspace Личный: %v", err)
	}
	log.Printf("Workspaces created")

	// user_workspaces + user_role_assignments (assignedBy = владелец workspace)
	assignments := []struct {
		userID     uuid.UUID
		wsID       uuid.UUID
		role       string
		assignedBy uuid.UUID
	}{
		{userOwner, wsDemo, "OWNER", userOwner},
		{userAdmin, wsDemo, "ADMIN", userOwner},
		{userMember, wsDemo, "MEMBER", userOwner},
		{userGuest, wsDemo, "GUEST", userOwner},
		{userMulti, wsDemo, "MEMBER", userOwner},
		{userMulti, wsPersonal, "OWNER", userMulti},
	}

	for _, a := range assignments {
		if err := ensureWorkspaceMember(ctx, db, a.userID, a.wsID, a.role, a.assignedBy, skipExisting); err != nil {
			log.Fatalf("Assign %s to %s as %s: %v", a.userID, a.wsID, a.role, err)
		}
	}
	log.Printf("Role assignments done")

	log.Printf(`
Seed completed. Users (password: %s):
  owner@demo.local   — OWNER в "Демо"
  admin@demo.local   — ADMIN в "Демо"
  member@demo.local — MEMBER в "Демо"
  guest@demo.local  — GUEST в "Демо"
  multi@demo.local  — MEMBER в "Демо" + OWNER в "Личный"

Для теста мультитенаси: войдите как multi@demo.local и переключайте workspace.
`, defaultPassword)
}

func resetSeedData(ctx context.Context, db *sql.DB) error {
	seedWsIDs := []uuid.UUID{wsDemo, wsPersonal}
	for _, wsID := range seedWsIDs {
		if _, err := db.ExecContext(ctx, `DELETE FROM user_role_assignments WHERE workspace_id = $1`, wsID); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM user_workspaces WHERE workspace_id = $1`, wsID); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM workspace_roles WHERE workspace_id = $1`, wsID); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = $1`, wsID); err != nil {
			return err
		}
	}
	return nil
}

func resetSeedDataFull(ctx context.Context, db *sql.DB) error {
	seedUserIDs := []uuid.UUID{userOwner, userAdmin, userMember, userGuest, userMulti}
	seedWsIDs := []uuid.UUID{wsDemo, wsPersonal}

	// 1. Workspaces, владельцами которых являются seed users
	for _, uid := range seedUserIDs {
		// Порядок: user_role_assignments → user_workspaces → workspace_roles → workspaces
		if _, err := db.ExecContext(ctx, `DELETE FROM user_role_assignments WHERE workspace_id IN (SELECT id FROM workspaces WHERE owner_id = $1)`, uid); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM user_workspaces WHERE workspace_id IN (SELECT id FROM workspaces WHERE owner_id = $1)`, uid); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM workspace_roles WHERE workspace_id IN (SELECT id FROM workspaces WHERE owner_id = $1)`, uid); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM workspaces WHERE owner_id = $1`, uid); err != nil {
			return err
		}
		// Назначения в других workspace
		if _, err := db.ExecContext(ctx, `DELETE FROM user_role_assignments WHERE user_id = $1`, uid); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM user_workspaces WHERE user_id = $1`, uid); err != nil {
			return err
		}
	}

	// 2. Seed workspaces (если остались)
	for _, wsID := range seedWsIDs {
		if _, err := db.ExecContext(ctx, `DELETE FROM user_role_assignments WHERE workspace_id = $1`, wsID); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM user_workspaces WHERE workspace_id = $1`, wsID); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM workspace_roles WHERE workspace_id = $1`, wsID); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = $1`, wsID); err != nil {
			return err
		}
	}

	// 3. Удалить seed users
	for _, uid := range seedUserIDs {
		if _, err := db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, uid); err != nil {
			return err
		}
	}
	return nil
}

func ensureUser(ctx context.Context, db *sql.DB, id uuid.UUID, email, name, role, hashedPassword string, skip bool) error {
	if skip {
		var n int
		err := db.QueryRowContext(ctx, `SELECT 1 FROM users WHERE id = $1`, id).Scan(&n)
		if err == nil {
			return nil
		}
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO users (id, email, password, name, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'ACTIVE', NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET password = EXCLUDED.password, name = EXCLUDED.name
	`, id, email, hashedPassword, name, role)
	return err
}

func ensureWorkspace(ctx context.Context, db *sql.DB, id uuid.UUID, name, desc string, ownerID uuid.UUID, skip bool) error {
	if skip {
		var n int
		err := db.QueryRowContext(ctx, `SELECT 1 FROM workspaces WHERE id = $1`, id).Scan(&n)
		if err == nil {
			return nil
		}
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, description, color, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, '#3B82F6', $4, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, id, name, desc, ownerID)
	if err != nil {
		return err
	}
	// Триггер создаёт системные роли. Проверяем, что они есть.
	var roleCount int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_roles WHERE workspace_id = $1`, id).Scan(&roleCount)
	if err != nil {
		return err
	}
	if roleCount == 0 {
		_, err = db.ExecContext(ctx, `
			INSERT INTO workspace_roles (id, workspace_id, name, is_system, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, 'OWNER', true, NOW(), NOW()),
			       (gen_random_uuid(), $1, 'ADMIN', true, NOW(), NOW()),
			       (gen_random_uuid(), $1, 'MEMBER', true, NOW(), NOW()),
			       (gen_random_uuid(), $1, 'GUEST', true, NOW(), NOW())
		`, id)
		return err
	}
	return nil
}

func ensureWorkspaceMember(ctx context.Context, db *sql.DB, userID, wsID uuid.UUID, roleName string, assignedBy uuid.UUID, _ bool) error {
	// 1. user_workspaces (членство)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO user_workspaces (id, user_id, workspace_id, role, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, NOW())
		ON CONFLICT (user_id, workspace_id) DO UPDATE SET role = EXCLUDED.role
	`, userID, wsID, roleName); err != nil {
		return err
	}

	// 2. user_role_assignments (права для Casbin)
	var roleID uuid.UUID
	if err := db.QueryRowContext(ctx, `SELECT id FROM workspace_roles WHERE workspace_id = $1 AND name = $2`, wsID, roleName).Scan(&roleID); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO user_role_assignments (id, user_id, role_id, workspace_id, assigned_by, assigned_at)
		SELECT gen_random_uuid(), $1, $2, $3, $4, NOW()
		WHERE NOT EXISTS (SELECT 1 FROM user_role_assignments WHERE user_id = $1 AND role_id = $2 AND workspace_id = $3)
	`, userID, roleID, wsID, assignedBy)
	return err
}
