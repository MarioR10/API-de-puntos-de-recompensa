package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MarioR10/Proyecto-Go/internal/auth/domain"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// MOCK FALSO DEL REPOSITORIO
type MockUserRepository struct {
	err          error       //permite simular un fallo del repositorio.
	saved        domain.User //permite inspeccionar qué usuario recibió el repositorio.
	createCalled bool        //permite verificar si el servicio llegó a invocar CreateUser.
	findCalled   bool
	findEmail    string
	foundUser    domain.User
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {

	m.createCalled = true
	m.saved = user
	if m.err != nil {
		return domain.User{}, m.err
	}
	return user, nil
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {

	m.findCalled = true
	m.findEmail = email

	if m.err != nil {
		return domain.User{}, m.err
	}
	return m.foundUser, nil

}

func TestNewAuthService(t *testing.T) {

	invalidCost := bcrypt.MaxCost + 1
	test := []struct {
		name    string
		cost    int
		wantErr error
	}{
		{
			//Con este test comprobamos que: 1. El constructor termina sin error y devuelve un servicio
			// que el hash corresponde al texto de referencia
			//el costo del hash coincide con el que le pasamos
			name:    "Valid AuthService",
			cost:    bcrypt.MinCost,
			wantErr: nil,
		},
		{
			name:    "Invalid AuthService",
			cost:    invalidCost,
			wantErr: bcrypt.InvalidCostError(invalidCost),
		},
	}
	for _, tt := range test {

		t.Run(tt.name, func(t *testing.T) {

			//PREPARACIONES
			//1. El repository
			repoMock := MockUserRepository{}

			//2. Creamos el servicio
			service, err := NewAuthService(&repoMock, tt.cost)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error recibido = %v, se esperaba %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {

				if service != nil {
					t.Fatalf("se esperaba que el servicio fuera nil")
				}
				return
			}

			if service == nil {
				t.Fatalf("se esperaba que el servicio fuera diferente de nil")
			}

			//Comparamos si el hash que genero el constructor es correcto
			hashGenerado := []byte(service.referenceHash)
			passwordReference := []byte(passwordReferenceHash)

			err = bcrypt.CompareHashAndPassword(hashGenerado, passwordReference)
			if err != nil {
				t.Fatalf("error al comparar los hashes %v", err)
			}

			//Comprobacion del costo ocupado para generar el hash
			cost, err := bcrypt.Cost(hashGenerado)

			if err != nil {
				t.Fatalf("error al obtener el cost del hash de referencia: %v", err)
			}

			if cost != tt.cost {
				t.Fatalf("los hash de referencia se genero con costos diferentes, se uso: %d, y se esperaba: %d", cost, tt.cost)
			}

		})

	}
}
func TestAuthService_Register(t *testing.T) {

	var errDummy = errors.New("la base de datos no responde")
	test := []struct {
		name       string
		email      string
		password   string
		repoErr    error
		wantErr    error
		wantCalled bool
		wantEmail  string
	}{
		{
			name:       "Register a valid user",
			email:      "  Mario@Test.COM ",
			password:   "clave-segura-123",
			repoErr:    nil,
			wantErr:    nil,
			wantCalled: true,
			wantEmail:  "mario@test.com",
		},
		{
			name:       "Invalid password, too short",
			email:      "  Mario@Test.COM ",
			password:   "1234567",
			repoErr:    nil,
			wantErr:    domain.ErrPasswordTooShort,
			wantCalled: false,
			wantEmail:  "",
		},
		{
			name:       "Valid password: has exactly 8 characters",
			email:      "  Mario@Test.COM ",
			password:   "12345678",
			repoErr:    nil,
			wantErr:    nil,
			wantCalled: true,
			wantEmail:  "mario@test.com",
		},
		{
			name:       "Invalid password: rejected a password with 7 characters, although it occupies 14 bytes",
			email:      "  Mario@Test.COM ",
			password:   "ñññññññ",
			repoErr:    nil,
			wantErr:    domain.ErrPasswordTooShort,
			wantCalled: false,
			wantEmail:  "",
		},
		{
			name:       "Valid password: has exactly 72 bytes",
			email:      "  Mario@Test.COM ",
			password:   strings.Repeat("a", 72),
			repoErr:    nil,
			wantErr:    nil,
			wantCalled: true,
			wantEmail:  "mario@test.com",
		},
		{
			name:       "Invalid password: reject a password with 73 bytes",
			email:      "  Mario@Test.COM ",
			password:   strings.Repeat("a", 73),
			repoErr:    nil,
			wantErr:    domain.ErrPasswordTooLong,
			wantCalled: false,
			wantEmail:  "",
		},
		{
			name:       "Persistence error: duplicate email",
			email:      "  Mario@Test.COM ",
			password:   "clave-segura-123",
			repoErr:    domain.ErrEmailAlreadyExists,
			wantErr:    domain.ErrEmailAlreadyExists,
			wantCalled: true,
			wantEmail:  "mario@test.com",
		},
		{
			name:       "Error in database",
			email:      "  Mario@Test.COM ",
			password:   "clave-segura-123",
			repoErr:    errDummy,
			wantErr:    errDummy,
			wantCalled: true,
			wantEmail:  "mario@test.com",
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockUserRepository{err: tt.repoErr}
			svc, err := NewAuthService(repo, bcrypt.MinCost)

			if err != nil {
				t.Fatalf("Error al cargar el servicio: %v", err)
			}

			user, err := svc.Register(context.Background(), tt.email, tt.password)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, se esperaba %v", err, tt.wantErr)
			}

			if repo.createCalled != tt.wantCalled {
				t.Fatalf("repositorio llamado = %v, se esperaba %v", repo.createCalled, tt.wantCalled)
			}
			if !tt.wantCalled {
				return
			}
			if repo.saved.Email != tt.wantEmail {
				t.Errorf("email guardado = %q, se esperaba %q", repo.saved.Email, tt.wantEmail)
			}

			if tt.wantErr != nil {
				return
			}

			if repo.saved.ID.Version() != 7 {
				t.Errorf("version del ID =%d, se esperaba 7", repo.saved.ID.Version())
			}

			if repo.saved.PasswordHash == tt.password {
				t.Error("se guardó la contraseña en texto plano")
			}
			if err := bcrypt.CompareHashAndPassword([]byte(repo.saved.PasswordHash), []byte(tt.password)); err != nil {
				t.Errorf("el hash no corresponde a la contraseña: %v", err)
			}
			if user.Email != tt.wantEmail {
				t.Errorf("email devuelto = %q, se esperaba %q", user.Email, tt.wantEmail)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	var errDummy = errors.New("la base de datos no responde")
	test := []struct {
		name              string
		email             string
		wantEmail         string
		password          string
		passwordValid     string
		repoErr           error
		wantErr           error
		wantCalled        bool
		storedInvalidHash string
	}{
		{
			name:          "Valid login for User",
			email:         `Mariojose@Gmail.com`,
			wantEmail:     "mariojose@gmail.com",
			password:      `clave-segura-123`,
			passwordValid: `clave-segura-123`,
			repoErr:       nil,
			wantErr:       nil,
			wantCalled:    true,
		},
		{
			name:          "User not found",
			email:         "Mariojose@Gmail.com",
			wantEmail:     "mariojose@gmail.com",
			password:      "clave-segura-123",
			passwordValid: `clave-segura-123`,
			repoErr:       domain.ErrUserNotFound,
			wantErr:       domain.ErrInvalidCredentials,
			wantCalled:    true,
		},
		{
			name:          "Invalid password",
			email:         "Mariojose@Gmail.com",
			wantEmail:     "mariojose@gmail.com",
			password:      "clave-segura-123456",
			passwordValid: `clave-segura-123`,
			repoErr:       nil,
			wantErr:       domain.ErrInvalidCredentials,
			wantCalled:    true,
		},
		{
			name:          "Repository failure",
			email:         "Mariojose@Gmail.com",
			wantEmail:     "mariojose@gmail.com",
			password:      "clave-segura-123",
			passwordValid: `clave-segura-123`,
			repoErr:       errDummy,
			wantErr:       errDummy,
			wantCalled:    true,
		},
		{
			name:          "Exactly 72 bytes",
			email:         "MARIOJOSE@Gmail.com",
			wantEmail:     "mariojose@gmail.com",
			password:      strings.Repeat("a", 72),
			passwordValid: strings.Repeat("a", 72),
			repoErr:       nil,
			wantErr:       nil,
			wantCalled:    true,
		},
		{
			name:          "More than 72 bytes",
			email:         "MARIOJOSE@Gmail.com",
			password:      strings.Repeat("a", 73),
			passwordValid: strings.Repeat("a", 72),
			repoErr:       nil,
			wantErr:       domain.ErrInvalidCredentials,
			wantCalled:    false,
		},
		{
			name:              "Hash malformed",
			email:             "Mariojose@Gmail.com",
			wantEmail:         "mariojose@gmail.com",
			password:          "clave-segura-123",
			passwordValid:     "short",
			repoErr:           nil,
			wantErr:           bcrypt.ErrHashTooShort,
			wantCalled:        true,
			storedInvalidHash: "hashInvalido",
		},
		{
			name:          "Recognizes ErrUserNotFound even when wrapped.",
			email:         "Mariojose@Gmail.com",
			wantEmail:     "mariojose@gmail.com",
			password:      "clave-segura-123",
			passwordValid: `clave-segura-123`,
			repoErr:       fmt.Errorf("Error de dominio envuelto %w", domain.ErrUserNotFound),
			wantErr:       domain.ErrInvalidCredentials,
			wantCalled:    true,
		},
		{
			name:      "Using the reference hash",
			email:     "  Mariojose@Gmail.com",
			wantEmail: "mariojose@gmail.com",
			password: passwordReferenceHash,
			repoErr: domain.ErrUserNotFound,
			wantErr: domain.ErrInvalidCredentials,
			wantCalled: true,
		},
	}

	for _, tt := range test {

		t.Run(tt.name, func(t *testing.T) {

			//1. PREPARACIONES INICIALES

			//1.1 preparamos un hash valido con bcrypt para una contraseña conocida (la que ocuparemos en el caso feliz)

			var hash string
			if tt.storedInvalidHash == "" {
				passwordBytes := []byte(tt.passwordValid)
				hashBytes, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.MinCost)
				if err != nil {
					t.Fatalf("error al generar una contraseña valida")
				}
				hashGenerated := string(hashBytes)
				hash = hashGenerated
			} else {
				hash = tt.storedInvalidHash
			}

			//1.2 preparamos el usuario que devolvera nuestro mock (Simula el resultado obtenido por findByEmail
			repo := &MockUserRepository{
				foundUser: domain.User{
					ID:           uuid.MustParse("019a15f2-8c34-7b21-9d45-6e8f0123abcd"),
					Email:        "mariojose@gmail.com",
					PasswordHash: hash,
					CreatedAt:    time.Date(2026, 10, 22, 9, 30, 0, 0, time.FixedZone("UTC-6", -6*60*60)),
				},
				err: tt.repoErr,
			}

			//2.Creamos el servicio y ejecutamos el metodo a probar, pasamos los argumentos necesarios
			service, err := NewAuthService(repo, bcrypt.MinCost)

			if err != nil {
				t.Fatalf("Error al cargar el servicio: %v", err)
			}

			err = service.Login(context.Background(), tt.email, tt.password)

			//Comprobamos, el error
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, se esperaba %v", err, tt.wantErr)
			}

			//Comprobamos que el servicio haya sido llamado
			if repo.findCalled != tt.wantCalled {
				t.Fatalf("repositorio llamado = %v, se esperaba %v", repo.findCalled, tt.wantCalled)
			}

			//Comprobamos que el email recibido por la búsqueda frente al esperado
			if repo.findEmail != tt.wantEmail {
				t.Fatalf("email recibido %q, se esperaba %q", repo.findEmail, tt.wantEmail)
			}

		})
	}

}
