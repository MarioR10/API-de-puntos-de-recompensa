// Command api es el punto de entrada del binario del servidor de la API.
//
// En la Fase 0 esto es solo un esqueleto que compila: nos da un binario
// ejecutable y confirma que el módulo está bien armado. En la Fase 1, este
// mismo main pasará a: cargar la configuración (paquete config), inicializar
// el logger (slog) y arrancar el servidor HTTP (chi) con graceful shutdown.
package main

import (
	"fmt"
	"log"
	"net/http"
	"proyecto-Go/internal/config"

	"github.com/go-chi/chi/v5"
)

func main() {

	//Cargamos las configuraciones del  yaml
	cfg, err := config.Load("application.yaml")
	if err != nil {
		log.Fatalf("no se pudo cargar la configuracion %v", err)
	}

	//Esta funcion nos devuelve un enrutador, que cumple el contrato http.Handler de net/http,
	//por eso es compatible, net/http solicita que cualquier enrutador implemente su contrato para ser compatible
	router := chi.NewRouter()

	//Regsitramos metodo HTTP (Get), mas ruta, mas funcion
	//En este caso: GET /health response 200
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	//Esto nos devuelve el int (8080)
	//lo formatemos y lo convertimos a string(":8080")
	//d es un placeholder donde metemos el 8080
	addr := fmt.Sprintf(":%d", cfg.ServerPort())

	log.Printf("Servidor escuchando en el puerto %s", addr)

	//Funcion que arranca el servidor: en el puerto especificado, y con un router como manejador
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("El servidor murio: %v", err)
	}
}
