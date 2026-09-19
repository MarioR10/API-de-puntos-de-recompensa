package service

import "time"

type UserRegisterRequest struct {
	Password string
	email    string
}

type UserRegisterResponse struct {
	ID        int64
	Email     string
	CreatedAt time.Time
}
