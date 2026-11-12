package domain

import "errors"

// Errores centinela: errores de dominio que representan condiciones especificas que suceden en el flujo
// el codigo puede reconocer estos condicones por comparacion y tomar una decision
var ErrInvalidCredentials = errors.New("credenciales invalidas")
var ErrEmailAlreadyExists = errors.New("el email ya existe")
var ErrPasswordTooLong = errors.New("la contraseña excede el limite permitido")
var ErrPasswordTooShort = errors.New("la contraseña es muy corta")
var ErrUserNotFound = errors.New("el usuario no fue encontrado")
