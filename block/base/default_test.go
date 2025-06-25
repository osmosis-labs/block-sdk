package base_test

import (
	"context"
	"math/rand"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/stretchr/testify/require"

	"github.com/skip-mev/block-sdk/v2/block/base"
	"github.com/skip-mev/block-sdk/v2/testutils"
)

func TestNewDefaultMempool(t *testing.T) {
	tests := []struct {
		name   string
		maxTxs int
	}{
		{
			name:   "unlimited capacity",
			maxTxs: 0,
		},
		{
			name:   "limited capacity",
			maxTxs: 10,
		},
		{
			name:   "negative capacity",
			maxTxs: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp := base.NewDefaultMempool[int64](tt.maxTxs)
			require.NotNil(t, mp)
			require.Equal(t, 0, mp.CountTx())
		})
	}
}

func TestDefaultMempool_Insert(t *testing.T) {
	ctx := context.Background()
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 3)
	txConfig := testutils.CreateTestEncodingConfig().TxConfig

	t.Run("insert single transaction", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		tx, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(100)))
		require.NoError(t, err)

		err = mp.Insert(ctx, tx)
		require.NoError(t, err)
		require.Equal(t, 1, mp.CountTx())
		require.True(t, mp.Contains(tx))
	})

	t.Run("insert multiple transactions", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)

		for i := 0; i < 3; i++ {
			tx, err := testutils.CreateTx(txConfig, accounts[i], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(int64(100+i))))
			require.NoError(t, err)

			err = mp.Insert(ctx, tx)
			require.NoError(t, err)
			require.Equal(t, i+1, mp.CountTx())
			require.True(t, mp.Contains(tx))
		}
	})

	t.Run("insert with max capacity", func(t *testing.T) {
		maxTx := 2
		mp := base.NewDefaultMempool[int64](maxTx)

		// Insert up to capacity
		var txs []sdk.Tx
		for i := 0; i < maxTx; i++ {
			tx, err := testutils.CreateTx(txConfig, accounts[i], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(int64(100+i))))
			require.NoError(t, err)
			_ = append(txs, tx)

			err = mp.Insert(ctx, tx)
			require.NoError(t, err)
		}
		require.Equal(t, maxTx, mp.CountTx())

		// Try to insert beyond capacity
		overCapacityTx, err := testutils.CreateTx(txConfig, accounts[2], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(300)))
		require.NoError(t, err)

		err = mp.Insert(ctx, overCapacityTx)
		require.ErrorIs(t, err, sdkmempool.ErrMempoolTxMaxCapacity)
		require.Equal(t, maxTx, mp.CountTx())
		require.False(t, mp.Contains(overCapacityTx))
	})

	t.Run("insert with negative max capacity", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](-1)
		tx, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(100)))
		require.NoError(t, err)

		err = mp.Insert(ctx, tx)
		require.NoError(t, err)
		require.Equal(t, 0, mp.CountTx()) // Should not actually insert
		require.False(t, mp.Contains(tx))
	})

	t.Run("insert duplicate transactions", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		tx, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(100)))
		require.NoError(t, err)

		// Insert same transaction multiple times
		err = mp.Insert(ctx, tx)
		require.NoError(t, err)
		err = mp.Insert(ctx, tx)
		require.NoError(t, err)
		err = mp.Insert(ctx, tx)
		require.NoError(t, err)

		// Should have multiple copies
		require.Equal(t, 3, mp.CountTx())
		require.True(t, mp.Contains(tx))
	})
}

