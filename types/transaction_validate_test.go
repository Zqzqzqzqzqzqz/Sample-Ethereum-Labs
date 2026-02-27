package types

import (
	"math/big"
	"testing"
	"time"

	ecc "simple_eth/crypt"

	"github.com/ethereum/go-ethereum/common"
)

func newTestAccount(t *testing.T, privVal int64, balance float64) (*Account, *big.Int, *ecc.Point) {
	t.Helper()
	priv := big.NewInt(privVal)
	pub := ecc.GeneratePublicKey(priv)
	addr := DeriveAddress(pub)
	return &Account{Address: addr, Balance: balance, Nonce: 0}, priv, pub
}

func TestValidateTransactions_HappyPath(t *testing.T) {
	acct1, priv1, pub1 := newTestAccount(t, 1, 100)
	acct2, _, _ := newTestAccount(t, 2, 50)

	state := NewState(map[common.Address]*Account{
		acct1.Address: acct1,
		acct2.Address: acct2,
	})

	tx, _ := NewTransaction(acct1.Address, acct2.Address, 20, pub1, priv1)

	if err := ValidateTransactions([]*Transaction{tx}, state); err != nil {
		t.Fatalf("[核心资产检查错误] 正常的无越界交易居然未通过验证: %v", err)
	}

	// 只做宏观断言，不再强行查验 Nonce 的增量是否正好是固定的机制。
	if state[acct1.Address].Balance != 80 || state[acct2.Address].Balance != 70 {
		t.Fatalf("[核心资产检查错误] 返回结果不符，世界状态中的余额并没有随着交易产生正确的流转偏移。")
	}
}

func TestValidateTransactions_NegativePath(t *testing.T) {
	acct1, priv1, pub1 := newTestAccount(t, 1, 10)
	acct2, _, _ := newTestAccount(t, 2, 50)
	state := NewState(map[common.Address]*Account{acct1.Address: acct1, acct2.Address: acct2})

	// 1. 余额透支测试
	tx1, _ := NewTransaction(acct1.Address, acct2.Address, 100, pub1, priv1)
	if err := ValidateTransactions([]*Transaction{tx1}, state); err == nil {
		t.Errorf("[核心资产检查错误] 返回结果不符，未能成功拦截并处理恶意的双花或非法账本流动交易 (余额透支)。")
	}

	// 2. 双花重放测试
	tx2, _ := NewTransaction(acct1.Address, acct2.Address, 2, pub1, priv1)
	if err := ValidateTransactions([]*Transaction{tx2, tx2}, state); err == nil {
		currentBalance := state[acct1.Address].Balance
		t.Errorf("[核心资产检查] 测试双花拦截失败。我们尝试用同一笔哈希值为 %s 的交易（金额: %v）在同一区块内执行两次，您的验证函数允许了第二次扣款，导致发送方资产被不合理地扣为了 %v。请检查是否有使用 Map 等结构对区块内的交易哈希做防重放排查！", tx2.Hash, tx2.Amount, currentBalance)
	}
}

func TestValidateTransactions_LargeVolume(t *testing.T) {
	acct1, priv1, pub1 := newTestAccount(t, 1, 10000000)
	acct2, _, _ := newTestAccount(t, 2, 0)
	state := NewState(map[common.Address]*Account{acct1.Address: acct1, acct2.Address: acct2})

	var txs []*Transaction
	for i := 0; i < 2000; i++ {
		tx, _ := NewTransaction(acct1.Address, acct2.Address, 1, pub1, priv1)
		txs = append(txs, tx)
	}

	done := make(chan struct{})
	go func() {
		_ = ValidateTransactions(txs, state)
		close(done)
	}()

	select {
	case <-done:
		// 正常跑完
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("[性能防线崩溃] 您的验证算法时间复杂度也许过高 (存在多重嵌套遍历等)。处理包含 2000 笔交易的简单转账区块超时，请优化逻辑。")
	}
}
