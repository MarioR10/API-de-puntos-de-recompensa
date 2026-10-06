package repository

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"github.com/MarioR10/Proyecto-Go/internal/auth/domain"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// variable por la cual podemos acceder a los archivos .sql embebidos
//
//go:embed queries/*.sql
var queryFile embed.FS

// Implementacion concreta de la interfaz
// Tiene dos campos: el pool de conexiones para la base de datos
// la query a ejecutar
type userRepositoryImpl struct {
	pool  *pgxpool.Pool
	query string
}

// Constructor que permite crear la implementacion concreta: valida que el repositorio nazca en un estado consistente
// valida primero que no haya error al obtener la query, si hay erro falla rapido.
func NewUserRepository(pool *pgxpool.Pool) (domain.IUserRepository, error) {

	//obtenemos la query desde los archivos embebidos utilizando el metodo auxiliar
	query, err := loadQuery("insert_user.sql")

	//si ocurre un error lo documentamos, no es error de dominio, es error de infraestructura
	if err != nil {
		return nil, fmt.Errorf("construyendo el repositorio de usuarios: %w", err)
	}
	return &userRepositoryImpl{
		pool:  pool,
		query: query}, nil
}

// funcion principal, implementa el metodo de la interfaz, sirve para crear un usuario en la base de datos
func (repo *userRepositoryImpl) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {

	//variable donde se almacena el usuario creado
	var userCreated domain.User
	//1. obtenemos la query previamente cargada
	query := repo.query
	//2. Obtenemos el pool de conexiones y ejecutamos la consulta
	// QueryRow no devuelve error: guarda dentro del Row cualquier fallo (pool, red, rechazo
	// de Postgres o Scan) y es Scan quien lo entrega. Por eso este único err puede ser un
	// PgError de Postgres o algo totalmente distinto.
	err := repo.pool.QueryRow(
		ctx, query,
		//parametros que espera la querie en orden
		user.Email,
		user.PasswordHash,
	).Scan( // Si no hay error previo scaneamos el resultado en nuestra variable. Si lo hay Scan entrega el error
		&userCreated.ID,
		&userCreated.Email,
		&userCreated.PasswordHash,
		&userCreated.CreatedAt,
	)

	//Como pgx representa los errores SQL en Go
	var pgxErr *pgconn.PgError

	//Validamos si es un error de tipo PgError, si lo es verificamos si ese error postgres es debido a una violacion a UNIQUE
	if errors.As(err, &pgxErr) && pgxErr.Code == "23505" {
		return domain.User{}, fmt.Errorf("insertando usuario: %w", domain.ErrEmailAlreadyExists)
	}

	//Para cualquier otro error, validamos tambien para no mandar un error silencioso
	if err != nil {
		return domain.User{}, fmt.Errorf("insertando usuario: %w", err)
	}

	//Si no hay error, retornamos el usuario creado y nil
	return userCreated, nil
}

func loadQuery(name string) (string, error) {
	query, err := queryFile.ReadFile("queries/" + name)
	if err != nil {
		return "", fmt.Errorf("cargando el query %q: %w", name, err)
	}
	return string(query), nil
}
