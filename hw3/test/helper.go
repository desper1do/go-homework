//go:build api

// Пакет api содержит интеграционные тесты против уже запущенных сервисов (make run-all).
// Тесты знают только сгенерированный gRPC контракт product-service и HTTP API cart.
package api

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "hw2/api/product"
)

const (
	productAddr = "localhost:50051"
	cartURL     = "http://localhost:8080"

	requestTimeout = 5 * time.Second
)

var userSeq atomic.Int64

// newUserID возвращает user_id, которого не было ни в этом, ни в прошлых запусках,
// поэтому тест не зависит от корзин, накопленных в живом сервисе.
func newUserID() int64 {
	return time.Now().UnixNano() + userSeq.Add(1)
}

// dialProduct подключается к product-service и закрывает соединение по окончании теста.
func dialProduct(t *testing.T) pb.ProductServiceClient {
	t.Helper()

	conn, err := grpc.NewClient(productAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return pb.NewProductServiceClient(conn)
}

func newContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	t.Cleanup(cancel)
	return ctx
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: requestTimeout}
}
