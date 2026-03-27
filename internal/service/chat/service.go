package chat

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"backend/internal/model"
	chatRepo "backend/internal/repository/chat"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/realtime"
)

var (
	ErrForbidden = errors.New("forbidden")
	ErrNotFound  = errors.New("not found")
)

type Service struct {
	repo      *chatRepo.Repository
	wsSvc     *workspaceService.Service
	publisher realtime.Publisher
}

func NewService(repo *chatRepo.Repository, wsSvc *workspaceService.Service, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.NoopPublisher{}
	}
	return &Service{repo: repo, wsSvc: wsSvc, publisher: publisher}
}

func (s *Service) ListThreads(ctx context.Context, workspaceID, actorUserID string) ([]model.ChatThread, error) {
	return s.repo.ListThreads(ctx, workspaceID, actorUserID, 50)
}

func (s *Service) GetOrCreatePrivateThread(ctx context.Context, workspaceID, actorUserID, otherUserID string) (*model.ChatThread, bool, error) {
	otherUserID = strings.TrimSpace(otherUserID)
	if otherUserID == "" || otherUserID == actorUserID {
		return nil, false, ErrNotFound
	}
	t, created, err := s.repo.GetOrCreatePrivateThread(ctx, workspaceID, actorUserID, otherUserID, actorUserID)
	if err != nil {
		return nil, false, err
	}
	if created {
		_ = s.publisher.Publish(ctx, realtime.WorkspaceChannel(workspaceID), realtime.Event{
			EventType: realtime.EventChatThreadUpserted,
			Target:    realtime.Target{Type: "workspace", ID: workspaceID},
			Payload: map[string]interface{}{
				"thread": t,
			},
		})
	}
	return t, created, nil
}

func (s *Service) ListMessages(ctx context.Context, workspaceID, threadID, actorUserID string) ([]model.ChatMessage, error) {
	list, err := s.repo.ListMessages(ctx, workspaceID, threadID, actorUserID, 50, nil)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return list, err
}

func (s *Service) SendMessage(ctx context.Context, workspaceID, threadID, actorUserID, body string) (*model.ChatMessage, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, ErrNotFound
	}
	m, err := s.repo.CreateMessage(ctx, workspaceID, threadID, actorUserID, body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, realtime.WorkspaceChannel(workspaceID), realtime.Event{
		EventType: realtime.EventChatMessageCreated,
		Target:    realtime.Target{Type: "workspace", ID: workspaceID},
		Payload: map[string]interface{}{
			"message": m,
		},
	})
	return m, nil
}

