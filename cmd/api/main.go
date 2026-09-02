// Command api es el punto de entrada del binario del servidor de la API.
//
// En la Fase 0 esto es solo un esqueleto que compila: nos da un binario
// ejecutable y confirma que el módulo está bien armado. En la Fase 1, este
// mismo main pasará a: cargar la configuración (paquete config), inicializar
// el logger (slog) y arrancar el servidor HTTP (chi) con graceful shutdown.
package main

import "fmt"

func main() {
	fmt.Println("API de puntos de recompensa — esqueleto (Fase 0)")
}
