package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ketn1337/reshka-backend/internal/auth"
	"github.com/ketn1337/reshka-backend/internal/domain"
	"github.com/ketn1337/reshka-backend/internal/repo"
)

type AuthService struct {
	users     *repo.UserRepo
	jwtSecret string
	jwtTTL    time.Duration
}

func NewAuthService(users *repo.UserRepo, secret string, ttl time.Duration) *AuthService {
	return &AuthService{users: users, jwtSecret: secret, jwtTTL: ttl}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	u, hash, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", nil, domain.ErrUnauthorized
		}
		return "", nil, err
	}
	if !u.IsActive {
		return "", nil, domain.ErrForbidden
	}
	if err := auth.CheckPassword(hash, password); err != nil {
		return "", nil, domain.ErrUnauthorized
	}
	tok, err := auth.Issue(s.jwtSecret, s.jwtTTL, u.ID, u.Role, u.Email)
	if err != nil {
		return "", nil, fmt.Errorf("issue jwt: %w", err)
	}
	return tok, u, nil
}

// Me возвращает текущего пользователя по ID из контекста (для middleware).
func (s *AuthService) Me(ctx context.Context, userID int64) (*domain.User, error) {
	return s.users.GetByID(ctx, userID)
}
