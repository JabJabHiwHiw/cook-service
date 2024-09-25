generate:
	protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/cook.proto

compose-up:
	docker-compose up -d

compose-down:
	docker-compose down

dev:
	make generate
	go run main.go