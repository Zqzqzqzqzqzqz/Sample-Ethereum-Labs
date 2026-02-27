package wallet

import (
    "fmt"
    "math/big"

    "github.com/ethereum/go-ethereum/common"
    ecc "simple_eth/crypt"
    "simple_eth/types"
)

// KeyStore keeps an in-memory mapping of address to key pairs.
type KeyStore struct {
    keys map[common.Address]*keyPair
}

type keyPair struct {
    priv *big.Int
    pub  *ecc.Point
}

// NewKeyStore creates an empty keystore.
func NewKeyStore() *KeyStore {
    return &KeyStore{keys: make(map[common.Address]*keyPair)}
}

// CreateAccount generates keys, stores them, and returns an account with balance preset.
func (ks *KeyStore) CreateAccount(balance float64) (*types.Account, error) {
    if ks == nil {
        return nil, fmt.Errorf("keystore ????")
    }
    priv, err := ecc.NewPrivateKey()
    if err != nil {
        return nil, fmt.Errorf("??????: %w", err)
    }
    pub := ecc.GeneratePublicKey(priv)
    addr := types.DeriveAddress(pub)
    ks.keys[addr] = &keyPair{priv: priv, pub: pub}
    return &types.Account{Address: addr, Balance: balance, Nonce: 0}, nil
}

// BuildTransaction creates and signs a transaction using stored keys.
func (ks *KeyStore) BuildTransaction(from common.Address, to common.Address, amount float64) (*types.Transaction, error) {
    pair := ks.keys[from]
    if pair == nil {
        return nil, fmt.Errorf("????? %s ???", from.Hex())
    }
    return types.NewTransaction(from, to, amount, pair.pub, pair.priv)
}

// PublicKey returns the public key for an address if available.
func (ks *KeyStore) PublicKey(addr common.Address) *ecc.Point {
    if ks == nil {
        return nil
    }
    if pair := ks.keys[addr]; pair != nil {
        return pair.pub
    }
    return nil
}

// PrivateKey returns the private key for an address if available.
func (ks *KeyStore) PrivateKey(addr common.Address) *big.Int {
    if ks == nil {
        return nil
    }
    if pair := ks.keys[addr]; pair != nil {
        return pair.priv
    }
    return nil
}
