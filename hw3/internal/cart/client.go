package cart

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "hw2/api/product"
)

// ErrProductNotFound означает, что product-service не знает такой товар.
var ErrProductNotFound = errors.New("product not found")

// ProductClient оборачивает gRPC клиент product-service.
type ProductClient struct {
	client pb.ProductServiceClient
}

func NewProductClient(client pb.ProductServiceClient) *ProductClient {
	return &ProductClient{client: client}
}

// GetProduct возвращает имя и цену товара. NotFound от сервиса превращается в ErrProductNotFound.
func (c *ProductClient) GetProduct(ctx context.Context, sku int64) (name string, price uint32, err error) {
	resp, err := c.client.GetProduct(ctx, &pb.GetProductRequest{Sku: sku})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", 0, fmt.Errorf("get product %d: %w", sku, ErrProductNotFound)
		}
		return "", 0, fmt.Errorf("get product %d: %w", sku, err)
	}

	// По контракту цена приходит как int32, внутри работаем с uint32.
	if resp.GetPrice() < 0 {
		return "", 0, fmt.Errorf("get product %d: negative price %d", sku, resp.GetPrice())
	}
	return resp.GetName(), uint32(resp.GetPrice()), nil
}
