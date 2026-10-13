package product

import (
	"context"
	"math"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "hw2/api/product"
)

const (
	defaultLimit = 10
	maxLimit     = 100
)

// Server реализует gRPC сервис product-service.
type Server struct {
	pb.UnimplementedProductServiceServer
	storage *Storage
}

func NewServer(storage *Storage) *Server {
	return &Server{storage: storage}
}

func (s *Server) GetProduct(_ context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	if req.GetSku() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "sku must be greater than zero")
	}

	p, ok := s.storage.Get(req.GetSku())
	if !ok {
		return nil, status.Error(codes.NotFound, "product not found")
	}

	return toResponse(req.GetSku(), p)
}

// ListProducts возвращает страницу каталога. Страницы нумеруются с единицы,
// page=0 означает первую страницу, limit=0 означает defaultLimit.
func (s *Server) ListProducts(_ context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	page, limit := int64(req.GetPage()), int64(req.GetLimit())
	if page < 0 {
		return nil, status.Error(codes.InvalidArgument, "page must not be negative")
	}
	if limit < 0 || limit > maxLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be between 0 and %d", maxLimit)
	}
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = defaultLimit
	}

	// page ограничен int32, а limit значением maxLimit, поэтому offset не переполняет int64.
	items := s.storage.List(int((page-1)*limit), int(limit))
	resp := &pb.ListProductsResponse{
		Products: make([]*pb.GetProductResponse, 0, len(items)),
		Total:    int32(s.storage.Total()),
	}
	for _, item := range items {
		p, err := toResponse(item.SKU, item.Product)
		if err != nil {
			return nil, err
		}
		resp.Products = append(resp.Products, p)
	}
	return resp, nil
}

// toResponse переводит товар в ответ gRPC. По контракту цена в ответе int32,
// а внутри uint32, поэтому проверяем переполнение.
func toResponse(sku int64, p Product) (*pb.GetProductResponse, error) {
	if p.Price > math.MaxInt32 {
		return nil, status.Error(codes.Internal, "product price is too large")
	}
	return &pb.GetProductResponse{Sku: sku, Name: p.Name, Price: int32(p.Price)}, nil
}
