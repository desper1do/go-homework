.PHONY: generate run-all

generate:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/product/product.proto

# product-service запускается в фоне, cart в основном процессе.
# Фоновый процесс игнорирует Ctrl+C, поэтому останавливаем его через trap по его pid.
run-all:
	go build -o bin/product-service ./cmd/product-service
	go build -o bin/cart ./cmd/cart
	./bin/product-service & pid=$$!; trap 'kill $$pid' INT TERM EXIT; ./bin/cart
