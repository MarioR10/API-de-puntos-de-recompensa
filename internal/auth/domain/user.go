package domain

import "time"

// Entidad de dominio User
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}