func TestDefaultMempool_Remove(t *testing.T) {
	ctx := context.Background()
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 2)
	txConfig := testutils.CreateTestEncodingConfig().TxConfig

	t.Run("remove existing transaction", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		tx1, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(100)))
		require.NoError(t, err)
		tx2, err := testutils.CreateTx(txConfig, accounts[1], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(200)))
		require.NoError(t, err)

		// Insert transactions
		err = mp.Insert(ctx, tx1)
		require.NoError(t, err)
		err = mp.Insert(ctx, tx2)
		require.NoError(t, err)
		require.Equal(t, 2, mp.CountTx())

		// Remove one transaction
		err = mp.Remove(tx1)
		require.NoError(t, err)
		require.Equal(t, 1, mp.CountTx())
		require.False(t, mp.Contains(tx1))
		require.True(t, mp.Contains(tx2))
	})

	t.Run("remove non-existing transaction", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		tx1, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(100)))
		require.NoError(t, err)
		tx2, err := testutils.CreateTx(txConfig, accounts[1], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(200)))
		require.NoError(t, err)

		// Insert one transaction
		err = mp.Insert(ctx, tx1)
		require.NoError(t, err)

		// Try to remove non-existing transaction
		err = mp.Remove(tx2)
		require.NoError(t, err) // Should not error
		require.Equal(t, 1, mp.CountTx())
		require.True(t, mp.Contains(tx1))
	})

	t.Run("remove from empty mempool", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		tx, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(100)))
		require.NoError(t, err)

		err = mp.Remove(tx)
		require.NoError(t, err) // Should not error
		require.Equal(t, 0, mp.CountTx())
	})

	t.Run("remove duplicate transactions", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		tx, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(100)))
		require.NoError(t, err)

		// Insert same transaction multiple times
		err = mp.Insert(ctx, tx)
		require.NoError(t, err)
		err = mp.Insert(ctx, tx)
		require.NoError(t, err)
		require.Equal(t, 2, mp.CountTx())

		// Remove should only remove first occurrence
		err = mp.Remove(tx)
		require.NoError(t, err)
		require.Equal(t, 1, mp.CountTx())
		require.True(t, mp.Contains(tx)) // Still contains the duplicate
	})
}

func TestDefaultMempool_Select(t *testing.T) {
	ctx := context.Background()
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 3)
	txConfig := testutils.CreateTestEncodingConfig().TxConfig

	t.Run("select from empty mempool", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)

		iter := mp.Select(ctx, nil)
		require.Nil(t, iter)
	})

	t.Run("select single transaction", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		tx, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(100)))
		require.NoError(t, err)

		err = mp.Insert(ctx, tx)
		require.NoError(t, err)

		iter := mp.Select(ctx, nil)
		require.NotNil(t, iter)

		// Check first transaction
		require.Equal(t, tx, iter.Tx())

		// Check no more transactions
		next := iter.Next()
		require.Nil(t, next)
	})

	t.Run("select multiple transactions", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		var txs []sdk.Tx

		// Insert transactions
		for i := 0; i < 3; i++ {
			tx, err := testutils.CreateTx(txConfig, accounts[i], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(int64(100+i))))
			require.NoError(t, err)
			txs = append(txs, tx)

			err = mp.Insert(ctx, tx)
			require.NoError(t, err)
		}

		// Collect all transactions via iterator
		var collectedTxs []sdk.Tx
		for iter := mp.Select(ctx, nil); iter != nil; iter = iter.Next() {
			collectedTxs = append(collectedTxs, iter.Tx())
		}

		// Should have all transactions in insertion order
		require.Equal(t, len(txs), len(collectedTxs))
		for i, tx := range txs {
			require.Equal(t, tx, collectedTxs[i])
		}
	})

	t.Run("select with ignored parameters", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		tx, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(100)))
		require.NoError(t, err)

		err = mp.Insert(ctx, tx)
		require.NoError(t, err)

		// The txs parameter is ignored in DefaultMempool
		iter := mp.Select(ctx, [][]byte{{1, 2, 3}})
		require.NotNil(t, iter)
		require.Equal(t, tx, iter.Tx())
	})
}

