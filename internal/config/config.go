package config

//Abstraccion que nos permite extraer configuraciones desde el archivo de propiedades (.yaml)
//Tiene como objetivo unico, traducir las configuraciones del yaml a algo que Go comprenda y entregarselas al resto de la app
//ocupamos la libreria Viper, pero la abstraemos para usar nuestra interfaz en codigo.

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Implementacion concreta de la interfaz, envuelve el Viper
type Config struct {
	vpr *viper.Viper
}

// Cada funcion delega al viper el trabajo. Cada método es la asbtraccion, envoltorio del viper
func (c *Config) GetString(path string) string { return c.vpr.GetString(path) }
func (c *Config) GetInt(path string) int       { return c.vpr.GetInt(path) }
func (c *Config) GetBool(path string) bool     { return c.vpr.GetBool(path) }

// Metodo Load que envuelve la logica de leer el yaml inidicado y devuelve la config lista para consultar
func Load(path string) (*Config, error) {

	//Creamos el viper
	vpr := viper.New()
	//le dice que archivo cargar (buscar)  y por la extension sabe en que formato parsearlo
	//1. verison config base (sin secrets)
	vpr.SetConfigFile(path)
	// ReadInConfig: abre el archivo, PARSEA su texto YAML (lo convierte de caracteres
	// a una estructura) y lo carga en memoria como un árbol anidado listo para consultar.
	if err := vpr.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("leyendo config %q: %w", path, err)
	}
	//2, overlay local (secrets/overrides)
	vpr.SetConfigFile("application-local.yaml")
	//Merge lee el archivo que le inidicamos arriba y lo fusiona encima del arbol que ya teniamos construido
	//al final tiene todo junto, pero de esta forma evitamos exponer secrets en el archivo de propiedades que se sube al repo
	//tenemos uno donde no hay secret y el otro donde hay secret se mergea creando uno solo sin exponer dicho archivo

	//este archivo puede no existir en el codigo por si los secrets nos vienen desde un vault
	if err := vpr.MergeInConfig(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("mezclando config local: %w", err)
	}

	return &Config{vpr: vpr}, nil
}
