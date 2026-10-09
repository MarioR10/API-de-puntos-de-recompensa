package domain

import "context"

type UserRepository interface {
	//Metodo que nos ayuda en la construccion de un nuevo usuario en la base de datos
	//el campo ID y CreateAt se ignoran para el usuario pasado,, ya que son campos que la base los genera
	//Si el email utilizado existe, devuelve un error de dominio: ErrEmailAlreadyExists
	CreateUser(ctx context.Context, user User) (User, error)
}
