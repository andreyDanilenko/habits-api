package workspace

import (
	"context"

	"backend/internal/model"
)

// AccessChecker проверяет доступ пользователя к workspace.
// Позволяет подменить реализацию (например, кэшированную) без изменения middleware.
type AccessChecker interface {
	HasAccess(ctx context.Context, workspaceID, userID string, userRole model.UserRole) (bool, error)
}
