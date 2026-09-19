package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"proyecto-Go/internal/auth/domain"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength = 8
	maxPasswordBytes  = 72
)

type authServiceImpl struct {
	logger     *slog.Logger
	repository domain.IUserRepository
	cost       int
}

func NewAuthService(logger *slog.Logger, repository domain.IUserRepository, cost int) IAuthService {
	return &authServiceImpl{logger: logger, repository: repository, cost: cost}
}

func (s *authServiceImpl) Register(ctx context.Context, request UserRegisterRequest) (UserRegisterResponse, error) {

	//1.Validar contraseña antes de hashear

	//1.1 es muy corta
	if utf8.RuneCountInString(request.Password) < minPasswordLength {
		return UserRegisterResponse{}, fmt.Errorf("registrando al usuario: %w", domain.ErrPasswordTooShort)
	}
	//1.2 es muy larga
	if len(request.Password) > maxPasswordBytes {
		return UserRegisterResponse{}, fmt.Errorf("registrando al usuario: %w", domain.ErrPasswordTooLong)
	}

	//2. Proceso de hashing

	// 2.1 convertimos en bytes la contraseña
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Password), s.cost)

	if errors.Is(err, bcrypt.ErrPasswordTooLong) {
		return UserRegisterResponse{}, fmt.Errorf("contraseña invalida: %w", domain.ErrPasswordTooLong)
	}

	if err != nil {

		return UserRegisterResponse{}, fmt.Errorf("hubo un problema al hashear la contraseña: %w", err)
	}

	// 2.2 Convertimos a entidad de dominio
	userDomain := domain.User{
		Email:        request.email,
		PasswordHash: string(passwordHash),
	}

	// 2.3 persistimos a la basse de datos
	userLoad, err := s.repository.CreateUser(ctx, userDomain)

	if err != nil {
		return UserRegisterResponse{}, fmt.Errorf("ocurrio un problema al persistir: %w", err)
	}

	//2.4 convertimos a DTO
	response := UserRegisterResponse{
		ID:        userLoad.ID,
		Email:     userLoad.Email,
		CreatedAt: userLoad.CreatedAt,
	}

	return response, nil

}
