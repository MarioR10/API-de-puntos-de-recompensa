package database

import (
	"context"
	"fmt"
	"time"

	"github.com/MarioR10/Proyecto-Go/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Arma y verifica un pool de conexiones a postgres
func NewPool(ctx context.Context, configurations *config.Config) (*pgxpool.Pool, error) {

	//Obtenemos la conexion a la base de datos
	strconn := BuildDSN(configurations)

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

// Metodo que arma el string connection (DSN) a partir de la config
func BuildDSN(configurations *config.Config) string {

	//Plantilla para la dsn
	//La dejamos aca y no en el archivo de propiedades para no depender de una plantilla externa que alguien puede
	//modificar y cambiar el orden
	connection := "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s"

	strconnection := fmt.Sprintf(connection,
		configurations.GetString("database.postgres.host"),
		configurations.GetInt("database.postgres.port"),
		configurations.GetString("database.postgres.user"),
		configurations.GetString("database.postgres.password"),
		configurations.GetString("database.postgres.dbname"),
		configurations.GetString("database.postgres.sslmode"),
	)
	return strconnection
}
