package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"proyecto-Go/internal/config"
	"proyecto-Go/internal/platform/database"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("migracion fallida: %v", err)
	}
}

func run() error {
	cfg, err := config.Load("application.yaml")
	if err != nil {
		return fmt.Errorf("cargando las configuraciones: %w", err)
	}

	// Abrimos la conexión mediante la interfaz estándar (database/sql) que exige Goose,
	// utilizando internamente el driver 'pgx' de PostgreSQL y pasando el DSN (connection string).
	db, err := sql.Open("pgx", database.BuildDSN(cfg))
	if err != nil {
		return fmt.Errorf("abriendo una conexion: %w", err)
	}
	// Garantizamos que la conexión se cierre al finalizar la función, sin importar cómo termine.
	defer db.Close()

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
		err = fmt.Errorf("comando desconocido: %q (debe de usarse up, down o status)", command)
	}
	if err != nil {
		return fmt.Errorf("ejecutando %q: %w", command, err)
	}
	return nil
}
