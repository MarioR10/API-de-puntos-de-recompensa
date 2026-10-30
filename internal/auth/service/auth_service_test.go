package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/MarioR10/Proyecto-Go/internal/auth/domain"
	"golang.org/x/crypto/bcrypt"
)

// MOCK FALSO DEL REPOSITORIO
type MockUserRepository struct {
	err    error       //permite simular un fallo del repositorio.
	saved  domain.User //permite inspeccionar qué usuario recibió el repositorio.
	called bool        //permite verificar si el servicio llegó a invocar CreateUser.
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {

	m.called = true
	m.saved = user
	if m.err != nil {
		return domain.User{}, m.err
	}
	return user, nil
}

func TestAuthService_Register(t *testing.T) {

	var errDummy = errors.New("La base de datos no responde")
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
			svc := NewAuthService(repo, bcrypt.MinCost)
			user, err := svc.Register(context.Background(), tt.email, tt.password)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, se esperaba %v", err, tt.wantErr)
			}

			if repo.called != tt.wantCalled {
				t.Fatalf("repositorio llamado = %v, se esperaba %v", repo.called, tt.wantCalled)
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
