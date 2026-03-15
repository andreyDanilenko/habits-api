package notification

import (
	"context"

	"backend/internal/model"
	notificationRepo "backend/internal/repository/notification"
)

type Service struct {
	repo *notificationRepo.Repository
}

func NewService(repo *notificationRepo.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Upsert(ctx context.Context, userID string, dto *model.CreateNotificationDto) (*model.Notification, error) {
	if dto.Channel == "" {
		dto.Channel = "activity"
	}
	return s.repo.Upsert(ctx, userID, dto)
}

func (s *Service) List(ctx context.Context, userID string, opts model.NotificationListOpts) ([]model.Notification, int, error) {
	return s.repo.List(ctx, userID, opts)
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) error {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID, channel string) error {
	return s.repo.MarkAllRead(ctx, userID, channel)
}
