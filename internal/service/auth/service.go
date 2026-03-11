package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"backend/internal/model"
	regTokenRepo "backend/internal/repository/registration_token"
	userRepo "backend/internal/repository/user"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/auth/token"
	"backend/pkg/email"
	"backend/pkg/password"

	"github.com/google/uuid"
)

var (
	ErrUserExists            = errors.New("user already exists")
	ErrInvalidCredentials     = errors.New("invalid email or password")
	ErrUserNotFound           = errors.New("user not found")
	ErrInvalidRefreshToken    = errors.New("invalid or expired refresh token")
	ErrInvalidVerificationToken = errors.New("invalid or expired verification token")
)

type AuthService struct {
	userRepo           userRepo.UserRepository
	regTokenRepo       regTokenRepo.Repository
	workspaceService   *workspaceService.Service
	tokenGen           *token.Generator
	emailSender        email.Sender
	accessExpiry       time.Duration
	refreshExpiry      time.Duration
	regTokenLifetime   time.Duration
	verificationBaseURL string
}

type LoginResponse struct {
	User         *model.User `json:"user"`
	AccessToken  string      `json:"-"`
	RefreshToken string      `json:"-"`
	ExpiresIn    int         `json:"expires_in"`
}

// RegisterOutput — результат Register: либо ожидание подтверждения, либо сразу логин (реактивация).
type RegisterOutput struct {
	PendingVerification bool         `json:"-"` // true = отправлено письмо, пользователь не создан
	Message             string       `json:"message,omitempty"`
	LoginResponse       *LoginResponse `json:"-"` // при реактивации — сразу логин
}

func NewService(
	userRepo userRepo.UserRepository,
	regTokenRepo regTokenRepo.Repository,
	workspaceService *workspaceService.Service,
	tokenGen *token.Generator,
	emailSender email.Sender,
	accessExpiry time.Duration,
	refreshExpiry time.Duration,
	regTokenLifetime time.Duration,
	verificationBaseURL string,
) *AuthService {
	return &AuthService{
		userRepo:            userRepo,
		regTokenRepo:        regTokenRepo,
		workspaceService:    workspaceService,
		tokenGen:            tokenGen,
		emailSender:         emailSender,
		accessExpiry:        accessExpiry,
		refreshExpiry:       refreshExpiry,
		regTokenLifetime:    regTokenLifetime,
		verificationBaseURL: verificationBaseURL,
	}
}

