package wallet

import (
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestKeyStoreCreateAndRetrieve(t *testing.T) {
	ks := NewKeyStore()
	acct, err := ks.CreateAccount(100)
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	if math.Abs(acct.Balance-100) > 1e-9 {
		t.Errorf("expected balance 100, got %f", acct.Balance)
	}

	pub := ks.PublicKey(acct.Address)
	if pub == nil {
		t.Errorf("expected to retrieve public key, got nil")
	}

	priv := ks.PrivateKey(acct.Address)
	if priv == nil {
		t.Errorf("expected to retrieve private key, got nil")
	}
}

func TestKeyStoreMissingKeyRetrieval(t *testing.T) {
	ks := NewKeyStore()
	unknownAddr := common.HexToAddress("0x0000000000000000000000000000000000000999")

	if ks.PublicKey(unknownAddr) != nil {
		t.Errorf("expected nil public key for unknown address")
	}
	if ks.PrivateKey(unknownAddr) != nil {
		t.Errorf("expected nil private key for unknown address")
	}
}

func TestBuildTransactionWithValidKeys(t *testing.T) {
	ks := NewKeyStore()
	acct1, _ := ks.CreateAccount(50)
	acct2, _ := ks.CreateAccount(0)

	tx, err := ks.BuildTransaction(acct1.Address, acct2.Address, 10)
	if err != nil {
		t.Fatalf("failed to build transaction: %v", err)
	}

	if tx.Amount != 10 {
		t.Errorf("unexpected transaction amount: %f", tx.Amount)
	}
	if tx.From != acct1.Address || tx.To != acct2.Address {
		t.Errorf("unexpected addresses in transaction")
	}

	if !tx.VerifySignature() {
		t.Errorf("transaction signature is invalid")
	}
}

func TestBuildTransactionWithMissingKeys(t *testing.T) {
	ks := NewKeyStore()
	acct1, _ := ks.CreateAccount(50)
	unknownSender := common.HexToAddress("0x0000000000000000000000000000000000000999")

	_, err := ks.BuildTransaction(unknownSender, acct1.Address, 10)
	if err == nil {
		t.Fatalf("expected error when building transaction with unknown sender, got nil")
	}
}
