package base

import (
	"container/list"
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"

	signer_extraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
)

var (
	_ MempoolInterface    = (*DefaultMempool[int64])(nil)
	_ sdkmempool.Iterator = (*DefaultIterator)(nil)
)

// DefaultMempool implements a FIFO mempool with duplicate detection based on sender:nonce.
// It uses a linked list for FIFO ordering and a map for O(1) duplicate detection.
type DefaultMempool[C comparable] struct {
	txs             *list.List                // Linked list for FIFO ordering
	seen            map[string]*list.Element  // Map from sender:nonce to list element
	MaxTx           int                       // Maximum number of transactions
	signerExtractor signer_extraction.Adapter // For extracting signer info
}

// NewDefaultMempool creates a new FIFO mempool with duplicate detection
func NewDefaultMempool[C comparable](maxTxs int, signerExtractor signer_extraction.Adapter) *DefaultMempool[C] {
	return &DefaultMempool[C]{
		txs:             list.New(),
		seen:            make(map[string]*list.Element),
		MaxTx:           maxTxs,
		signerExtractor: signerExtractor,
	}
}

// getTxKey creates a unique key from sender:nonce combination
func (mp *DefaultMempool[C]) getTxKey(tx sdk.Tx) (string, error) {
	signers, err := mp.signerExtractor.GetSigners(tx)
	if err != nil {
		return "", err
	}
	if len(signers) == 0 {
		return "", fmt.Errorf("tx must have at least one signer")
	}

	sig := signers[0]
	nonce := sig.Sequence
	sender := sig.Signer.String()

	return fmt.Sprintf("%s:%d", sender, nonce), nil
}

// Insert implements MempoolInterface.
func (mp *DefaultMempool[C]) Insert(_ context.Context, tx sdk.Tx) error {
	if mp.MaxTx < 0 {
		return nil // No-op if MaxTx is negative
	}

	key, err := mp.getTxKey(tx)
	if err != nil {
		return fmt.Errorf("failed to get tx key for insertion: %w", err)
	}

	// Check if this tx already exists
	if _, exists := mp.seen[key]; exists {
		// No-op: transaction with same sender:nonce already exists
		return nil
	}

	// Check capacity before adding new transaction
	if mp.MaxTx > 0 && mp.CountTx() >= mp.MaxTx {
		return sdkmempool.ErrMempoolTxMaxCapacity
	}

	// Add new transaction
	element := mp.txs.PushBack(tx)
	mp.seen[key] = element

	return nil
}

// Remove implements MempoolInterface.
func (mp *DefaultMempool[C]) Remove(tx sdk.Tx) error {
	key, err := mp.getTxKey(tx)
	if err != nil {
		return fmt.Errorf("failed to get tx key for removal: %w", err)
	}

	// Remove by key if found, error if not found
	if element, exists := mp.seen[key]; exists {
		mp.txs.Remove(element)
		delete(mp.seen, key)

		return nil
	}

	return sdkmempool.ErrTxNotFound
}

// Select implements MempoolInterface.
func (mp *DefaultMempool[C]) Select(_ context.Context, _ [][]byte) sdkmempool.Iterator {
	if mp.txs.Len() == 0 {
		return nil
	}

	return &DefaultIterator{
		current: mp.txs.Front(),
	}
}

// CountTx implements MempoolInterface.
func (mp *DefaultMempool[C]) CountTx() int {
	return mp.txs.Len()
}

// Contains implements MempoolInterface.
func (mp *DefaultMempool[C]) Contains(tx sdk.Tx) bool {
	key, err := mp.getTxKey(tx)
	if err != nil {
		return false
	}

	// Check if we have this sender:nonce combination
	_, exists := mp.seen[key]
	return exists
}

// DefaultIterator implements sdkmempool.Iterator for FIFO mempool
type DefaultIterator struct {
	current *list.Element
}

// Next implements sdkmempool.Iterator
func (i *DefaultIterator) Next() sdkmempool.Iterator {
	if i.current == nil {
		return nil
	}

	i.current = i.current.Next()
	if i.current == nil {
		return nil
	}

	return i
}

// Tx implements sdkmempool.Iterator
func (i *DefaultIterator) Tx() sdk.Tx {
	if i.current == nil {
		return nil
	}

	return i.current.Value.(sdk.Tx)
}
