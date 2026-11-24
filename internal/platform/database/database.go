package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	minPortAccepted = 1
	maxPortAccepted = 65535
)

// Arma y verifica un pool de conexiones a postgres
func NewPool(ctx context.Context, config ConfigDSN) (*pgxpool.Pool, error) {

	//Obtenemos la conexion a la base de datos
	strconn := BuildDSN(config)

	//Crea el pool de conexiones: es perezozo, todavia no garantiza que Postgres responda
	pool, err := pgxpool.New(ctx, strconn)
	if err != nil {
		return nil, fmt.Errorf("creando pool de conexiones: %w", err)
	}

	//Por eso hacemos ping, para forzar una conexion real y verificar que si responda
	//asi fallamos rapido si algo sale mal
	ping, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(ping); err != nil {
		pool.Close() //Si el ping falla cerramos el pool
		return nil, fmt.Errorf("verificando conexion (ping): %w", err)
	}
	return pool, nil
}

// Metodo que arma el string connection (DSN)
func BuildDSN(config ConfigDSN) string {

	//Plantilla para la dsn
	//La dejamos aca y no en el archivo de propiedades para no depender de una plantilla externa que alguien puede
	//modificar y cambiar el orden
	connection := "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s"

	strconnection := fmt.Sprintf(connection,
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.DBName,
		config.SSLMode,
	)
	return strconnection
}

type ConfigDSN struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

//Nos permite tener centralizada las validaciones de los parametros que nos permiten abrir conexiones con postgreSQL

func (c ConfigDSN) ValidateDatabaseParameters() error {

	//1.Creamos el slice donde vamos almacenar errores de configuraciones
	var errs []error

	//2.Validamos todas las configuraciones, antes de asignarlas a los structs correspondientes [ETAPA: Validacion]
	err := validateParameterDatabase(c.Host, "host")
	if err != nil {
		errs = append(errs, err)
	}
	err = validatePort(c.Port, "port")
	if err != nil {
		errs = append(errs, err)
	}
	err = validateParameterDatabase(c.User, "user")
	if err != nil {
		errs = append(errs, err)
	}
	err = validateParameterDatabase(c.Password, "password")
	if err != nil {
		errs = append(errs, err)
	}
	err = validateParameterDatabase(c.DBName, "DBName")
	if err != nil {
		errs = append(errs, err)
	}
	err = validateParameterDatabase(c.SSLMode, "SSLMode")
	if err != nil {
		errs = append(errs, err)
	}

	//Unimos(envolvemos) todos los errores que salieron durante el proceso en un solo error, si no es nil devolvemos el error [ETAPA: DECISION]
	//De esta forman devolvemos la unidad y luego podemos comparar donde necesitemos validar, con  errors.Is o .As
	errJoin := errors.Join(errs...)
	if errJoin != nil {
		return errJoin
	} //si todos son nil, entonces errJoin es nil

	return nil
}

func validateParameterDatabase(value, parameter string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("el %s no debe ser vacio", parameter)
	}
	return nil
}

func validatePort(value int, parameter string) error {
	if value < minPortAccepted || value > maxPortAccepted {
		return fmt.Errorf(" el %s vale: %v debe estar entre %v y %v", parameter, value, minPortAccepted, maxPortAccepted)
	}
	return nil
}
