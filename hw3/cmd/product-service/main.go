package main

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "hw2/api/product"
	"hw2/internal/product"
)

const (
	addr         = ":50051"
	productsFile = "data/products.json"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	storage, err := product.Load(productsFile)
	if err != nil {
		return fmt.Errorf("load products: %w", err)
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	return serve(lis, storage)
}

// serve запускает gRPC сервер на переданном listener. Так в тесте можно взять свободный порт.
func serve(lis net.Listener, storage *product.Storage) error {
	srv := grpc.NewServer()
	pb.RegisterProductServiceServer(srv, product.NewServer(storage))

	log.Printf("product-service listens on %s", lis.Addr())
	if err := srv.Serve(lis); err != nil {
		return fmt.Errorf("serve grpc: %w", err)
	}
	return nil
}
