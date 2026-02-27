package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	ecc "simple_eth/crypt"

	"github.com/ethereum/go-ethereum/common"
)

// Transaction represents a value transfer between accounts.
type Transaction struct {
	Hash      string
	From      common.Address
	To        common.Address
	Amount    float64
	Timestamp int64
	PublicKey *ecc.Point
	Signature *ecc.Signature
}

// TransactionPool buffers pending transactions for block inclusion.
type TransactionPool struct {
	Transactions []*Transaction
}

// NewTransactionPool instantiates an empty pool.
func NewTransactionPool() *TransactionPool {
	return &TransactionPool{Transactions: make([]*Transaction, 0)}
}

// NewTransaction constructs a transaction, hashes it, and signs it with the provided keys.
func NewTransaction(from common.Address, to common.Address, amount float64, pub *ecc.Point, priv *big.Int) (*Transaction, error) {
	if from == (common.Address{}) {
		return nil, fmt.Errorf("缺少发送账户")
	}
	if (to == common.Address{}) {
		return nil, fmt.Errorf("接收地址无效")
	}
	if pub == nil || priv == nil {
		return nil, fmt.Errorf("缺少签名所需的密钥")
	}
	tx := &Transaction{
		From:      from,
		To:        to,
		Amount:    amount,
		Timestamp: time.Now().Unix(),
		PublicKey: pub,
	}
	tx.Hash = CalculateTransactionHash(tx)
	if err := tx.sign(priv); err != nil {
		return nil, err
	}
	return tx, nil
}

// AddTransaction appends a new transaction to the pool.
func (pool *TransactionPool) AddTransaction(tx *Transaction) {
	pool.Transactions = append(pool.Transactions, tx)
}

// GetTransactions exposes pending transactions.
func (pool *TransactionPool) GetTransactions() []*Transaction {
	return pool.Transactions
}

// ClearTransactions resets the pool after a block is mined.
func (pool *TransactionPool) ClearTransactions() {
	pool.Transactions = make([]*Transaction, 0)
}

// CalculateTransactionHash produces the deterministic transaction digest.
func CalculateTransactionHash(tx *Transaction) string {
	if tx == nil {
		return ""
	}
	record := tx.From.Hex() + tx.To.Hex() + fmt.Sprintf("%f", tx.Amount) + fmt.Sprintf("%d", tx.Timestamp)
	if tx.PublicKey != nil && tx.PublicKey.X != nil && tx.PublicKey.Y != nil {
		record += tx.PublicKey.X.String() + tx.PublicKey.Y.String()
	}
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

// ValidateTransaction performs stateless checks on a single transaction.
func ValidateTransaction(tx *Transaction) error {
	if tx == nil {
		return fmt.Errorf("交易不能为空")
	}
	if tx.From == (common.Address{}) || tx.To == (common.Address{}) {
		return fmt.Errorf("交易账户不能为空")
	}
	if tx.Amount <= 0 {
		return fmt.Errorf("交易金额必须为正值")
	}
	if tx.PublicKey == nil || tx.Signature == nil {
		return fmt.Errorf("交易缺少签名或公钥")
	}
	derived := DeriveAddress(tx.PublicKey)
	if derived == (common.Address{}) || derived != tx.From {
		return fmt.Errorf("公钥与地址不匹配")
	}
	if !tx.VerifySignature() {
		return fmt.Errorf("交易签名验证失败")
	}
	if tx.Hash != CalculateTransactionHash(tx) {
		return fmt.Errorf("交易哈希验证失败: %s", tx.Hash)
	}
	return nil
}

// ValidateTransactions validates and applies a batch of transactions against state.
func ValidateTransactions(txs []*Transaction, state State) error {
	// TODO: Lab 2, execute sandbox validation on current state to intercept nonce, overdraft, or signature errors.
	panic("Not implemented yet")
}

func (tx *Transaction) sign(priv *big.Int) error {
	if tx == nil {
		return fmt.Errorf("交易不能为空")
	}
	if priv == nil || priv.Sign() <= 0 {
		return fmt.Errorf("无效的私钥")
	}
	signer := ecc.MyECC{}
	sig, err := signer.Sign([]byte(tx.Hash), priv)
	if err != nil {
		return err
	}
	tx.Signature = sig
	return nil
}

// VerifySignature ensures the cached signature matches the transaction payload.
func (tx *Transaction) VerifySignature() bool {
	if tx == nil || tx.Signature == nil || tx.PublicKey == nil {
		return false
	}
	signer := ecc.MyECC{}
	return signer.VerifySignature([]byte(tx.Hash), tx.Signature, tx.PublicKey)
}
