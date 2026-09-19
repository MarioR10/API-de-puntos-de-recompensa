package service

import (
	"context"
)

type IAuthService interface {
	Register(ctx context.Context, request UserRegisterRequest) (UserRegisterResponse, error)
}
