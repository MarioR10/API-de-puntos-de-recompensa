package domain

import (
	"time"

	"github.com/google/uuid"
)

// Entidad de dominio User
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}
