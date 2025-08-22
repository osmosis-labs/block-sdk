package base_test

import (
	"context"
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/stretchr/testify/require"

	signer_extraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/v2/block/base"
	"github.com/skip-mev/block-sdk/v2/block/utils"
	"github.com/skip-mev/block-sdk/v2/testutils"
)

func TestDefaultMempool_Insert(t *testing.T) {
	ctx := context.Background()
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 3)
	txConfig := testutils.CreateTestEncodingConfig().TxConfig
	signerExtractor := signer_extraction.NewDefaultAdapter()

	t.Run("insert single transaction", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0, signerExtractor)
		tx, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(100)))
		require.NoError(t, err)

		err = mp.Insert(ctx, tx)
		require.NoError(t, err)
		require.Equal(t, 1, mp.CountTx())
		require.True(t, mp.Contains(tx))
	})

	t.Run("insert multiple transactions", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0, signerExtractor)

		for i := 0; i < 3; i++ {
			tx, err := testutils.CreateTx(txConfig, accounts[i], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(int64(100+i))))
			require.NoError(t, err)

			err = mp.Insert(ctx, tx)
			require.NoError(t, err)
			require.Equal(t, i+1, mp.CountTx())
			require.True(t, mp.Contains(tx))
		}
	})

	t.Run("insert duplicate transaction (same sender:nonce) should be no-op", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0, signerExtractor)

		// Insert first transaction
		tx1, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(100)))
		require.NoError(t, err)

		err = mp.Insert(ctx, tx1)
		require.NoError(t, err)
		require.Equal(t, 1, mp.CountTx())
		require.True(t, mp.Contains(tx1))

		// Insert second transaction with same sender:nonce (should be no-op)
		tx2, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(200)))
		require.NoError(t, err)

		err = mp.Insert(ctx, tx2)
		require.NoError(t, err)
		require.Equal(t, 1, mp.CountTx())  // Count should remain 1
		require.True(t, mp.Contains(tx1))  // Original transaction should still be there
		require.False(t, mp.Contains(tx2)) // New transaction should not be added
	})

	t.Run("insert with max capacity", func(t *testing.T) {
		maxTx := 2
		mp := base.NewDefaultMempool[int64](maxTx, signerExtractor)

		// Insert up to capacity
		for i := 0; i < maxTx; i++ {
			tx, err := testutils.CreateTx(txConfig, accounts[i], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(int64(100+i))))
			require.NoError(t, err)

			err = mp.Insert(ctx, tx)
			require.NoError(t, err)
		}
		require.Equal(t, maxTx, mp.CountTx())

		// Try to insert beyond capacity (different sender:nonce)
		overCapacityTx, err := testutils.CreateTx(txConfig, accounts[2], 1, 0, nil, sdk.NewCoin("stake", math.NewInt(300)))
		require.NoError(t, err)

		err = mp.Insert(ctx, overCapacityTx)
		require.ErrorIs(t, err, sdkmempool.ErrMempoolTxMaxCapacity)
		require.Equal(t, maxTx, mp.CountTx())
		require.False(t, mp.Contains(overCapacityTx))
	})

	t.Run("insert duplicate within capacity should not trigger capacity error", func(t *testing.T) {
		maxTx := 1
		mp := base.NewDefaultMempool[int64](maxTx, signerExtractor)

		// Insert first transaction (fills capacity)
		tx1, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(100)))
		require.NoError(t, err)

		err = mp.Insert(ctx, tx1)
		require.NoError(t, err)
		require.Equal(t, 1, mp.CountTx())

		// Insert duplicate (should be no-op, not trigger capacity error)
		tx2, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(200)))
		require.NoError(t, err)

		err = mp.Insert(ctx, tx2)
		require.NoError(t, err) // Should NOT get capacity error
		require.Equal(t, 1, mp.CountTx())
		require.True(t, mp.Contains(tx1))  // Original transaction should remain
		require.False(t, mp.Contains(tx2)) // Duplicate should not be added
	})
}

func TestDefaultMempool_Remove(t *testing.T) {
	ctx := context.Background()
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 3)
	txConfig := testutils.CreateTestEncodingConfig().TxConfig
	signerExtractor := signer_extraction.NewDefaultAdapter()

	t.Run("remove existing transaction", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0, signerExtractor)
		tx1, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(100)))
		require.NoError(t, err)
		tx2, err := testutils.CreateTx(txConfig, accounts[1], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(200)))
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
		mp := base.NewDefaultMempool[int64](0, signerExtractor)
		tx1, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(100)))
		require.NoError(t, err)
		tx2, err := testutils.CreateTx(txConfig, accounts[1], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(200)))
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
}

