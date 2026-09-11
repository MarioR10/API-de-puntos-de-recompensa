package user

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed queries/*.sql
var queryFile embed.FS

type userRepositoryImpl struct {
	pool            *pgxpool.Pool
	countUsersQuery string
}

func NewRepository(pool *pgxpool.Pool) (IUserRepository, error) {

	//Hacemos esta validacion aca, para fallar rapido si hubo un error al obtener la query
	//evitamos que el error salga en etapas posteriores, fallamos rapido al iniciar la aplicacion
	countUsersQuery, err := loadQuery("count_users.sql")
	if err != nil {
		return nil, fmt.Errorf("al obtener el SQL: %w", err)
	}

	return &userRepositoryImpl{
		pool:            pool,
		countUsersQuery: countUsersQuery}, nil
}

func (repo *userRepositoryImpl) CountUsers(ctx context.Context) (int64, error) {

	var result int64
	err := repo.pool.QueryRow(ctx, repo.countUsersQuery).Scan(&result)

	if err != nil {

		return 0, fmt.Errorf("contando usuarios: %w", err)
	}

	return result, nil
}

// Metodo que nos ayuda a obtener el sql que se necesita
func loadQuery(name string) (string, error) {
	query, err := queryFile.ReadFile("queries/" + name)
	if err != nil {
		return "", fmt.Errorf("al cargar el sql: %w", err)
	}
	//query es bytes[], con string(query) convertimos el SQL a string
	return string(query), nil

}
