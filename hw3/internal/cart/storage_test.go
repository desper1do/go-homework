package cart

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

type addOp struct {
	userID, sku int64
	count       uint32
}

func TestStorage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		ops    []addOp
		userID int64
		want   []Entry
	}{
		{name: "unknown user has empty cart", userID: 1, want: []Entry{}},
		{name: "single item", ops: []addOp{{1, 10, 2}}, userID: 1, want: []Entry{{SKU: 10, Count: 2}}},
		{name: "repeated add sums count", ops: []addOp{{1, 10, 2}, {1, 10, 3}}, userID: 1, want: []Entry{{SKU: 10, Count: 5}}},
		{
			name:   "sorted by sku not by insertion",
			ops:    []addOp{{1, 30, 1}, {1, 10, 1}, {1, 20, 1}},
			userID: 1,
			want:   []Entry{{SKU: 10, Count: 1}, {SKU: 20, Count: 1}, {SKU: 30, Count: 1}},
		},
		{name: "other user is isolated", ops: []addOp{{2, 10, 1}}, userID: 1, want: []Entry{}},
		{name: "same sku for two users", ops: []addOp{{1, 10, 1}, {2, 10, 7}}, userID: 2, want: []Entry{{SKU: 10, Count: 7}}},
		{name: "max count is kept", ops: []addOp{{1, 10, math.MaxUint32}}, userID: 1, want: []Entry{{SKU: 10, Count: math.MaxUint32}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := NewStorage()
			for _, op := range tc.ops {
				s.Add(op.userID, op.sku, op.count)
			}

			require.Equal(t, tc.want, s.Get(tc.userID))
		})
	}
}
