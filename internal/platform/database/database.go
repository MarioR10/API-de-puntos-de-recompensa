package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
