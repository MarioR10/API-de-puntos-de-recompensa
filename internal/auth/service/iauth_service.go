package service

import (
	"context"

	"github.com/MarioR10/Proyecto-Go/internal/auth/domain"
)

type IAuthService interface {
	Register(ctx context.Context, email, password string) (domain.User, error)
}
