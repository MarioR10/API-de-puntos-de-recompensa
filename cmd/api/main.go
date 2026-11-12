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
	"strings"
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
	minPortAccepted = 1
	maxPortAccepted = 65535
)

func main() {

	//main minimo: llama a la funcion run(), si algo falla loguea y sale
	if err := run(); err != nil {
		slog.Error("error fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Creacion de LOGGER que escribe los logs en formato JSON
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// Establecemos este logger como el logger predeterminado
	slog.SetDefault(logger)

	// 2. Carga de configuraciones  desde application.yaml.
	cfg, err := config.Load("application.yaml")
	// Si la configuración no pudo cargarse, no podemos continuar.
	if err != nil {
		return fmt.Errorf("cargando las configuraciones: %w", err)
	}

	//3. Creacion de contexto, implementando graceful shutdown
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	// Cuando run termine, ejecutamos stop para liberar los recursos asociados al contexto.
	defer stop()

	//4. Llamado a las configuraciones ya cargadas y validadas
	configurations, err := loadRunConfigurations(cfg)
	if err != nil {
		return fmt.Errorf("validando las configuraciones: %w", err)
	}

	//5. Creacion de Pool de conexiones a Postgres
	pool, err := database.NewPool(ctx, configurations.databaseConfig)
	if err != nil {
		return fmt.Errorf("conectando a la base de datos: %w", err)
	}
	defer pool.Close() //corre al retornar run(): cierra el pool de conexiones
	logger.Info("conectado a la base de datos")

	//6. Creacion del repositorio que conecta a la abse de datos
	userRepo := repository.NewUserRepository(pool)

	//7. Creacion del servicio de autenticacion
	authService, err := service.NewAuthService(userRepo, configurations.bcryptCost)

	if err != nil{
		return fmt.Errorf("creando servicio de autenticación: %w", err)
	}

	//8. Creacion del handler del servicio de autenticacion
	authHandler := transport.NewAuthHandler(authService, logger)
	// Creamos el router de chi.
	// chi.Router implementa la interfaz http.Handler de Go,
	// por lo que puede utilizarse directamente como Handler
	// de un http.Server.
	router := chi.NewRouter()

	// Registramos una ruta:  GET /health
	// Cuando alguien haga esta petición, respondemos con HTTP 200
	// y el texto "ok".
	router.Route(
		"/v1", func(r chi.Router) {
			r.Post("/auth/register", authHandler.Register)
			r.Post("/auth/login", authHandler.Login)

		})

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	//9. Creacion del servidor (objeto Server). Es un struct que representa el servidor web con su configuracion.
	//le pasamos la direccion en la que escucha y el router que atiende las peticiones

	srv := &http.Server{
		Addr:              configurations.serverConfig.addr,
		IdleTimeout:       configurations.serverConfig.idleTimeout,
		ReadTimeout:       configurations.serverConfig.readTimeout,
		ReadHeaderTimeout: configurations.serverConfig.readHeaderTimeout,
		WriteTimeout:      configurations.serverConfig.writeTimeout,
		Handler:           router,
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

	// Ejecutamos ListenAndServe en una goroutine porque es un método
	// bloqueante: mientras el servidor está funcionando, esta llamada
	// permanece ejecutándose atendiendo peticiones.
	//
	// Al utilizar "go", run() puede continuar y quedarse esperando la señal de apagado.
	//dentro del goroutine se levanta el servidor web
	go func() {
		logger.Info("servicio escuchando", "addr", configurations.serverConfig.addr)

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

	// run() se queda bloqueado aquí.
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
		return fmt.Errorf("apagando el servidor: %w", err)
	}
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

func loadParameterDatabase(cfg *config.Config, path string) (string, error) {
	parameter := strings.TrimSpace(cfg.GetString(path))
	if parameter == "" {
		return "", fmt.Errorf("%q no debe ser vacio", path)
	}

	return parameter, nil
}

func loadPort(cfg *config.Config, path string) (int, error) {
	port := cfg.GetInt(path)
	if port < minPortAccepted || port > maxPortAccepted {
		return 0, fmt.Errorf("%q vale %v, debe estar entre %v y %v", path, port, minPortAccepted, maxPortAccepted)
	}
	return port, nil
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
	host, err := loadParameterDatabase(cfg, "database.postgres.host")
	if err != nil {
		errs = append(errs, err)
	}
	portDatabase, err := loadPort(cfg, "database.postgres.port")
	if err != nil {
		errs = append(errs, err)
	}
	user, err := loadParameterDatabase(cfg, "database.postgres.user")
	if err != nil {
		errs = append(errs, err)
	}
	password, err := loadParameterDatabase(cfg, "database.postgres.password")
	if err != nil {
		errs = append(errs, err)
	}

	dbname, err := loadParameterDatabase(cfg, "database.postgres.dbname")
	if err != nil {
		errs = append(errs, err)
	}
	sslMode, err := loadParameterDatabase(cfg, "database.postgres.sslmode")
	if err != nil {
		errs = append(errs, err)
	}

	cost := cfg.GetInt("bcrypt.cost")
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		errs = append(errs, fmt.Errorf("bcrypt.cost vale %d, debe estar entre %d y %d", cost, bcrypt.MinCost, bcrypt.MaxCost))
	}

	portServer, err := loadPort(cfg, "server.port")
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

	//Unimos todos los errores que salieron durante el proceso, si no es nil devolvemos el error [ETAPA: DECISION]
	errJoin := errors.Join(errs...)
	if errJoin != nil {
		return runConfig{}, errJoin
	}

	//3. Creamos el objetos con las configuraciones ya validadas [ETAPA: ARMAR]
	r := runConfig{
		databaseConfig: database.ConfigDSN{
			Host:     host,
			Port:     portDatabase,
			User:     user,
			Password: password,
			DBName:   dbname,
			SSLMode:  sslMode,
		},

		bcryptCost: cost,

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
