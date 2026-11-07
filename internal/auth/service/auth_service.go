package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/MarioR10/Proyecto-Go/internal/auth/domain"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength = 8
	maxPasswordBytes  = 72
)

type AuthService struct {
	repository domain.UserRepository
	cost       int
}

func NewAuthService(repository domain.UserRepository, cost int) *AuthService {
	return &AuthService{repository: repository, cost: cost}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (domain.User, error) {

	//1. Normalizamos el email
	emailNormalized := normalizeEmail(email)

	//2.Validar contraseña antes de hashear

	//2.1 es muy corta
	if utf8.RuneCountInString(password) < minPasswordLength {
		return domain.User{}, fmt.Errorf("registrando al usuario: %w", domain.ErrPasswordTooShort)
	}
	//2.2 es muy larga
	if len(password) > maxPasswordBytes {
		return domain.User{}, fmt.Errorf("registrando al usuario: %w", domain.ErrPasswordTooLong)
	}

	//3. Proceso de hashing

	// 3.1 convertimos en bytes la contraseña
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)

	if errors.Is(err, bcrypt.ErrPasswordTooLong) {
		return domain.User{}, fmt.Errorf("registrando al usuario: %w", domain.ErrPasswordTooLong)
	}

	if err != nil {

		return domain.User{}, fmt.Errorf("registrando al usuario: %w", err)
	}

	//4. Generamos el id (UUID) del usuario
	userID, err := uuid.NewV7()
	if err != nil {
		return domain.User{}, fmt.Errorf("registrando al usuario: %w", err)
	}

	userDomain := domain.User{
		ID:           userID,
		Email:        emailNormalized,
		PasswordHash: string(passwordHash),
	}

	// 4. persistimos a la basse de datos
	userLoad, err := s.repository.CreateUser(ctx, userDomain)

	if err != nil {
		return domain.User{}, fmt.Errorf("registrando al usuario: %w", err)
	}
	return userLoad, nil

}

// funcion helper, normalizamos el email
func normalizeEmail(email string) string {
	//1. Normalizamos email (eliminamos espacio y pasamos todo a minuscula)

	//1.1 eliminamos espacios en blanco
	//1.2 pasamos todo a mayucula, no diferenciamos en mayusculas y minusculas.
	// Mario@Gmail.com y mario@gmail.com son lo mismo, mismo correo asociado a un usuario
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	return normalizedEmail
}
