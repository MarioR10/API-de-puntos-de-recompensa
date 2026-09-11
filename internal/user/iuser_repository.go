package user

import "context"

type IUserRepository interface {
	CountUsers(ctx context.Context) (int64, error)
}
