package invitation

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/config"
	"backend/internal/model"
	invRepo "backend/internal/repository/invitation"
	userRepo "backend/internal/repository/user"
	workspaceRepo "backend/internal/repository/workspace"
	permService "backend/internal/service/permission"
	"backend/pkg/email"
	"backend/pkg/realtime"

	"github.com/google/uuid"
)

var (
	ErrUserAlreadyInWorkspace = errors.New("user already in workspace")
	ErrInvitationNotFound     = errors.New("invitation not found")
	ErrInvitationExpired      = errors.New("invitation expired or already used")
	ErrInvitationWrongEmail   = errors.New("invitation is for different email")
)

type Service struct {
	invRepo       *invRepo.Repository
	workspaceRepo *workspaceRepo.Repository
	userRepo      userRepo.UserRepository
	permSvc       *permService.Service
	emailSender   email.Sender
	cfg           *config.Config
	publisher     realtime.Publisher
}

func NewService(
	invRepo *invRepo.Repository,
	workspaceRepo *workspaceRepo.Repository,
	userRepo userRepo.UserRepository,
	permSvc *permService.Service,
	emailSender email.Sender,
	cfg *config.Config,
	publisher realtime.Publisher,
) *Service {
	if publisher == nil {
		publisher = realtime.NoopPublisher{}
	}
	return &Service{
		invRepo:       invRepo,
		workspaceRepo: workspaceRepo,
		userRepo:      userRepo,
		permSvc:       permSvc,
		emailSender:   emailSender,
		cfg:           cfg,
		publisher:     publisher,
	}
}

