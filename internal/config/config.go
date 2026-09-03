package config

//Abstraccion que nos permite extraer configuraciones desde el archivo de propiedades (.yaml)
//Tiene como objetivo unico, traducir las configuraciones del yaml a algo que Go comprenda y entregarselas al resto de la app

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Asbtraccion de configuraciones
type IConfig interface {
	ServerPort() int
}

// Implementacion concreta de la interfaz
type configImpl struct {

	//Server es un campo que representa la clave 'server' del yaml, y almacena en  los campos campos las claves aninadas dentro del 'server'
	//estamos copiando el estilo anidado del yaml.
	//los struct tags le dicen a la libreria que ocupamos para el parseo, a que clave del yaml corresponde cada campo
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
}

// Funcion que devuelve el port
func (c configImpl) ServerPort() int {
	return c.Server.Port
}

// funcion suelta que lee el y parsea los campos del archivo yaml al struct
// No le pertenece a  configImpl, debido a que esta crea la configuracion desde cero, en el momento que llamamos a esta funcion
// no existe una configuracion (configImpl)
func Load(path string) (IConfig, error) {

	//1. Leemos el archivo yaml (pasandole la path donde se encuenta al metodo ReadFile)
	//   Esto nos devuelve los datos en bytes o un error
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("leyendo config %q: %w", path, err)
	}
	//Crea la configuracion
	var configuration configImpl

	//2. Comienza el parseo, la libreria lee los bytes obtenidos  y los parsea correctamente al struct (rellena el struct con esos datos)
	//   verifica si hay error
	if err := yaml.Unmarshal(data, &configuration); err != nil {
		return nil, fmt.Errorf("parseando configuraciones %q: %w", path, err)
	}

	//Retornamos el objeto IConfig con las claves del yaml cargadas correctamente
	//En Go podemos devolver una implementacion concreta como tipo Interfaz, ya que la impl es de tipo interfaz
	return configuration, nil
}
