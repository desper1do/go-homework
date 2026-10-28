//go:build race

package cart

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// Много горутин одновременно добавляют товары одному пользователю.
// Под -race без блокировок здесь был бы DATA RACE на map.
func TestStorageConcurrentAdd(t *testing.T) {
	t.Parallel()

	const (
		goroutines = 50
		perRoutine = 100
	)

	s := NewStorage()
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perRoutine; j++ {
				s.Add(1, int64(j%5+1), 1)
			}
		}()
	}
	wg.Wait()

	// 5 разных sku, на каждый пришло goroutines*perRoutine/5 добавлений по 1.
	want := []Entry{{1, 1000}, {2, 1000}, {3, 1000}, {4, 1000}, {5, 1000}}
	require.Equal(t, want, s.Get(1))
}
