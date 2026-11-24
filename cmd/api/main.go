package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MarioR10/Proyecto-Go/internal/auth/repository"
	"github.com/MarioR10/Proyecto-Go/internal/auth/service"
	"github.com/MarioR10/Proyecto-Go/internal/auth/transport"
	"github.com/MarioR10/Proyecto-Go/internal/config"
	"github.com/MarioR10/Proyecto-Go/internal/platform/database"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	minTimeOutValue = 1 * time.Second
)

func main() {

	// main ejecuta run. Si recibe un error, lo registra y sale con código 1.
	if err := run(); err != nil {
		slog.Error("error fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {

	// Configuramos slog para escribir logs JSON en stdout por defecto.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Leemos la configuración desde application.yaml.
	cfg, err := config.Load("application.yaml")
	if err != nil {
		return fmt.Errorf("cargando las configuraciones: %w", err)
	}

	// Creamos ctx para detectar cuándo se solicita el apagado del proceso.
	// os.Interrupt llega con Ctrl+C. SIGTERM se usa, por ejemplo, al detener o reiniciar un contenedor de Docker.
	// NotifyContext cancela ctx al recibir una de esas señales. Eso cierra ctx.Done() y permite que run continúe y llame a Shutdown.
	// stop() nos permite cancelar ctx desde el código y dejar de recibir  notificaciones de os.Interrupt y SIGTERM para este contexto.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop() // Cuando run termine, ejecutamos stop para liberar los recursos asociados al contexto.

	// Validamos la configuración antes de abrir conexiones.
	configurations, err := loadRunConfigurations(cfg)
	if err != nil {
		return fmt.Errorf("validando las configuraciones: %w", err)
	}

	// Abrimos el pool de conexiones a PostgreSQL.
	pool, err := database.NewPool(ctx, configurations.databaseConfig)
	if err != nil {
		return fmt.Errorf("conectando a la base de datos: %w", err)
	}
	defer pool.Close() //corre al retornar run(): cierra el pool de conexiones
	logger.Info("conectado a la base de datos")

	// El repositorio usa el pool, el servicio usa el repositorio y el handler usa el servicio.
	userRepo := repository.NewUserRepository(pool)

	authService, err := service.NewAuthService(userRepo, configurations.bcryptCost)
	if err != nil {
		return fmt.Errorf("creando servicio de autenticación: %w", err)
	}
	authHandler := transport.NewAuthHandler(authService, logger)

	// Creamos el router de chi.chi.Router implementa la interfaz http.Handler de Go, por lo que puede utilizarse directamente como Handler de un http.Server.
	router := chi.NewRouter()

	// Registramos rutas :
	router.Route(
		"/v1", func(r chi.Router) {
			r.Post("/auth/register", authHandler.Register)
			r.Post("/auth/login", authHandler.Login)

		})
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	//Creacion del servidor (objeto Server). Es un struct que representa el servidor web con su configuracion.
	//le pasamos la direccion en la que escucha y el router que atiende las peticiones

	srv := &http.Server{
		Addr:              configurations.serverConfig.addr,
		IdleTimeout:       configurations.serverConfig.idleTimeout,
		ReadTimeout:       configurations.serverConfig.readTimeout,
		ReadHeaderTimeout: configurations.serverConfig.readHeaderTimeout,
		WriteTimeout:      configurations.serverConfig.writeTimeout,
		Handler:           router,
	}

	// Creamos un canal  queempieza vacío y puede guardar un (1) error (chan error) aunque run todavía no lo esté leyendo.
	// esto crea un espacio para que se puedan enviar errores entre las goroutines
	ErrChannel := make(chan error, 1)

	// ListenAndServe arranca el servidor y bloquea mientras atiende peticiones.
	// Lo ejecutamos en otra goroutine para que run pueda esperar en <-ctx.Done() la señal de apagado del ctx.
	go func() {
		logger.Info("servicio escuchando", "addr", configurations.serverConfig.addr)

		// Esta goroutine se queda dentro de ListenAndServe mientras el servidor está activo.
		// err recibe un valor cuando la llamada retorna.
		// ListenAndServe siempre devuelve un error que indica cómo terminó.
		//Puede ser un fallo real como un puerto ocupado, o un
		// http.ErrServerClosed que es la manera en que Go indica que el servicio se cerro mediante Shutdown
		err := srv.ListenAndServe()

		//Cuando la llamada por fin retorna, nuesta goroutine hace dos cosas:

		// Enviamos ese error a run. Como el canal tiene un búfer de 1,
		// podemos guardar el resultado aunque run todavía esté en <-ctx.Done().
		// Eso nos permite seguir hasta stop() si la escucha falló.
		ErrChannel <- err //enviamos el error por el canal, para comunicarselo a la otra goroutines (canal <- valor)

		// Si el apagado empezó por Ctrl+C o SIGTERM, la señal ya canceló ctx.
		// run salió de <-ctx.Done() y llamó a Shutdown. Al cerrarse la escucha,
		// ListenAndServe devuelve http.ErrServerClosed y esta goroutine termina
		// después de enviar ese resultado por el canal.
		//
		// Si la escucha falla antes de que llegue una señal, guardamos el error
		// en el canal y llamamos a stop() para cancelar ctx y desbloquear a run.
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			stop()
		}
	}()

	// La goroutine principal se queda esperando aquí, dentro de run.
	// ctx.Done() devuelve un canal y <- espera para recibir de él.
	// Cuando una señal o una llamada a stop() cancela ctx, el canal se cierra
	// y run puede continuar. Mientras tanto, main sigue esperando que run retorne.
	<-ctx.Done()
	logger.Info("apagando el servidor")

	// Como ctx ya está cancelado, creamos un contexto nuevo para el apagado.
	// Le damos a Shutdown hasta 10 segundos para esperar las peticiones activas.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// Liberamos los recursos asociados al contexto cuando terminemos.
	defer cancel()

	// Shutdown deja de aceptar nuevas conexiones y espera a que terminen
	// las peticiones activas. Si se agota el plazo, deja de esperar y devuelve
	// el error del contexto. Si el apagado termina bien, devuelve nil.
	shutdownErr := srv.Shutdown(shutdownCtx)

	// Leemos el resultado que la otra goroutine envió al terminar ListenAndServe.
	// http.ErrServerClosed indica que la escucha terminó por el cierre del servidor.
	errSave := <-ErrChannel

	//ListenAndServe siempre devuelve un error. ErrServerClosed indica el cierre esperado.
	// Si la escucha falló, devolvemos ese error a main y terminamos run.
	// En ese caso, ya llamamos a Shutdown, pero omitimos revisar su resultado.
	if !errors.Is(errSave, http.ErrServerClosed) {
		return fmt.Errorf("ejecutando el servidor: %w", errSave)
	}

	// Si la escucha terminó por un cierre esperado, revisamos el apagado.
	// Shutdown puede fallar, por ejemplo, si se agota el plazo mientras
	// todavía hay peticiones activas.
	if shutdownErr != nil {
		return fmt.Errorf("apagando el servidor: %w", shutdownErr)
	}

	// La escucha se cerró como esperábamos y el apagado terminó bien.
	logger.Info("servidor apagado correctamente")
	return nil
}

func loadTimeouts(cfg *config.Config, path string) (time.Duration, error) {
	duration := cfg.GetDuration(path)
	if duration < minTimeOutValue {
		return 0, fmt.Errorf("%q vale %s, debe ser al menos %s", path, duration, minTimeOutValue)
	}
	return duration, nil
}

type serverConfig struct {
	addr              string
	idleTimeout       time.Duration
	readTimeout       time.Duration
	readHeaderTimeout time.Duration
	writeTimeout      time.Duration
}
type runConfig struct {
	databaseConfig database.ConfigDSN
	bcryptCost     int
	serverConfig   serverConfig
}

func loadRunConfigurations(cfg *config.Config) (runConfig, error) {

	//1.Creamos el slice donde vamos almacenar errores de configuraciones
	var errs []error

	//2.Validamos todas las configuraciones, antes de asignarlas a los structs correspondientes [ETAPA: Validacion]
	databaseConfig := database.ConfigDSN{
		Host:     cfg.GetString("database.postgres.host"),
		Port:     cfg.GetInt("database.postgres.port"),
		User:     cfg.GetString("database.postgres.user"),
		Password: cfg.GetString("database.postgres.password"),
		DBName:   cfg.GetString("database.postgres.dbname"),
		SSLMode:  cfg.GetString("database.postgres.sslmode"),
	}
	err := databaseConfig.ValidateDatabaseParameters()
	if err != nil {
		errs = append(errs, err)
	}

	cost := cfg.GetInt("bcrypt.cost")
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		errs = append(errs, fmt.Errorf("bcrypt.cost vale %d, debe estar entre %d y %d", cost, bcrypt.MinCost, bcrypt.MaxCost))
	}

	portServer, err := config.LoadPort(cfg, "server.port")
	if err != nil {
		errs = append(errs, err)
	}
	addr := fmt.Sprintf(":%d", portServer)

	idleTimeout, err := loadTimeouts(cfg, "server.timeouts.idle")
	if err != nil {
		errs = append(errs, err)
	}
	readTimeout, err := loadTimeouts(cfg, "server.timeouts.read")
	if err != nil {
		errs = append(errs, err)
	}
	readHeaderTimeout, err := loadTimeouts(cfg, "server.timeouts.read_header")
	if err != nil {
		errs = append(errs, err)
	}
	writeTimeout, err := loadTimeouts(cfg, "server.timeouts.write")
	if err != nil {
		errs = append(errs, err)
	}

	//Unimos(envolvemos) todos los errores que salieron durante el proceso en un solo error, si no es nil devolvemos el error [ETAPA: DECISION]
	//De esta forman devolvemos la unidad y luego podemos comparar donde necesitemos validar, con  errors.Is o .As
	errJoin := errors.Join(errs...)
	if errJoin != nil {
		return runConfig{}, errJoin
	} //si todos son nil, entonces errJoin es nil

	//3. Creamos el objetos con las configuraciones ya validadas [ETAPA: ARMAR]
	r := runConfig{
		databaseConfig: databaseConfig,
		bcryptCost:     cost,
		serverConfig: serverConfig{
			addr:              addr,
			idleTimeout:       idleTimeout,
			readTimeout:       readTimeout,
			readHeaderTimeout: readHeaderTimeout,
			writeTimeout:      writeTimeout,
		},
	}
	return r, nil
}
