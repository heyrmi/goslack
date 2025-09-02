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

server:
	go run main.go

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/rahulmishra/goslack/db/sqlc Store

.PHONY: network postgres createdb dropdb droptestdb migrateup migratedown migrateup1 migratedown1 new_migration sqlc test server mock
