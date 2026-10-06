package service

import (
	"context"
	"proyecto-Go/internal/auth/domain"
)

type IAuthService interface {
	Register(ctx context.Context, email, password string) (domain.User, error)
}
