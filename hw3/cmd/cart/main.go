package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "hw2/api/product"
	"hw2/internal/cart"
)

const (
	addr           = ":8080"
	productService = "localhost:50051"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	return serve(lis, productService)
}

// serve запускает HTTP сервер cart на переданном listener, productAddr это адрес product-service.
func serve(lis net.Listener, productAddr string) error {
	// NewClient не подключается сразу, а делает это при первом вызове,
	// поэтому порядок запуска сервисов не важен.
	conn, err := grpc.NewClient(productAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("create product-service client: %w", err)
	}
	defer conn.Close()

	products := cart.NewProductClient(pb.NewProductServiceClient(conn))
	handler := cart.NewHandler(cart.NewStorage(), products)

	log.Printf("cart listens on %s", lis.Addr())
	if err := http.Serve(lis, handler.Routes()); err != nil {
		return fmt.Errorf("serve http: %w", err)
	}
	return nil
}
