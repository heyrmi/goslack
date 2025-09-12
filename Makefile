DB_URL=postgresql://root:secret@localhost:5432/goslack?sslmode=disable

network:
	docker network create goslack-network

postgres:
	docker run --name postgres17 --network goslack-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:17-alpine

createdb:
	docker exec -it postgres17 createdb --username=root --owner=root goslack

dropdb:
	docker exec -it postgres17 dropdb goslack

droptestdb:
	docker exec -it postgres17 dropdb goslack_test

migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migrateup1:
	migrate -path db/migration -database "$(DB_URL)" -verbose up 1

migratedown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down

migratedown1:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)

sqlc:
	sqlc generate

test:
	go test -v -cover -count=1 ./...

test-unit:
	go test -v -cover -count=1 -short ./...

test-integration:
	go test -v -cover -count=1 -run TestIntegrationSuite ./...

test-coverage:
	go test -v -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

test-db:
	go test -v -cover -count=1 ./db/sqlc/...

test-api:
	go test -v -cover -count=1 ./api/...

test-service:
	go test -v -cover -count=1 ./service/...

server:
	go run main.go

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/heyrmi/goslack/db/sqlc Store

swagger:
	swag init -g main.go -o docs/ --parseInternal --parseDependency

swagger-serve: swagger
	@echo "Swagger documentation available at: http://localhost:8080/swagger/index.html"
	make server

swagger-clean:
	rm -rf docs/

.PHONY: network postgres createdb dropdb droptestdb migrateup migratedown migrateup1 migratedown1 new_migration sqlc test server mock swagger swagger-serve swagger-clean
