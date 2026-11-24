package database

import (
	"strings"
	"testing"
)

func TestConfigDSN_ValidateDatabaseParameters(t *testing.T) {

	test := []struct {
		name       string
		dsn        ConfigDSN
		wantErr    bool
		wantFields []string
	}{
		{
			name: "valid parameters",
			dsn: ConfigDSN{
				Host:     "localhost",
				Port:     5432,
				User:     "prueba",
				Password: "prueba",
				DBName:   "prueba base de datos",
				SSLMode:  "disable",
			},
			wantErr:    false,
			wantFields: []string{},
		},
		{
			name: "empty host",
			dsn: ConfigDSN{
				Host:     "",
				Port:     5432,
				User:     "prueba",
				Password: "prueba",
				DBName:   "prueba base de datos",
				SSLMode:  "disable",
			},
			wantErr:    true,
			wantFields: []string{"host"},
		},
		{
			name: "host only with spaces",
			dsn: ConfigDSN{
				Host:     "    ",
				Port:     5432,
				User:     "prueba",
				Password: "prueba",
				DBName:   "prueba base de datos",
				SSLMode:  "disable",
			},
			wantErr:    true,
			wantFields: []string{"host"},
		},
		{
			name: "Invalid port equal to zero",
			dsn: ConfigDSN{
				Host:     "localhost",
				Port:     0,
				User:     "prueba",
				Password: "prueba",
				DBName:   "prueba base de datos",
				SSLMode:  "disable",
			},
			wantErr:    true,
			wantFields: []string{"port"},
		},
		{
			name: "Invalid port equal to 70000",
			dsn: ConfigDSN{
				Host:     "localhost",
				Port:     70000,
				User:     "prueba",
				Password: "prueba",
				DBName:   "prueba base de datos",
				SSLMode:  "disable",
			},
			wantErr:    true,
			wantFields: []string{"port"},
		},
		{
			name: "valid password",
			dsn: ConfigDSN{
				Host:     "localhost",
				Port:     5432,
				User:     "prueba",
				Password: "prueba   ",
				DBName:   "prueba base de datos",
				SSLMode:  "disable",
			},
			wantErr:    false,
			wantFields: []string{},
		},
		{
			name: "empty host and invalid port",
			dsn: ConfigDSN{
				Host:     "",
				Port:     70000,
				User:     "prueba",
				Password: "prueba",
				DBName:   "prueba base de datos",
				SSLMode:  "disable",
			},
			wantErr:    true,
			wantFields: []string{"port", "host"},
		},
	}
	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {

			//Validaciones
			err := tt.dsn.ValidateDatabaseParameters()

			huboError := err != nil
			if tt.wantErr != huboError {
				t.Fatalf("obtuvimos %t, y se esperaba %t", huboError, tt.wantErr)
			}

			if err != nil {

					for _, parametro := range tt.wantFields {

						if strings.Contains(err.Error(), parametro) != true {
							t.Fatalf(" Se esperaba %s", parametro)
						}
					}

			}

		})
	}
}