func TestDefaultMempool_CountTx(t *testing.T) {
	ctx := context.Background()
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 5)
	txConfig := testutils.CreateTestEncodingConfig().TxConfig

	t.Run("count starts at zero", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		require.Equal(t, 0, mp.CountTx())
	})

	t.Run("count increases with inserts", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)

		for i := 1; i <= 5; i++ {
			tx, err := testutils.CreateTx(txConfig, accounts[i-1], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(int64(100+i))))
			require.NoError(t, err)

			err = mp.Insert(ctx, tx)
			require.NoError(t, err)
			require.Equal(t, i, mp.CountTx())
		}
	})

	t.Run("count decreases with removes", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		var txs []sdk.Tx

		// Insert transactions
		for i := 0; i < 3; i++ {
			tx, err := testutils.CreateTx(txConfig, accounts[i], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(int64(100+i))))
			require.NoError(t, err)
			txs = append(txs, tx)

			err = mp.Insert(ctx, tx)
			require.NoError(t, err)
		}
		require.Equal(t, 3, mp.CountTx())

		// Remove transactions
		for i, tx := range txs {
			err := mp.Remove(tx)
			require.NoError(t, err)
			require.Equal(t, 2-i, mp.CountTx())
		}
	})
}

func TestDefaultMempool_Contains(t *testing.T) {
	ctx := context.Background()
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 3)
	txConfig := testutils.CreateTestEncodingConfig().TxConfig

	t.Run("contains returns false for empty mempool", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		tx, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(100)))
		require.NoError(t, err)

		require.False(t, mp.Contains(tx))
	})

	t.Run("contains returns true for inserted transaction", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		tx, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(100)))
		require.NoError(t, err)

		err = mp.Insert(ctx, tx)
		require.NoError(t, err)
		require.True(t, mp.Contains(tx))
	})

	t.Run("contains returns false for removed transaction", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		tx, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(100)))
		require.NoError(t, err)

		err = mp.Insert(ctx, tx)
		require.NoError(t, err)
		require.True(t, mp.Contains(tx))

		err = mp.Remove(tx)
		require.NoError(t, err)
		require.False(t, mp.Contains(tx))
	})

	t.Run("contains works with multiple transactions", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		tx1, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(100)))
		require.NoError(t, err)
		tx2, err := testutils.CreateTx(txConfig, accounts[1], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(200)))
		require.NoError(t, err)
		tx3, err := testutils.CreateTx(txConfig, accounts[2], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(300)))
		require.NoError(t, err)

		err = mp.Insert(ctx, tx1)
		require.NoError(t, err)
		err = mp.Insert(ctx, tx3)
		require.NoError(t, err)

		require.True(t, mp.Contains(tx1))
		require.False(t, mp.Contains(tx2))
		require.True(t, mp.Contains(tx3))
	})
}

func TestDefaultIterator(t *testing.T) {
	ctx := context.Background()
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 4)
	txConfig := testutils.CreateTestEncodingConfig().TxConfig

	t.Run("iterator next returns nil when at end", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		tx, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(100)))
		require.NoError(t, err)

		err = mp.Insert(ctx, tx)
		require.NoError(t, err)

		iter := mp.Select(ctx, nil)
		require.NotNil(t, iter)

		// First transaction
		require.Equal(t, tx, iter.Tx())

		// Next should return nil (end of iterator)
		next := iter.Next()
		require.Nil(t, next)
	})

	t.Run("iterator maintains position correctly", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		var txs []sdk.Tx

		for i := 0; i < 3; i++ {
			tx, err := testutils.CreateTx(txConfig, accounts[i], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(int64(100+i))))
			require.NoError(t, err)
			txs = append(txs, tx)

			err = mp.Insert(ctx, tx)
			require.NoError(t, err)
		}

		iter := mp.Select(ctx, nil)
		require.NotNil(t, iter)

		// Manually iterate and check each position
		for i, expectedTx := range txs {
			require.Equal(t, expectedTx, iter.Tx(), "iteration %d", i)
			if i < len(txs)-1 {
				iter = iter.Next()
				require.NotNil(t, iter, "iteration %d", i)
			}
		}

		// Should be at end now
		next := iter.Next()
		require.Nil(t, next)
	})

	t.Run("iterator returns correct transaction at each position", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0)
		var txs []sdk.Tx

		for i := 0; i < 4; i++ {
			tx, err := testutils.CreateTx(txConfig, accounts[i], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(int64(100+i))))
			require.NoError(t, err)
			txs = append(txs, tx)

			err = mp.Insert(ctx, tx)
			require.NoError(t, err)
		}

		iter := mp.Select(ctx, nil)
		for i, expectedTx := range txs {
			require.NotNil(t, iter, "position %d should have iterator", i)
			require.Equal(t, expectedTx, iter.Tx(), "position %d should have correct tx", i)
			iter = iter.Next()
		}
		require.Nil(t, iter, "should be at end after all transactions")
	})
}