func TestDefaultMempool_Select(t *testing.T) {
	ctx := context.Background()
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 3)
	txConfig := testutils.CreateTestEncodingConfig().TxConfig
	signerExtractor := signer_extraction.NewDefaultAdapter()

	t.Run("select from empty mempool", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0, signerExtractor)

		iter := mp.Select(ctx, nil)
		require.Nil(t, iter)
	})

	t.Run("select maintains FIFO order", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](0, signerExtractor)
		var txs []sdk.Tx

		// Insert transactions in order
		for i := 0; i < 3; i++ {
			tx, err := testutils.CreateTx(txConfig, accounts[i], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(int64(100+i))))
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

		// Should maintain FIFO order
		require.Equal(t, len(txs), len(collectedTxs))
		for i, tx := range txs {
			require.Equal(t, tx, collectedTxs[i])
		}
	})
}

func TestDefaultMempool_Integration(t *testing.T) {
	ctx := context.Background()
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 5)
	txConfig := testutils.CreateTestEncodingConfig().TxConfig
	signerExtractor := signer_extraction.NewDefaultAdapter()

	t.Run("comprehensive integration test with duplicates", func(t *testing.T) {
		mp := base.NewDefaultMempool[int64](10, signerExtractor)

		// Insert initial transactions
		var txs []sdk.Tx
		for i := 0; i < 3; i++ {
			tx, err := testutils.CreateTx(txConfig, accounts[i], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(int64(100+i*10))))
			require.NoError(t, err)
			txs = append(txs, tx)

			err = mp.Insert(ctx, tx)
			require.NoError(t, err)
		}
		require.Equal(t, 3, mp.CountTx())

		// Attempt to insert duplicate transaction (should be no-op)
		duplicateTx, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(500)))
		require.NoError(t, err)

		err = mp.Insert(ctx, duplicateTx)
		require.NoError(t, err)
		require.Equal(t, 3, mp.CountTx())          // Count should remain the same
		require.True(t, mp.Contains(txs[0]))       // Original transaction should still be there
		require.False(t, mp.Contains(duplicateTx)) // Duplicate should not be added

		// Verify FIFO order is maintained (no changes)
		var collectedTxs []sdk.Tx
		for iter := mp.Select(ctx, nil); iter != nil; iter = iter.Next() {
			collectedTxs = append(collectedTxs, iter.Tx())
		}
		require.Len(t, collectedTxs, 3)
		// Transactions should remain in original order
		require.Equal(t, txs[0], collectedTxs[0])
		require.Equal(t, txs[1], collectedTxs[1])
		require.Equal(t, txs[2], collectedTxs[2])
	})
}

// TestDefaultMempool_ContainsWithRedecodedTransactions tests that Contains() works correctly
// with transactions that have been re-decoded (different object, same content)
func TestDefaultMempool_ContainsWithRedecodedTransactions(t *testing.T) {
	ctx := context.Background()
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 1)
	txConfig := testutils.CreateTestEncodingConfig().TxConfig
	signerExtractor := signer_extraction.NewDefaultAdapter()
	mp := base.NewDefaultMempool[int64](5, signerExtractor)

	// Create and insert original transaction
	originalTx, err := testutils.CreateTx(txConfig, accounts[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(100)))
	require.NoError(t, err)
	err = mp.Insert(ctx, originalTx)
	require.NoError(t, err)

	// Encode and decode to create a new object with same content
	encodedTxs, err := utils.GetEncodedTxs(txConfig.TxEncoder(), []sdk.Tx{originalTx})
	require.NoError(t, err)
	decodedTxs, err := utils.GetDecodedTxs(txConfig.TxDecoder(), encodedTxs)
	require.NoError(t, err)
	redecodedTx := decodedTxs[0]

	// Verify they are different objects but same content
	require.False(t, originalTx == redecodedTx, "Should be different objects")

	// Contains() should return true for both original and re-decoded transaction
	require.True(t, mp.Contains(originalTx), "Should contain original transaction")
	require.True(t, mp.Contains(redecodedTx), "Should contain re-decoded transaction")
}

// TestDefaultMempool_ImplementsInterface verifies that DefaultMempool implements MempoolInterface
func TestDefaultMempool_ImplementsInterface(t *testing.T) {
	signerExtractor := signer_extraction.NewDefaultAdapter()
	var _ base.MempoolInterface = (*base.DefaultMempool[int64])(nil)

	// Test that we can create the mempool
	mp := base.NewDefaultMempool[int64](100, signerExtractor)
	require.NotNil(t, mp)
}
