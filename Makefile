.PHONY: build run-api run-worker swagger docker-build up down clean test

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

run-api:
	go run ./cmd/api/main.go

run-worker:
	go run ./cmd/worker/main.go

swagger:
	go run github.com/swaggo/swag/cmd/swag init -g cmd/api/main.go

docker-build:
	docker build --target api -t gopay-api .
	docker build --target worker -t gopay-worker .

up:
	docker-compose up --build -d

down:
	docker-compose down

clean:
	rm -rf bin/

test:
	go test -v ./...
