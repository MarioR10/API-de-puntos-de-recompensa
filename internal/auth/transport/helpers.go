package transport

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Creamos un error, que devuelve el status HTTP ocurrido y el mensaje de lo que ocurrio
type malformedRequest struct {
	status  int
	message string
}

func (e *malformedRequest) Error() string {
	return e.message
}

const (
	MaxAllowedJSONSize = 1048576
)

// decodeJSONBody representa una funcion helper, que nos ayuda con las validaciones de decodificacion del JSON body
// y escribe el resultado del JSON body validado en data (que es un puntero)
func decodeJSONBody(w http.ResponseWriter, r *http.Request, data any) error {

	//--VALIDACION DE HEADER
	//1. Primera validacion: el header debe traer como valor el tipo correcto que estamos esperando para el body
	ct := r.Header.Get("Content-Type")

	if ct != "" {
		//De adentro hacia afuera
		//a) Divide ct cada vez que ve un ; devuelve un arreglo de strings con los elementos separados, obtenemos el primer elemento
		//b) Elimina espacios al inicio y al final del texto
		//c) normaliza todo a minuscula
		// ct := Content-Type: application/json; charset=utf-8  -> ["application/json","charset=utf-8"] -> application/json
		mediaType := strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))

		//Asi esta comparacion no falla, sabemos que viene todo bien escrito como lo que estamos comparando
		if mediaType != "application/json" {
			message := "Content-Type header differs from application/json"
			return &malformedRequest{status: http.StatusUnsupportedMediaType, message: message}
		}
	}

	//--VALIDACION DE BODY

	//2. Aca estamos reasignando el objeto que lee los bytes del body del request HTTP entrante
	//  le decimos:  Toma el lector que ya tenía r.Body y reemplázalo por otro lector que impone un límite de MaxAllowedJSONSize y ocupa ese para leer los bytes
	r.Body = http.MaxBytesReader(w, r.Body, MaxAllowedJSONSize)

	//3. Comenzamos el proceso de Deserializar

	// 3.1 Creamos un json.Decoder que usará r.Body como fuente de datos para leer e interpretar el JSON.
	decode := json.NewDecoder(r.Body)

	//3.2 Configuramos el decoder  "encendiendole" una opcion, para que valide que no vengan campos no especificados
	//El efecto viene despues cuando usamos Decode, ahi validara que no vengan campos que el struct donde desearializa no tenga
	//aqui es meramente configuracion del decoder
	decode.DisallowUnknownFields()

	// 3.3 Lee el JSON desde r.Body, lo interpreta y mapea sus datos al objeto Go recibido en data.
	err := decode.Decode(data)

	//Si algo salio mal en la decodificacion, identificamos las posibles causas que pudieron ocurrir
	if err != nil {

		//Creamos variables que apuntan a un tipo de error especifico
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var maxBytesError *http.MaxBytesError

		switch {
		// .As(...) → sabemos que ocurrió un error, pero queremos responder:
		// "¿Este error es del tipo que estoy buscando?"
		// Si lo es, lo asignamos a la variable que definimos como target,
		// que es de ese tipo, para poder acceder a la información específica del error.
		//
		// .Is(...) → sabemos que ocurrió un error, pero queremos responder:
		// "¿Este error corresponde al error específico que estoy buscando?"
		// Si corresponde, devuelve true. No necesitamos obtener el error ni
		// acceder a su información; solamente queremos saber si corresponde.

		// As → busca por TIPO y, si lo encuentra, me entrega el error.
		// Is → busca un ERROR específico y me dice si lo encuentra.

		//CASO 1: un JSON roto, por ejemplo { "email": }, no tiene nada despues  del ":", error de sintaxis
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, message: msg}

		//CASO 2: Un JSON incompleto, por ejemplo { "email": "mario@example.com } abrimos comillas pero nunca las cerramos
		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintf("The request body is truncated halfway")
			return &malformedRequest{status: http.StatusBadRequest, message: msg}

		//CASO 3: Un JSON valido, pero al deserealizar los tipos no coinciden, por ejemplo: {"email": 123},email → número,
		//pero el struct esperaba un string email → string
		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, message: msg}

		//CASO 4: Un cliente manda un campo que no esta especificado en el struct al que vamos a deserializar
		//ejemplo: {"email": "mario@example.com","edad": 30 } como configuramos decoder.DisallowUnknownFields() -> json: unknown field "edad"

		//nota: la libreria no nos da un tipo de error especifico para este error, no podmeos ocupar .As para obtener la informacion
		//o .Is si solo queremos comparar, aqui nos guiamos por el prefijo  "json: unknown field "
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			return &malformedRequest{status: http.StatusBadRequest, message: msg}

		//CASO 5: No hay un body, el decoder intenta leer el JSON, al no haber nada entonces devuelve que no hay nada que leer
		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			return &malformedRequest{status: http.StatusBadRequest, message: msg}

		//CASO 6: Valida que se cumpla el limite de peso para el JSON del body
		case errors.As(err, &maxBytesError):
			msg := fmt.Sprintf("Request body must not be larger than %d bytes", maxBytesError.Limit)
			return &malformedRequest{status: http.StatusRequestEntityTooLarge, message: msg}

		//CASO DEFAULT: El error que sucedio no se reconoce como ninguno de los anteriores esperados, lo devolvemos
		default:
			return err
		}
	}

	//Llamamos a Decode por segunda vez para detectar si hay otro valor JSON despues del primero
	//por ejemplo: {"email": "a@b.com", "password": "12345678"}{"email": "otro"}
	//                                                         ↑ esto
	// Si ya no hay mas devuelve io.EOF que es el que comparamos, como Decode espera un puntero a cualquier destino,le pasamos un struct anonimo vacio para que decodifique ahi
	//ya que realmente no necesitamos los valores que tiene el segundo JSON solo queremos validar que nos devuelve
	err = decode.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		msg := "Request body must only contain a single JSON object"
		return &malformedRequest{status: http.StatusBadRequest, message: msg}

	}
	return nil
}

// writeProblem es una funcion que representa lo que se le manda al cliente cuando ocurre un error en la peticion HTTP
func writeProblem(w http.ResponseWriter, status int, details string) {

	problemResponse := ErrorResponse{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: details,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problemResponse)
}

// writeJSON es una funcion que construye la repsuesta que se le manda al cliente
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)

}
