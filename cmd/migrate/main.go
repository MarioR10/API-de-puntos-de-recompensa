package main

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/MarioR10/Proyecto-Go/internal/config"
	"github.com/MarioR10/Proyecto-Go/internal/platform/database"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("migracion fallida: %v", err)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load("application.yaml")
	if err != nil {
		return fmt.Errorf("cargando la configuracion: %w", err)
	}

	//Configuraciones para la base de datos
	parameters := database.ConfigDSN{
		Host:     cfg.GetString("database.postgres.host"),
		Port:     cfg.GetInt("database.postgres.port"),
		User:     cfg.GetString("database.postgres.user"),
		Password: cfg.GetString("database.postgres.password"),
		DBName:   cfg.GetString("database.postgres.dbname"),
		SSLMode:  cfg.GetString("database.postgres.sslmode"),
	}

	// Abrimos la conexión mediante la interfaz estándar (database/sql) que exige Goose,
	// utilizando internamente el driver 'pgx' de PostgreSQL y pasando el DSN (connection string).
	db, err := sql.Open("pgx", database.BuildDSN(parameters))
	if err != nil {
		return fmt.Errorf("abriendo una conexion: %w", err)
	}
	// Garantizamos que la conexión se cierre al finalizar la función, sin importar cómo termine.

	// defer recibe una llamada a una función y pospone su ejecución hasta que run() termine.
	// Por ejemplo, defer db.Close() le indica a Go que ejecute db.Close() al finalizar run().
	// Como necesitamos manejar el error que devuelve db.Close(), en lugar de llamar directamente
	// a db.Close(), hacemos que defer ejecute una función anónima que se encarga de llamar a
	// db.Close() y manejar el posible error.
	//
	// func() { ... } define la función anónima y los () finales la invocan.
	// Por eso defer puede posponer su ejecución, igual que hace con defer db.Close().
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close database", "error", err)
		}
	}()

	// Configuramos el dialecto a PostgreSQL para que Goose sepa qué sintaxis SQL
	// utilizar en sus operaciones internas de control de versiones.
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("configurando goose: %w", err)
	}

	//NOTA:
	//os.Args es un slice de string que Go llena a la hora que se ejecuta un comando
	//Es la forma en la que Go puede leer que escribimos despues del comando al ejecutarlo
	//os.Args[0] siempre es la ruta del propio programa
	//os.Args[1] es el argumento del comando que ejecutamos

	//Si ejecutamos make migrate-up entonces os.Args[1] vale "up"
	//Si ejecutamos make migrate-down entonces os.Args[1] vale "down"
	command := "up" //comando por defecto

	//si el slice es mayor a 1, entonces tenemos arguentos extra ademas del nombre/ruta del programa
	if len(os.Args) > 1 {
		command = os.Args[1] //le asignamos a command el valor del comando ejecutado
	}

	//dependiendo del caso ejecutamos la accion
	switch command {
	case "up":
		err = goose.Up(db, "migrations")
	case "down":
		err = goose.Down(db, "migrations")
	case "status":
		err = goose.Status(db, "migrations")
	default:
		err = fmt.Errorf("comando desconocido: %q (usa up, down o status)", command)
	}
	if err != nil {
		return fmt.Errorf("ejecutando %q: %w", command, err)
	}
	return nil
}
