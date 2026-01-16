package auth

import (
	"context"
	"errors"
	"time"

	"backend/internal/model"
	userRepo "backend/internal/repository/user"
	"backend/pkg/auth/token"
	"backend/pkg/password"

	"github.com/google/uuid"
)

var (
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
)

type AuthService struct {
	userRepo     userRepo.UserRepository
	tokenGen     *token.Generator
	accessExpiry time.Duration
}

type LoginResponse struct {
	User        *model.User `json:"user"`
	AccessToken string      `json:"-"`
	ExpiresIn   int         `json:"expires_in"`
}

func NewService(
	userRepo userRepo.UserRepository,
	tokenGen *token.Generator,
	accessExpiry time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		tokenGen:     tokenGen,
		accessExpiry: accessExpiry,
	}
}

// Register - регистрация нового пользователя
func (s *AuthService) Register(ctx context.Context, req model.RegisterRequest) (*model.User, error) {
	// 1. Проверяем, существует ли пользователь с таким email
	existing, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUserExists
	}

	// 2. Хешируем пароль
	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	// 3. Создаем пользователя
	now := time.Now()
	user := &model.User{
		ID:        uuid.New().String(),
		Email:     req.Email,
		Password:  hashedPassword,
		Name:      &req.Name,
		Role:      model.UserRoleUser,
		Status:    userStatusPtr(model.UserStatusActive),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 4. Сохраняем в БД
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// 5. Очищаем пароль перед возвратом
	user.Password = ""
	return user, nil
}

// Login - аутентификация пользователя
func (s *AuthService) Login(ctx context.Context, req model.LoginRequest) (*LoginResponse, error) {
	// 1. Находим пользователя по email
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// 2. Проверяем статус пользователя
	if user.Status != nil && *user.Status != model.UserStatusActive {
		return nil, errors.New("account is not active")
	}

	// 3. Проверяем пароль
	if !password.Check(req.Password, user.Password) {
		return nil, ErrInvalidCredentials
	}

	// 4. Генерируем access token
	accessToken, err := s.tokenGen.Generate(user.ID, string(user.Role))
	if err != nil {
		return nil, err
	}

	// 5. Создаем ответ
	user.Password = "" // Очищаем пароль
	return &LoginResponse{
		User:        user,
		AccessToken: accessToken,
		ExpiresIn:   int(s.accessExpiry.Seconds()),
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

// Logout - выход (здесь можно добавить инвалидацию токена при необходимости)
func (s *AuthService) Logout(ctx context.Context, userID string) error {

	return nil
}

func stringPtr(s string) *string {
	return &s
}

func userStatusPtr(s model.UserStatus) *model.UserStatus {
	return &s
}
