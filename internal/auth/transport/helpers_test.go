package transport

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
)

func Test_decodeJSONBody(t *testing.T) {

	test := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantEmail   string
	}{
		{
			name:        "Header correcto, tipo de body esperado",
			contentType: "application/json",
			body:        `{"email":"mario@test.com","password":"aB7dhsbdiuff"}`,
			wantStatus:  0,
			wantEmail:   "mario@test.com",
		},
		{
			name:        "Sin header",
			contentType: "",
			body:        `{"email":"mario@test.com","password":"aB7dhsbdiuff"}`,
			wantStatus:  0,
			wantEmail:   "mario@test.com",
		},
		{
			name:        "Header con parametros y mayusculas",
			contentType: "Application/JSON; charset = utf-8",
			body:        `{"email":"mario@test.com","password":"aB7dhsbdiuff"}`,
			wantStatus:  0,
			wantEmail:   "mario@test.com",
		},
		{
			name:        "Header incorrecto",
			contentType: "text/plain",
			body:        `{"email":"mario@test.com","password":"aB7dhsbdiuff"}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantEmail:   "",
		},
		{
			name:        "JSON roto",
			contentType: "application/json",
			body:        `{"email":}`,
			wantStatus:  http.StatusBadRequest,
			wantEmail:   "",
		},
		{
			name:        "JSON incompleto",
			contentType: "application/json",
			body:        `{ "email": "mario@example.com }`,
			wantStatus:  http.StatusBadRequest,
			wantEmail:   "",
		},
		{
			name:        "Tipo que no coincide",
			contentType: "application/json",
			body:        `{"email": 123}`,
			wantStatus:  http.StatusBadRequest,
			wantEmail:   "",
		},
		{
			name:        "Campo Desconocido",
			contentType: "application/json",
			body:        `{"email": "mario@example.com","edad": 30 }`,
			wantStatus:  http.StatusBadRequest,
			wantEmail:   "",
		},
		{
			name:        "Body vacio",
			contentType: "application/json",
			body:        ``,
			wantStatus:  http.StatusBadRequest,
			wantEmail:   "",
		},
		{
			name:        "body de mas de 1 MB",
			contentType: "application/json",
			body:        `{"email":"` + strings.Repeat("a", maxAllowedJSONSize) + `"}`,
			wantStatus:  http.StatusRequestEntityTooLarge,
			wantEmail:   "",
		},
		{
			name:        "dos objetos JSON seguidos",
			contentType: "application/json",
			body:        `{"email":"a@test.com"}{"email":"b@test.com"}`,
			wantStatus:  http.StatusBadRequest,
			wantEmail:   "",
		},
	}

	for _, tt := range test {

		t.Run(tt.name, func(t *testing.T) {

			request := httptest.NewRequest(
				http.MethodPost,
				"/v1/auth/register",
				strings.NewReader(tt.body),
			)

			if tt.contentType != "" {
				request.Header.Set("Content-Type", tt.contentType)
			}
			response := httptest.NewRecorder()

			var dst registerRequest
			err := decodeJSONBody(response, request, &dst)

			if tt.wantStatus == 0 {
				if err != nil {
					t.Fatalf("error inesperado: %v", err)
				}
				if dst.Email != tt.wantEmail {
					t.Errorf("EMAIL + %q, se esperaba %q", dst.Email, tt.wantEmail)
				}
				return
			}

			var mr *malformedRequest
			if !errors.As(err, &mr) {
				t.Fatalf("error + %v, se espera un *malformedRequest", err)
			}

			if mr.status != tt.wantStatus {
				t.Errorf("status = %d, se esperaba %d", mr.status, tt.wantStatus)
			}
		})
	}
}

func TestDecodeJSONBody_ReadError(t *testing.T) {
	errRead := errors.New("falla al leer el body")
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", iotest.ErrReader(errRead))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	var dst registerRequest
	err := decodeJSONBody(rec, req, &dst)

	var mr *malformedRequest
	if errors.As(err, &mr) {
		t.Fatalf("se esperaba un error interno, se obtuvo un *malformedRequest con status %d", mr.status)
	}
	if !errors.Is(err, errRead) {
		t.Errorf("error = %v, se esperaba %v", err, errRead)
	}
}
