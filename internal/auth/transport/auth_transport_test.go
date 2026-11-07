package transport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/MarioR10/Proyecto-Go/internal/auth/domain"
	"github.com/google/uuid"
)

type MockUserService struct {
	err    error
	user   domain.User
	called bool
}

func (m *MockUserService) Register(ctx context.Context, email, password string) (domain.User, error) {

	m.called = true
	if m.err != nil {
		return domain.User{}, m.err
	}
	return m.user, nil
}

func TestAuthHandler_Register(t *testing.T) {

	//CASOS DE PRUEBA: entradas y salidas esperadas

	validBody := `{"email": "mariotest@gmail.com", "password": "1234567"}`
	passwordTooLong := strings.Repeat("ñ", 74)

	//Error utilizado para que el reader falle al leer el body
	errRead := errors.New("falla al leer el body")
	errInesperado := errors.New("error inesperado")

	test := []struct {
		name            string
		contentType     string
		body            string
		wantCode        int
		wantCalled      bool
		wantContentType string
		wantResponse    registerResponse
		wantProblem     errorResponse
		bodyReader      io.Reader
		serverErr       error
	}{

		{
			name:            "Registration successful",
			contentType:     "application/json",
			body:            validBody,
			wantCode:        http.StatusCreated,
			wantCalled:      true,
			wantContentType: "application/json",
			wantResponse: registerResponse{
				ID:        uuid.MustParse("019a15f2-8c34-7b21-9d45-6e8f0123abcd"),
				Email:     "mariotest@gmail.com",
				CreatedAt: time.Date(2026, 10, 22, 15, 30, 0, 0, time.UTC),
			},
			wantProblem: errorResponse{},
		},
		{
			name:            "invalid email",
			contentType:     "application/json",
			body:            `{"email": "         ", "password": "1234567"}`,
			wantCode:        http.StatusBadRequest,
			wantCalled:      false,
			wantContentType: "application/problem+json",
			wantResponse:    registerResponse{},
			wantProblem: errorResponse{
				Type:   "about:blank",
				Title:  http.StatusText(http.StatusBadRequest),
				Status: http.StatusBadRequest,
				Detail: "when registering, the email field is required",
			},
		},
		{
			name:            "invalid password",
			contentType:     "application/json",
			body:            `{"email": "mariotest@gmail.com", "password": " "}`,
			wantCode:        http.StatusBadRequest,
			wantCalled:      false,
			wantContentType: "application/problem+json",
			wantResponse:    registerResponse{},
			wantProblem: errorResponse{
				Type:   "about:blank",
				Title:  http.StatusText(http.StatusBadRequest),
				Status: http.StatusBadRequest,
				Detail: "when registering, the password field is required",
			},
		},
		{
			name:            "invalid content-type",
			contentType:     "application/xml",
			body:            validBody,
			wantCode:        http.StatusUnsupportedMediaType,
			wantCalled:      false,
			wantContentType: "application/problem+json",
			wantResponse:    registerResponse{},
			wantProblem: errorResponse{
				Type:   "about:blank",
				Title:  http.StatusText(http.StatusUnsupportedMediaType),
				Status: http.StatusUnsupportedMediaType,
				Detail: "Content-Type header differs from application/json",
			},
		},
		{
			name:            "Error decoding JSON in the Decode method",
			contentType:     "application/json",
			body:            "",
			wantCode:        http.StatusInternalServerError,
			wantCalled:      false,
			wantContentType: "application/problem+json",
			wantResponse:    registerResponse{},
			wantProblem: errorResponse{
				Type:   "about:blank",
				Title:  http.StatusText(http.StatusInternalServerError),
				Status: http.StatusInternalServerError,
				Detail: http.StatusText(http.StatusInternalServerError),
			},
			bodyReader: iotest.ErrReader(errRead), //Construimos un lector que devuelve un error, cuando intente leer el body
		},
		{
			name:            "Duplicate email",
			contentType:     "application/json",
			body:            validBody,
			wantCode:        http.StatusConflict,
			wantCalled:      true,
			wantContentType: "application/problem+json",
			wantResponse:    registerResponse{},
			wantProblem: errorResponse{
				Type:   "about:blank",
				Title:  http.StatusText(http.StatusConflict),
				Status: http.StatusConflict,
				Detail: "The email address is already in use. Please try another one",
			},
			bodyReader: nil,
			serverErr:  domain.ErrEmailAlreadyExists,
		},
		{
			name:            "Password too long",
			contentType:     "application/json",
			body:            `{"email": "mariotest@gmail.com", "password":` + `"` + passwordTooLong + `"` + "}",
			wantCode:        http.StatusBadRequest,
			wantCalled:      true,
			wantContentType: "application/problem+json",
			wantResponse:    registerResponse{},
			wantProblem: errorResponse{
				Type:   "about:blank",
				Title:  http.StatusText(http.StatusBadRequest),
				Status: http.StatusBadRequest,
				Detail: "The password exceeds the maximum allowed length",
			},
			bodyReader: nil,
			serverErr:  domain.ErrPasswordTooLong,
		},
		{
			name:            "Password too short",
			contentType:     "application/json",
			body:            `{"email": "mariotest@gmail.com", "password": "123"}`,
			wantCode:        http.StatusBadRequest,
			wantCalled:      true,
			wantContentType: "application/problem+json",
			wantResponse:    registerResponse{},
			wantProblem: errorResponse{
				Type:   "about:blank",
				Title:  http.StatusText(http.StatusBadRequest),
				Status: http.StatusBadRequest,
				Detail: "The password is shorter than the minimum required length",
			},
			bodyReader: nil,
			serverErr:  domain.ErrPasswordTooShort,
		},
		{
			name:            "Internal Server Error",
			contentType:     "application/json",
			body:            validBody,
			wantCode:        http.StatusInternalServerError,
			wantCalled:      true,
			wantContentType: "application/problem+json",
			wantResponse:    registerResponse{},
			wantProblem: errorResponse{
				Type:   "about:blank",
				Title:  http.StatusText(http.StatusInternalServerError),
				Status: http.StatusInternalServerError,
				Detail: "An unexpected error occurred on our end. Please try again",
			},
			bodyReader: nil,
			serverErr:  errInesperado,
		},
	}

	for _, tt := range test {

		t.Run(tt.name, func(t *testing.T) {
			//PREPARACION: preparacion de request y response

			//elegir el reader que vamos a utilizar.
			reader := tt.bodyReader
			if reader == nil {
				reader = strings.NewReader(tt.body)
			}
			//Creamos la request
			request := httptest.NewRequest(
				http.MethodPost,
				"/v1/auth/register",
				reader,
			)
			//Le asignamos headers
			if tt.contentType != "" {
				request.Header.Set("Content-Type", tt.contentType)
			}

			//Creamos un recoder que escribira la response
			response := httptest.NewRecorder()

			//Creacion de dependencias: logger, mock. handler
			logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
			serviceMock := &MockUserService{
				user: struct {
					ID           uuid.UUID
					Email        string
					PasswordHash string
					CreatedAt    time.Time
				}{
					ID:           uuid.MustParse("019a15f2-8c34-7b21-9d45-6e8f0123abcd"),
					Email:        "mariotest@gmail.com",
					PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
					CreatedAt:    time.Date(2026, 10, 22, 9, 30, 0, 0, time.FixedZone("UTC-6", -6*60*60)),
				},
				err: tt.serverErr,
			}

			handler := NewAuthHandler(serviceMock, logger)

			//Ejecucion del handler creado, pasamos la request y response
			handler.Register(response, request)

			//Validaciones comunes para cada uno de los casos: Codigo HTTP esperado, llamada al servicio, content-Type
			if response.Code != tt.wantCode {
				t.Fatalf("code  %v, se espera un %v", response.Code, tt.wantCode)
			}

			if serviceMock.called != tt.wantCalled {
				t.Fatalf("servicio llamado = %v, se esperaba %v", serviceMock.called, tt.wantCalled)
			}

			if response.Header().Get("content-type") != tt.wantContentType {
				t.Fatalf("header equivocado = %v, se esperaba %v", response.Header().Get("content-type"), tt.wantContentType)
			}

			//VALIDACIONES del body de exito: interpretar el usuario y comprobar los campos: ID, email y fecha

			if tt.wantCode == http.StatusCreated {
				//Decodificamos
				userResponse := registerResponse{}

				// Conservamos createdAt como texto para comprobar su representación
				// en el JSON; esperamos que termine en Z, que indica UTC.
				date := struct {
					CreatedAt string `json:"createdAt"`
				}{}

				// Bytes permite acceder al contenido pendiente sin avanzar la posición
				// de lectura del buffer. No crea una copia independiente.
				// Unmarshal interpreta esos bytes y rellena date; como no lee mediante
				// el buffer, su posición permanece intacta.
				bytesBody := response.Body.Bytes()

				//Queremos validar que campos estamos obteniendo del JSON, los almacenamos en un map
				//las claves (string) son los nombres de los campos, los valores de estas llaves pueden tener distintos tipos (any)
				fieldsJson := map[string]any{}

				err := json.Unmarshal(bytesBody, &fieldsJson)
				if err != nil {
					t.Fatalf("Hubo un problema al decodificar el JSON : %s", err)
				}

				for key := range fieldsJson {

					if key != "id" && key != "email" && key != "createdAt" {
						t.Fatalf("Hay un campo inesperado en el JSON: %s", key)
					}
				}

				err = json.Unmarshal(bytesBody, &date)
				if err != nil {
					t.Fatalf("Hubo un problema al decodificar el JSON : %s", err)

				}

				//Con esto validamos que un texto termine con la letra Z
				//Este metodo nos ayuda a verificar que un string termine con un texto especifico

				if !strings.HasSuffix(date.CreatedAt, "Z") {
					t.Fatalf("Se esperaba una fecha UTC terminada en Z, se obtuvo: %s", date.CreatedAt)
				}

				// Decode lee mediante response.Body y avanza la posición del buffer.
				// Por eso inspeccionamos primero los bytes con Unmarshal: si un decoder
				// consumiera el body antes, una lectura posterior podría encontrar EOF.
				// Ahora rellenamos user para comparar sus campos con los esperados.
				err = json.NewDecoder(response.Body).Decode(&userResponse)

				if err != nil {
					t.Fatalf("Hubo un problema al decodificar el JSON : %s", err)

				}

				if userResponse.ID != tt.wantResponse.ID {
					t.Fatalf("ID equivocado = %v, se esperaba %v", userResponse.ID, tt.wantResponse.ID)

				}
				if userResponse.Email != tt.wantResponse.Email {
					t.Fatalf("email equivocado = %v, se esperaba %v", userResponse.Email, tt.wantResponse.Email)

				}
				if !userResponse.CreatedAt.Equal(tt.wantResponse.CreatedAt) {
					t.Fatalf("Fecha de creacion equivocada = %v, se esperaba %v", userResponse.CreatedAt, tt.wantResponse.CreatedAt)

				}

			} else {

				//COMPROBACIONES DE ERROR
				errResponse := &errorResponse{}

				err := json.NewDecoder(response.Body).Decode(errResponse)
				if err != nil {
					t.Fatalf("Hubo un problema al decodificar el JSON : %s", err)
				}

				if errResponse.Type != tt.wantProblem.Type {
					t.Fatalf("El type esperado era: %s, y recibimos: %s ", tt.wantProblem.Type, errResponse.Type)

				}
				if errResponse.Title != tt.wantProblem.Title {
					t.Fatalf("El titulo esperado era: %s, y recibimos: %s ", tt.wantProblem.Title, errResponse.Title)

				}

				if errResponse.Detail != tt.wantProblem.Detail {
					t.Fatalf("El mensaje esperado era: %s, y recibimos: %s ", tt.wantProblem.Detail, errResponse.Detail)
				}

				if errResponse.Status != tt.wantProblem.Status {
					t.Fatalf("Se espera el codigo: %d", tt.wantProblem.Status)

				}
			}

		})
	}
}
