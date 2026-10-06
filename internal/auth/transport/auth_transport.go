package transport

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/MarioR10/Proyecto-Go/internal/auth/domain"
	"github.com/MarioR10/Proyecto-Go/internal/auth/service"
)

//capa que traduce HTTP a negocio

type AuthHandler struct {
	authService service.IAuthService
	logger      *slog.Logger
}

func NewAuthHandler(authService service.IAuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{authService: authService, logger: logger}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {

	//Entidad
	var request RegisterRequest

	//1. Validamos el JSON body que viene el la request
	err := decodeJSONBody(w, r, &request)

	//1.1 Validamos que el error que devuelve sea del tipo que manejamos en la funcion helper
	//    si es asi, copiamos el contenido en dicha variable, para poder manejar sus datos, mensaje y status
	if err != nil {
		var mr *malformedRequest
		if errors.As(err, &mr) {
			writeProblem(w, mr.status, mr.message)
			return
		}
		h.logger.ErrorContext(r.Context(), "Ocurrio un error al decodificar el body del request ", "error", err.Error())
		writeProblem(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	//2. Validamos que los campos que vienen en la request no sean vacios //preguntar si es correcto
	if strings.TrimSpace(request.Password) == "" {
		msg := "when registering, the password field is required"
		writeProblem(w, http.StatusBadRequest, msg)
		return
	}
	if strings.TrimSpace(request.Email) == "" {
		msg := "when registering, the email field is required"
		writeProblem(w, http.StatusBadRequest, msg)
		return
	}

	//3. Llamamos al servicio
	user, err := h.authService.Register(r.Context(), request.Email, request.Password)

	if err != nil {

		switch {
		case errors.Is(err, domain.ErrEmailAlreadyExists):
			msg := "The email address is already in use. Please try another one"
			writeProblem(w, http.StatusConflict, msg)
			return
		case errors.Is(err, domain.ErrPasswordTooLong):
			msg := "The password exceeds the maximum allowed length"
			writeProblem(w, http.StatusBadRequest, msg)
			return
		case errors.Is(err, domain.ErrPasswordTooShort):
			msg := "The password is shorter than the minimum required length"
			writeProblem(w, http.StatusBadRequest, msg)
			return
		default:
			h.logger.ErrorContext(r.Context(), "Ocurrio un error en la llamada al servicio ", "error", err.Error())
			msg := "An unexpected error occurred on our end. Please try again"
			writeProblem(w, http.StatusInternalServerError, msg)
			return
		}
	}
	//4. Convertimos dominio a dto de respuesta
	responseDTO := RegisterResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}

	//enviamos la respuesta
	writeJSON(w, http.StatusCreated, responseDTO)
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterResponse struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}
