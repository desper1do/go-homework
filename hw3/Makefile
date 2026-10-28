.PHONY: generate run-all test test-race test-api bench

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

# Юнит тесты, тесты с тегами api и race в обычный прогон не попадают.
test:
	go test ./...

test-race:
	go test -race ./...

# Нужны запущенные сервисы (make run-all). -count=1 отключает кэш результатов,
# иначе тест против живого сервиса вернет старый ответ.
test-api:
	go test -v -count=1 -tags=api ./test/

# Бенчмарки с аллокациями, -run='^$' чтобы заодно не гонять обычные тесты.
bench:
	go test -run='^$$' -bench=. -benchmem ./...