func (s *Service) emitInvitationAccepted(ctx context.Context, workspaceID, userID string, inv *model.Invitation) {
	ch := realtime.WorkspaceChannel(workspaceID)
	event := realtime.Event{
		EventType: realtime.EventInvitationAccepted,
		Target:    realtime.Target{Type: "workspace", ID: workspaceID},
		Payload: map[string]interface{}{
			"workspaceId": workspaceID,
			"userId":      userID,
			"email":       inv.Email,
			"role":        inv.SystemRole,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	_ = s.publisher.Publish(ctx, ch, event)
	// Также отправить приглашённому пользователю
	userCh := realtime.UserChannel(userID)
	userEvent := realtime.Event{
		EventType: realtime.EventInvitationAccepted,
		Target:    realtime.Target{Type: "user", ID: userID},
		Payload: map[string]interface{}{
			"workspaceId": workspaceID,
			"userId":      userID,
			"email":       inv.Email,
			"role":        inv.SystemRole,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	_ = s.publisher.Publish(ctx, userCh, userEvent)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:43], nil
}

func (s *Service) Create(ctx context.Context, workspaceID, userID string, req model.CreateInvitationRequest) (*model.InvitationResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	_, err = uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	inviteEmail := strings.TrimSpace(strings.ToLower(req.Email))
	if inviteEmail == "" {
		return nil, errors.New("email is required")
	}
	systemRole := strings.ToUpper(strings.TrimSpace(req.SystemRole))
	if systemRole != "MEMBER" && systemRole != "GUEST" {
		return nil, errors.New("systemRole must be MEMBER or GUEST")
	}

	// Проверка: пользователь уже в workspace
	alreadyMember, err := s.workspaceRepo.IsMemberInWorkspace(ctx, wsID, inviteEmail)
	if err != nil {
		return nil, err
	}
	if alreadyMember {
		return nil, ErrUserAlreadyInWorkspace
	}

	expiresIn := 7 * 24 * time.Hour // 7 дней по умолчанию
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		expiresIn = time.Duration(*req.ExpiresIn) * time.Second
	}
	expiresAt := time.Now().Add(expiresIn)

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	inv := &model.Invitation{
		WorkspaceID: workspaceID,
		Email:       inviteEmail,
		InvitedBy:   userID,
		SystemRole:  systemRole,
		Status:      model.InvitationStatusPending,
		Token:       token,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}
	if err := s.invRepo.Create(ctx, inv); err != nil {
		return nil, err
	}

	ws, err := s.workspaceRepo.Get(ctx, wsID)
	if err != nil || ws == nil {
		return nil, fmt.Errorf("workspace not found")
	}

	invitedByUser, _ := s.userRepo.FindByID(ctx, userID)
	invitedByName := ""
	invitedByEmail := ""
	if invitedByUser != nil {
		if invitedByUser.Name != nil {
			invitedByName = *invitedByUser.Name
		}
		invitedByEmail = invitedByUser.Email
	}
	if invitedByName == "" {
		invitedByName = invitedByEmail
	}

	baseURL := strings.TrimSuffix(s.cfg.Auth.VerificationBaseURL, "/")
	inviteLink := fmt.Sprintf("%s/invite/%s", baseURL, token)

	// Отправляем email
	msg := email.BuildInviteEmail(inv.Email, inviteLink, ws.Name, invitedByName, inv.SystemRole, inv.ExpiresAt)
	if err = s.emailSender.Send(ctx, msg); err != nil {
		// Логируем, но не прерываем — приглашение создано
		_ = err
	}

	return &model.InvitationResponse{
		ID:          inv.ID,
		Email:       inv.Email,
		WorkspaceID: inv.WorkspaceID,
		InvitedBy: &model.InvitedByUser{
			ID:    userID,
			Name:  invitedByName,
			Email: invitedByEmail,
		},
		SystemRole: inv.SystemRole,
		Status:     inv.Status,
		InviteLink: inviteLink,
		ExpiresAt:  inv.ExpiresAt.Format(time.RFC3339),
		CreatedAt:  inv.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Service) List(ctx context.Context, workspaceID, userID string, status *string, limit, offset int) ([]model.InvitationResponse, int, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, 0, err
	}
	uid, _ := uuid.Parse(userID)
	hasAccess, err := s.workspaceRepo.HasAccess(ctx, wsID, uid)
	if err != nil || !hasAccess {
		return nil, 0, errors.New("access denied")
	}

	list, total, err := s.invRepo.ListByWorkspace(ctx, workspaceID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]model.InvitationResponse, 0, len(list))
	for _, inv := range list {
		invitedByUser, _ := s.userRepo.FindByID(ctx, inv.InvitedBy)
		invitedByName := ""
		invitedByEmail := ""
		if invitedByUser != nil {
			if invitedByUser.Name != nil {
				invitedByName = *invitedByUser.Name
			}
			invitedByEmail = invitedByUser.Email
		}
		if invitedByName == "" {
			invitedByName = invitedByEmail
		}
		responses = append(responses, model.InvitationResponse{
			ID:         inv.ID,
			Email:      inv.Email,
			WorkspaceID: inv.WorkspaceID,
			InvitedBy: &model.InvitedByUser{
				ID:    inv.InvitedBy,
				Name:  invitedByName,
				Email: invitedByEmail,
			},
			SystemRole: inv.SystemRole,
			Status:     inv.Status,
			ExpiresAt:  inv.ExpiresAt.Format(time.RFC3339),
			CreatedAt:  inv.CreatedAt.Format(time.RFC3339),
		})
	}
	return responses, total, nil
}

func (s *Service) Cancel(ctx context.Context, workspaceID, invitationID, userID string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	hasAccess, err := s.workspaceRepo.HasAccess(ctx, wsID, uid)
	if err != nil || !hasAccess {
		return errors.New("access denied")
	}

	inv, err := s.invRepo.GetByID(ctx, invitationID)
	if err != nil || inv == nil {
		return ErrInvitationNotFound
	}
	if inv.WorkspaceID != workspaceID {
		return ErrInvitationNotFound
	}
	if inv.Status != model.InvitationStatusPending {
		return errors.New("cannot cancel invitation that is not pending")
	}

	return s.invRepo.Cancel(ctx, invitationID)
}

func (s *Service) GetByToken(ctx context.Context, token string, currentUser *model.User) (*model.PublicInvitationResponse, error) {
	inv, err := s.invRepo.GetByToken(ctx, token)
	if err != nil || inv == nil {
		return nil, ErrInvitationNotFound
	}
	if inv.Status != model.InvitationStatusPending {
		return nil, ErrInvitationExpired
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, ErrInvitationExpired
	}

	wsID, _ := uuid.Parse(inv.WorkspaceID)
	ws, err := s.workspaceRepo.Get(ctx, wsID)
	if err != nil || ws == nil {
		return nil, ErrInvitationNotFound
	}

	invitedByUser, _ := s.userRepo.FindByID(ctx, inv.InvitedBy)
	invitedByName := ""
	if invitedByUser != nil {
		if invitedByUser.Name != nil {
			invitedByName = *invitedByUser.Name
		}
		if invitedByName == "" {
			invitedByName = invitedByUser.Email
		}
	}

	existingUser, _ := s.userRepo.FindByEmail(ctx, inv.Email)
	userExists := existingUser != nil

	isAuthenticated := false
	if currentUser != nil && strings.EqualFold(currentUser.Email, inv.Email) {
		isAuthenticated = true
	}

	return &model.PublicInvitationResponse{
		Email:           inv.Email,
		WorkspaceName:   ws.Name,
		InvitedByName:   invitedByName,
		SystemRole:      inv.SystemRole,
		ExpiresAt:       inv.ExpiresAt.Format(time.RFC3339),
		UserExists:      userExists,
		IsAuthenticated: isAuthenticated,
	}, nil
}

func (s *Service) Accept(ctx context.Context, token string, currentUser *model.User) (*model.AcceptInvitationResponse, error) {
	inv, err := s.invRepo.GetByToken(ctx, token)
	if err != nil || inv == nil {
		return nil, ErrInvitationNotFound
	}
	if inv.Status != model.InvitationStatusPending {
		return &model.AcceptInvitationResponse{Status: "expired", Message: "Invitation has expired or already used"}, nil
	}
	if time.Now().After(inv.ExpiresAt) {
		return &model.AcceptInvitationResponse{Status: "expired", Message: "Invitation has expired"}, nil
	}

	baseURL := strings.TrimSuffix(s.cfg.Auth.VerificationBaseURL, "/")
	inviteLink := fmt.Sprintf("%s/invite/%s", baseURL, token)

	// Пользователь аутентифицирован
	if currentUser != nil {
		if !strings.EqualFold(currentUser.Email, inv.Email) {
			return &model.AcceptInvitationResponse{
				Status:     "wrong_email",
				Message:    fmt.Sprintf("Invitation is for %s. Please log in with that email.", inv.Email),
				RedirectTo: inviteLink,
			}, nil
		}
		// Добавляем в workspace
		if err := s.addUserToWorkspace(ctx, inv, currentUser.ID); err != nil {
			return nil, err
		}
		if err := s.markAcceptedAndEmit(ctx, inv, currentUser.ID); err != nil {
			return nil, err
		}
		baseURL := strings.TrimSuffix(s.cfg.Auth.VerificationBaseURL, "/")
		return &model.AcceptInvitationResponse{
			Status:     "accepted",
			RedirectTo: baseURL + "/",
		}, nil
	}

	// Требуется аутентификация
	existingUser, _ := s.userRepo.FindByEmail(ctx, inv.Email)
	userExists := existingUser != nil

	if userExists {
		return &model.AcceptInvitationResponse{
			Status:     "requires_auth",
			Email:      inv.Email,
			UserExists: true,
			RedirectTo: fmt.Sprintf("/login?email=%s&inviteToken=%s", inv.Email, token),
		}, nil
	}

	return &model.AcceptInvitationResponse{
		Status:     "requires_auth",
		Email:      inv.Email,
		UserExists: false,
		RedirectTo: fmt.Sprintf("/register?email=%s&inviteToken=%s", inv.Email, token),
	}, nil
}

func (s *Service) addUserToWorkspace(ctx context.Context, inv *model.Invitation, userID string) error {
	wsID, _ := uuid.Parse(inv.WorkspaceID)
	uid, _ := uuid.Parse(userID)
	if err := s.workspaceRepo.AddMember(ctx, wsID, uid, inv.SystemRole); err != nil {
		return err
	}
	return s.permSvc.AssignRoleByName(ctx, userID, inv.WorkspaceID, inv.SystemRole, inv.InvitedBy)
}

func (s *Service) markAcceptedAndEmit(ctx context.Context, inv *model.Invitation, userID string) error {
	if err := s.invRepo.MarkAccepted(ctx, inv.ID); err != nil {
		return err
	}
	s.emitInvitationAccepted(ctx, inv.WorkspaceID, userID, inv)
	return nil
}

func (s *Service) AcceptAfterRegistration(ctx context.Context, inviteToken, userID string) error {
	if inviteToken == "" {
		return nil
	}
	inv, err := s.invRepo.GetByToken(ctx, inviteToken)
	if err != nil || inv == nil {
		return nil
	}
	if inv.Status != model.InvitationStatusPending {
		return nil
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return nil
	}
	if !strings.EqualFold(user.Email, inv.Email) {
		return nil
	}
	if err := s.addUserToWorkspace(ctx, inv, userID); err != nil {
		return err
	}
	return s.markAcceptedAndEmit(ctx, inv, userID)
}