func TestDefaultMempool_Integration(t *testing.T) {
	// Integration test with more complex scenarios
	ctx := context.Background()
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 5)
	txConfig := testutils.CreateTestEncodingConfig().TxConfig

	t.Run("comprehensive integration test", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](10)

		// Create various transactions
		var txs []sdk.Tx
		for i := 0; i < 5; i++ {
			tx, err := testutils.CreateTx(txConfig, accounts[i], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(int64(100+i*10))))
			require.NoError(t, err)
			txs = append(txs, tx)
		}

		// Test basic operations
		for i, tx := range txs {
			err := mp.Insert(ctx, tx)
			require.NoError(t, err)
			require.Equal(t, i+1, mp.CountTx())
			require.True(t, mp.Contains(tx))
		}

		// Test iterator returns all transactions
		var collectedTxs []sdk.Tx
		for iter := mp.Select(ctx, nil); iter != nil; iter = iter.Next() {
			collectedTxs = append(collectedTxs, iter.Tx())
		}
		require.Len(t, collectedTxs, 5)

		// Test removal
		err := mp.Remove(txs[2]) // Remove middle transaction
		require.NoError(t, err)
		require.Equal(t, 4, mp.CountTx())
		require.False(t, mp.Contains(txs[2]))

		// Verify other transactions still exist
		for i, tx := range txs {
			if i != 2 {
				require.True(t, mp.Contains(tx))
			}
		}

		// Test iterator after removal
		collectedTxs = nil
		for iter := mp.Select(ctx, nil); iter != nil; iter = iter.Next() {
			collectedTxs = append(collectedTxs, iter.Tx())
		}
		require.Len(t, collectedTxs, 4)
	})
}

// TestDefaultMempool_ImplementsInterface verifies that DefaultMempool implements MempoolInterface
func TestDefaultMempool_ImplementsInterface(t *testing.T) {
	var _ base.MempoolInterface = (*base.DefaultMempool[int64])(nil)
	// Test passes if compilation succeeds
}

// Benchmark tests
func BenchmarkDefaultMempool_Insert(b *testing.B) {
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 1000)
	txConfig := testutils.CreateTestEncodingConfig().TxConfig
	ctx := context.Background()
	mp := base.NewDefaultMempool[int64](0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, err := testutils.CreateTx(txConfig, accounts[i%1000], uint64(i), 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(int64(100+i))))
		if err != nil {
			b.Fatal(err)
		}
		err = mp.Insert(ctx, tx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDefaultMempool_Contains(b *testing.B) {
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 1000)
	txConfig := testutils.CreateTestEncodingConfig().TxConfig
	ctx := context.Background()
	mp := base.NewDefaultMempool[int64](0)

	// Pre-populate mempool
	var txs []sdk.Tx
	for i := 0; i < 1000; i++ {
		tx, err := testutils.CreateTx(txConfig, accounts[i], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(int64(100+i))))
		if err != nil {
			b.Fatal(err)
		}
		err = mp.Insert(ctx, tx)
		if err != nil {
			b.Fatal(err)
		}
		txs = append(txs, tx)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		found := mp.Contains(txs[i%1000])
		if !found {
			b.Fatal("transaction not found")
		}
	}
}
