package signerextraction

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
)

type SignerData struct {
	Signer   sdk.AccAddress
	Sequence uint64
}

// NewSignerData returns a new SignerData instance.
func NewSignerData(signer sdk.AccAddress, sequence uint64) SignerData {
	return SignerData{
		Signer:   signer,
		Sequence: sequence,
	}
}

// String implements the fmt.Stringer interface.
func (s SignerData) String() string {
	return fmt.Sprintf("SignerData{Signer: %s, Sequence: %d}", s.Signer, s.Sequence)
}

// SignerExtractionAdapter is an interface used to determine how the signers of a transaction should be extracted
// from the transaction.
type Adapter interface {
	GetSigners(sdk.Tx) ([]SignerData, error)
}

var _ Adapter = DefaultAdapter{}

// DefaultSignerExtractionAdapter is the default implementation of SignerExtractionAdapter. It extracts the signers
// from a cosmos-sdk tx via GetSignaturesV2.
type DefaultAdapter struct{}

func NewDefaultAdapter() DefaultAdapter {
	return DefaultAdapter{}
}

func (DefaultAdapter) GetSigners(tx sdk.Tx) ([]SignerData, error) {
	sigTx, ok := tx.(signing.SigVerifiableTx)
	if !ok {
		return nil, fmt.Errorf("tx of type %T does not implement SigVerifiableTx", tx)
	}

	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return nil, err
	}

	signers := make([]SignerData, len(sigs))
	for i, sig := range sigs {
		signers[i] = SignerData{
			Signer:   sig.PubKey.Address().Bytes(),
			Sequence: sig.Sequence,
		}
	}

	return signers, nil
}
