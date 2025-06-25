package base

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

var (
	_ MempoolInterface    = (*DefaultMempool[int64])(nil)
	_ sdkmempool.Iterator = (*DefaultIterator)(nil)
)

// DefaultMempool implements a simple mempool that stores all transactions
type DefaultMempool[C comparable] struct {
	txs   []sdk.Tx
	MaxTx int
}

// NewDefaultMempool creates a new DefaultMempool
func NewDefaultMempool[C comparable](maxTxs int) *DefaultMempool[C] {
	return &DefaultMempool[C]{
		txs:   make([]sdk.Tx, 0),
		MaxTx: maxTxs,
	}
}

// Insert implements MempoolInterface.
func (mp *DefaultMempool[C]) Insert(_ context.Context, tx sdk.Tx) error {
	if mp.MaxTx > 0 && mp.CountTx() >= mp.MaxTx {
		return sdkmempool.ErrMempoolTxMaxCapacity
	} else if mp.MaxTx < 0 {
		return nil
	}
	mp.txs = append(mp.txs, tx)
	return nil
}

// Remove implements MempoolInterface.
func (mp *DefaultMempool[C]) Remove(tx sdk.Tx) error {
	for i, t := range mp.txs {
		if t == tx {
			mp.txs = append(mp.txs[:i], mp.txs[i+1:]...)
			return nil
		}
	}
	return nil
}

// Select implements MempoolInterface.
func (mp *DefaultMempool[C]) Select(_ context.Context, _ [][]byte) sdkmempool.Iterator {
	if len(mp.txs) == 0 {
		return nil
	}

	return &DefaultIterator{
		txs:  mp.txs,
		curr: 0,
	}
}

// CountTx implements MempoolInterface.
func (mp *DefaultMempool[C]) CountTx() int {
	return len(mp.txs)
}

// Contains implements MempoolInterface.
func (mp *DefaultMempool[C]) Contains(tx sdk.Tx) bool {
	for _, t := range mp.txs {
		if t == tx {
			return true
		}
	}
	return false
}

// DefaultIterator implements sdkmempool.Iterator
type DefaultIterator struct {
	txs  []sdk.Tx
	curr int
}

// Next implements sdkmempool.Iterator
func (i *DefaultIterator) Next() sdkmempool.Iterator {
	if i.curr >= len(i.txs)-1 {
		return nil
	}
	i.curr++
	return i
}

// Tx implements sdkmempool.Iterator
func (i *DefaultIterator) Tx() sdk.Tx {
	return i.txs[i.curr]
}
