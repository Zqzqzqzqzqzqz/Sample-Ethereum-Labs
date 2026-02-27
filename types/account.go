package types

import (
	ecc "simple_eth/crypt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Account describes an address with a balance; key material is managed separately.
type Account struct {
	Address common.Address
	Balance float64
	Nonce   int64
}

// DeriveAddress creates a deterministic address from a public key.
func DeriveAddress(pub *ecc.Point) common.Address {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return common.Address{}
	}
	x := padScalar(pub.X.Bytes())
	y := padScalar(pub.Y.Bytes())
	pubBytes := append(x, y...)
	hash := crypto.Keccak256(pubBytes)
	return common.BytesToAddress(hash[12:])
}

func padScalar(b []byte) []byte {
	if len(b) >= 32 {
		return b
	}
	padding := make([]byte, 32-len(b))
	return append(padding, b...)
}