func (s *AuthService) Register(ctx context.Context, req model.RegisterRequest) (*RegisterOutput, error) {
	// 1. Активный пользователь с таким email уже есть — нельзя регистрироваться
	existing, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUserExists
	}

	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	// 2. Удалённый пользователь (soft delete) — реактивируем без подтверждения email
	anyStatus, err := s.userRepo.FindByEmailAnyStatus(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if anyStatus != nil && anyStatus.Status != nil && *anyStatus.Status == model.UserStatusDeleted {
		now := time.Now()
		anyStatus.Status = userStatusPtr(model.UserStatusActive)
		anyStatus.Password = hashedPassword
		anyStatus.Name = &req.Name
		anyStatus.UpdatedAt = now
		if err := s.userRepo.Update(ctx, anyStatus); err != nil {
			return nil, err
		}
		anyStatus.Password = ""
		accessToken, err := s.tokenGen.Generate(anyStatus.ID, string(anyStatus.Role))
		if err != nil {
			return nil, err
		}
		refreshToken, err := s.tokenGen.GenerateRefreshToken(anyStatus.ID, string(anyStatus.Role), s.refreshExpiry)
		if err != nil {
			return nil, err
		}
		return &RegisterOutput{
			PendingVerification: false,
			LoginResponse: &LoginResponse{
				User:         anyStatus,
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				ExpiresIn:    int(s.accessExpiry.Seconds()),
			},
		}, nil
	}

	// 3. Новая регистрация: сохраняем токен, отправляем письмо, пользователь НЕ создаётся
	_ = s.regTokenRepo.DeleteByEmail(ctx, req.Email)

	tok, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(s.regTokenLifetime)
	rt := &model.RegistrationToken{
		ID:           uuid.New().String(),
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Name:         &req.Name,
		Token:        tok,
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
	}
	if err := s.regTokenRepo.Create(ctx, rt); err != nil {
		return nil, fmt.Errorf("save registration token: %w", err)
	}

	verificationLink := s.verificationBaseURL + "/auth/verify-email?token=" + tok
	expiresInHours := int(s.regTokenLifetime.Hours())
	msg := email.BuildVerificationEmail(req.Email, verificationLink, expiresInHours)
	go func() {
		_ = s.emailSender.Send(context.Background(), msg)
	}()

	return &RegisterOutput{
		PendingVerification: true,
		Message:             "На вашу почту отправлена ссылка для подтверждения регистрации",
	}, nil
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// VerifyEmail подтверждает регистрацию по токену из ссылки. Создаёт пользователя и возвращает логин.
func (s *AuthService) VerifyEmail(ctx context.Context, token string) (*LoginResponse, error) {
	rt, err := s.regTokenRepo.FindByToken(ctx, token)
	if err != nil || rt == nil {
		return nil, ErrInvalidVerificationToken
	}
	if time.Now().After(rt.ExpiresAt) {
		_ = s.regTokenRepo.DeleteByToken(ctx, token)
		return nil, ErrInvalidVerificationToken
	}

	// Проверяем, что пользователь с таким email ещё не создан
	existing, err := s.userRepo.FindByEmail(ctx, rt.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		_ = s.regTokenRepo.DeleteByToken(ctx, token)
		return nil, ErrUserExists
	}

	// Создаём пользователя
	now := time.Now()
	user := &model.User{
		ID:        uuid.New().String(),
		Email:     rt.Email,
		Password:  rt.PasswordHash,
		Name:      rt.Name,
		Role:      model.UserRoleUser,
		Status:    userStatusPtr(model.UserStatusActive),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Создаём базовый workspace
	defaultName := "My Workspace"
	if user.Name != nil {
		defaultName = *user.Name + "'s Workspace"
	}
	_, _ = s.workspaceService.Create(ctx, model.CreateWorkspaceDto{
		Name:  defaultName,
		Color: stringPtr("#3B82F6"),
	}, user.ID)

	_ = s.regTokenRepo.DeleteByToken(ctx, token)

	accessToken, err := s.tokenGen.Generate(user.ID, string(user.Role))
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.tokenGen.GenerateRefreshToken(user.ID, string(user.Role), s.refreshExpiry)
	if err != nil {
		return nil, err
	}
	user.Password = ""
	return &LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.accessExpiry.Seconds()),
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req model.LoginRequest) (*LoginResponse, error) {
	// 1. Находим пользователя по email
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}

	// hashToCheck: при несуществующем пользователе используем dummy hash,
	// чтобы время ответа не отличалось (защита от user enumeration по timing attack).
	hashToCheck := password.DummyHash()
	if user != nil {
		hashToCheck = user.Password
	}

	// 2. Проверяем пароль (constant-time при любом user)
	if !password.Check(req.Password, hashToCheck) {
		return nil, ErrInvalidCredentials
	}
	if user == nil {
		// При user=nil hashToCheck=dummyHash, Check всегда false — сюда не попадём,
		// но оставляем для явности и на случай изменения логики.
		return nil, ErrInvalidCredentials
	}

	// 3. Проверяем статус пользователя
	if user.Status != nil && *user.Status != model.UserStatusActive {
		return nil, errors.New("account is not active")
	}

	// 4. Генерируем access и refresh токены
	accessToken, err := s.tokenGen.Generate(user.ID, string(user.Role))
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.tokenGen.GenerateRefreshToken(user.ID, string(user.Role), s.refreshExpiry)
	if err != nil {
		return nil, err
	}

	// 5. Создаем ответ
	user.Password = ""
	return &LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.accessExpiry.Seconds()),
	}, nil
}

func (s *AuthService) GetUserProfile(ctx context.Context, userID string) (*model.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	user.Password = ""
	return user, nil
}

// Refresh обновляет access token по refresh token. Возвращает новую пару токенов.
func (s *AuthService) Refresh(ctx context.Context, refreshTokenString string) (*LoginResponse, error) {
	claims, err := s.tokenGen.Validate(refreshTokenString)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil || user == nil {
		return nil, ErrInvalidRefreshToken
	}
	if user.Status != nil && *user.Status != model.UserStatusActive {
		return nil, ErrInvalidRefreshToken
	}

	accessToken, err := s.tokenGen.Generate(user.ID, string(user.Role))
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.tokenGen.GenerateRefreshToken(user.ID, string(user.Role), s.refreshExpiry)
	if err != nil {
		return nil, err
	}

	user.Password = ""
	return &LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.accessExpiry.Seconds()),
	}, nil
}

// Logout - выход (удаление cookies на стороне handler)
func (s *AuthService) Logout(ctx context.Context, userID string) error {
	return nil
}

func stringPtr(s string) *string {
	return &s
}

func userStatusPtr(s model.UserStatus) *model.UserStatus {
	return &s
}
