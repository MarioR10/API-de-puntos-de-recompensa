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
	minPasswordLength     = 8
	maxPasswordBytes      = 72
	passwordReferenceHash = "vR8!kZ3@pL6#tN9$xQ2"
)

type AuthService struct {
	repository    domain.UserRepository
	cost          int
	referenceHash string
}

func NewAuthService(repository domain.UserRepository, cost int) (*AuthService, error) {

	//Generamos el hash de referencia
	referenceHash, err := bcrypt.GenerateFromPassword([]byte(passwordReferenceHash), cost)
	if err != nil {
		return nil, fmt.Errorf("generando hash de referencia: %w", err)

	}
	return &AuthService{repository: repository, cost: cost, referenceHash: string(referenceHash)}, nil
}

func (s *AuthService) Register(ctx context.Context, email, password string) (domain.User, error) {

	//1. Normalizamos el email
	emailNormalized := normalizeEmail(email)

	//2.Validar contraseña antes de hashear

	err := minAndMaxPasswordLength(password)
	if err != nil {
		//Devolvemos el error ya envuelto por la funcion
		return domain.User{}, err
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

func (s *AuthService) Login(ctx context.Context, email, password string) error {

	// 1. Normalizamos el email
	emailNormalized := normalizeEmail(email)

	// 2. Validamos que la contraseña no supere los 72 bytes.
	// CompareHashAndPassword solo considera los primeros 72 bytes. Bcrypt ignora los bytes adicionales al comparar, por lo que podría
	// aceptar la contraseña correcta aunque tenga texto extra al final.
	//Rechazamos las entradas que lo superen  para mantener el mismo límite que aplicamos en el registro.
	if len(password) > maxPasswordBytes {
		return fmt.Errorf("mientras se validaba la contraseña: %w", domain.ErrInvalidCredentials)
	}

	//3. Obtenemos el usuario llamando al repositorio
	user, err := s.repository.FindByEmail(ctx, emailNormalized)

	//Validamos errores:
	//1. por usuario no encontrado,
	if errors.Is(err, domain.ErrUserNotFound) {
		//Si entramos en este caso, ocupamos el hash de referencia para hacer la comparacion junto a la contraseña recibida y no terminar directamente
		//independientemente de la comparacion, devolvemos credenciales invalidas
		_= bcrypt.CompareHashAndPassword([]byte(s.referenceHash),[]byte(password))

		//siempre al salir en este caso retornamos error
		return fmt.Errorf("mientras se iniciaba sesion: %w", domain.ErrInvalidCredentials)
	}
	//2. cualquier otro error
	if err != nil {
		return fmt.Errorf("mientras se iniciaba sesion: %w", err)
	}

	//4.Una vez obtenido el usuario y validada la longitud de la contraseñá,  validamos la contraseña
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))

	//validemos el error obtenido, con este metodo podemos tener diversos errores, dividamos el resto del de credenciales invalidas
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return fmt.Errorf("mientras se iniciaba sesion: %w", domain.ErrInvalidCredentials)
	}
	if err != nil {
		return fmt.Errorf("mientras se iniciaba sesion: %w", err)
	}
	return nil
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

// Funcion que me permite validar la longitud minima y maxima aceptable de la contraseña
func minAndMaxPasswordLength(password string) error {

	//2.1 es muy corta
	if utf8.RuneCountInString(password) < minPasswordLength {
		return fmt.Errorf("mientras se validaba la contraseña: %w", domain.ErrPasswordTooShort)
	}
	//2.2 es muy larga
	if len(password) > maxPasswordBytes {
		return fmt.Errorf("mientras se validaba la contraseña: %w", domain.ErrPasswordTooLong)
	}

	return nil

}
