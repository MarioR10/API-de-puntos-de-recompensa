#Documento de atajos para tareas comunes/repetitivas
#se ejecuta la tarea con la nomenclatura make <tarea objetivo>

up: #levanta los contenedores de Postgres y redis en segundo plano
	docker compose up -d
down: #mata los contenedores postgres y redis
	docker compose down
ps: #verifica el estado de los contenedores
	docker compose ps
logs: #logs en vivo d
	docker compose logs -f
run: #corre la API
	go run ./cmd/api
build: #compila todo el modulo (verifica que todo el programa comnpile)
	go build ./...
tidy: #ordena dependencias en go.mod /go.sum
	go mod tidy
test: #corre todos los test de la API
	go test ./...
.PHONY: up down ps logs run build tidy test