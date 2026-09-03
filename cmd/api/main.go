// Command api es el punto de entrada del binario del servidor de la API.
//
// En la Fase 0 esto es solo un esqueleto que compila: nos da un binario
// ejecutable y confirma que el módulo está bien armado. En la Fase 1, este
// mismo main pasará a: cargar la configuración (paquete config), inicializar
// el logger (slog) y arrancar el servidor HTTP (chi) con graceful shutdown.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"proyecto-Go/internal/config"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
)

func main() {

	// Creamos un logger que escribe los logs en formato JSON
	// y los envía a stdout (la salida estándar de la aplicación).
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Establecemos este logger como el logger predeterminado
	// de toda la aplicación.
	slog.SetDefault(logger)

	// Cargamos la configuración desde application.yaml.
	cfg, err := config.Load("application.yaml")

	// Si la configuración no pudo cargarse, no podemos continuar.
	if err != nil {

		// Los argumentos después del mensaje se pasan en pares:
		// "clave", valor
		logger.Error("no se pudieron cargar la configuraciones", "error", err)
		// Terminamos inmediatamente el proceso indicando que ocurrió
		// un error (código de salida 1).
		os.Exit(1)
	}

	// Creamos el router de chi.
	// chi.Router implementa la interfaz http.Handler de Go,
	// por lo que puede utilizarse directamente como Handler
	// de un http.Server.
	router := chi.NewRouter()

	// Registramos una ruta:  GET /health
	// Cuando alguien haga esta petición, respondemos con HTTP 200
	// y el texto "ok".
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Obtenemos el puerto configurado, por ejemplo: 8080
	// y lo convertimos al formato que necesita http.Server: ":8080"
	// %d indica que queremos insertar un número entero.
	addr := fmt.Sprintf(":%d", cfg.ServerPort())

	//Creamos un objeto server. Es un struct que representa el servidor web con su configuracion.
	//le pasamos la direccion en la que escucha y el router que atiende las peticiones
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Creamos un contexto que se cancelará cuando el sistema
	// reciba una señal de apagado.
	// os.Interrupt -> Ctrl+C
	// SIGTERM      -> señal utilizada normalmente para solicitar
	//                 el apagado de un proceso (por ejemplo, Docker).
	//
	// NotifyContext NO apaga el servidor.
	// Solamente detecta la señal y cancela el contexto.
	// ctx  -> contexto que será cancelado
	// stop -> función para cancelar manualmente el contexto
	//         y dejar de recibir señales.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	// Cuando main termine, ejecutamos stop para liberar
	// los recursos asociados al contexto.
	defer stop()

	// Ejecutamos ListenAndServe en una goroutine porque es un método
	// bloqueante: mientras el servidor está funcionando, esta llamada
	// permanece ejecutándose atendiendo peticiones.
	//
	// Al utilizar "go", main puede continuar y quedarse esperando la señal de apagado.
	//dentro del goroutine se levanta el servidor web
	go func() {
		logger.Info("servicio escuchando", "addr", addr)

		// ListenAndServe arranca el servidor y normalmente permanece
		// bloqueado mientras el servidor está funcionando.
		//
		// Cuando llamemos a srv.Shutdown(), ListenAndServe terminará
		// y devolverá http.ErrServerClosed.
		//
		// Ese error NO significa que el servidor haya fallado.
		// Significa que fue cerrado intencionalmente.
		//
		// Por eso solamente tratamos como error real aquellos errores
		// diferentes de http.ErrServerClosed.
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("el servidor murio", "error", err)
			// Si el servidor falla antes de recibir una señal del SO,
			// cancelamos el contexto para despertar al main.
			stop()
		}
	}()

	// main se queda bloqueado aquí.
	// ctx.Done() devuelve un canal utilizado para avisar
	// que el contexto fue cancelado.
	// <- significa "esperar/recibir del canal".
	// Por lo tanto:
	//     <-ctx.Done()
	// significa:
	//     "Espera aquí hasta que el contexto sea cancelado."
	// Esto evita que main termine mientras el servidor sigue
	// funcionando en la goroutine.
	<-ctx.Done()
	logger.Info("apagando el servidor")

	// Creamos un nuevo contexto con un límite de 10 segundos.
	// Shutdown utilizará este contexto para esperar a que terminen
	// las peticiones que todavía están siendo procesadas.
	// Si después de 10 segundos todavía existen peticiones activas,
	// el contexto se cancela y Shutdown deja de esperar.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// Liberamos los recursos asociados al contexto cuando terminemos.
	defer cancel()

	// Le indicamos al servidor que debe apagarse de forma elegante.
	// Shutdown:
	// 1. Deja de aceptar nuevas conexiones/peticiones.
	// 2. Espera a que terminen las peticiones que ya están ejecutándose.
	// 3. Respeta el límite de tiempo definido en shutdownCtx.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("apagado forzado (se acabo el plazo de espera)", "error", err)
		os.Exit(1)
	}
	logger.Info("servidor apagado correctamente")
}
