package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/MarioR10/Proyecto-Go/internal/auth/domain"
	"github.com/jackc/pgx/v5"

	_ "embed"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// variable por la cual podemos acceder a los archivos .sql embebidos
var (
	//go:embed queries/insert_user.sql
	sqlInsertUser string

	//go:embed queries/find_user_by_email.sql
	sqlFindUser string
)

// Implementacion concreta de la interfaz
// Tiene dos campos: el pool de conexiones para la base de datos
// la query a ejecutar
type UserRepository struct {
	pool  *pgxpool.Pool
}

// Constructor que permite crear la implementacion concreta
func NewUserRepository(pool *pgxpool.Pool) (*UserRepository) {

	return &UserRepository{
		pool: pool}
}

// funcion principal, implementa el metodo de la interfaz, sirve para crear un usuario en la base de datos
func (repo *UserRepository) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {

	//variable donde se almacena el usuario creado
	var userCreated domain.User
	//1. obtenemos la query previamente cargada
	query := sqlInsertUser

	//2. Obtenemos el pool de conexiones y ejecutamos la consulta
	// QueryRow no devuelve error: guarda dentro del Row cualquier fallo (pool, red, rechazo
	// de Postgres o Scan) y es Scan quien lo entrega. Por eso este único err puede ser un
	// PgError de Postgres o algo totalmente distinto.
	err := repo.pool.QueryRow(
		ctx, query,
		//parametros que espera la querie en orden
		user.ID,
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

func (repo *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {

	//1. variable que representa el usuario encontrado
	var userFound domain.User

	//2. obtenemos la query
	query := sqlFindUser

	//3. Abrimos la conexion con la base de datos
	err := repo.pool.QueryRow(
		//Parametros que espera la query
		ctx,
		query,
		email).Scan(
		//Donde almacenamos el reusltado, siempre tener cuidaod en el orden
		&userFound.ID,
		&userFound.Email,
		&userFound.PasswordHash,
		&userFound.CreatedAt,
	)

	//Validamos el tipo de error recibido
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, fmt.Errorf("al consultar por el usuario: %w", domain.ErrUserNotFound)
	}

	//Para cualquier otro error
	if err != nil {
		return domain.User{}, fmt.Errorf("al consultar por el usuario: %w", err)
	}

	return userFound, nil

}
