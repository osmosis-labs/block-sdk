package base

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

var (
	_ MempoolInterface    = (*PriorityNonceMempool[int64])(nil)
	_ sdkmempool.Iterator = (*PriorityNonceIterator[int64])(nil)
)

type (
	NoOpMempool[C comparable] struct{}
)

func NewNoOpMempool[C comparable]() *NoOpMempool[C] {
	mp := &NoOpMempool[C]{}
	return mp
}

// Insert implements MempoolInterface.
func (mp *NoOpMempool[C]) Insert(_ context.Context, _ sdk.Tx) error {
	return nil
}

// Remove implements MempoolInterface.
func (mp *NoOpMempool[C]) Remove(_ sdk.Tx) error {
	return nil
}

// Select implements MempoolInterface.
func (mp *NoOpMempool[C]) Select(_ context.Context, _ [][]byte) sdkmempool.Iterator {
	return nil
}

// CountTx implements MempoolInterface.
func (mp *NoOpMempool[C]) CountTx() int {
	return 0
}

// Contains implements MempoolInterface.
func (mp *NoOpMempool[C]) Contains(_ sdk.Tx) bool {
	return false
}
