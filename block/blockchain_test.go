package block

import (
	"math"
	"math/big"
	"testing"
	"time"

	ecc "simple_eth/crypt"
	"simple_eth/types"

	"github.com/ethereum/go-ethereum/common"
)

type testEngine struct {
	calls int
}

func (e *testEngine) ConstructConsensus(blockCandidate *Block) *Block {
	e.calls++
	if blockCandidate == nil {
		return nil
	}
	if blockCandidate.Header == nil {
		blockCandidate.Header = &BlockHeader{}
	}
	blockCandidate.Header.Hash = "sealed"
	blockCandidate.Header.Nonce = 7
	return blockCandidate
}

func (e *testEngine) ValidateConsensus(_ *Block, _ *Block) bool {
	return true
}

func newTestAccount(t *testing.T, privVal int64, balance float64) (*types.Account, *big.Int, *ecc.Point) {
	t.Helper()
	priv := big.NewInt(privVal)
	pub := ecc.GeneratePublicKey(priv)
	addr := types.DeriveAddress(pub)
	if addr == (common.Address{}) {
		t.Fatalf("derived address is empty")
	}
	return &types.Account{Address: addr, Balance: balance, Nonce: 0}, priv, pub
}

func assertAccountState(t *testing.T, state types.State, addr common.Address, balance float64, nonce int64) {
	t.Helper()
	acct := state[addr]
	if acct == nil {
		t.Fatalf("account missing: %s", addr.Hex())
	}
	if math.Abs(acct.Balance-balance) > 1e-9 {
		t.Fatalf("unexpected balance for %s: got %.9f want %.9f", addr.Hex(), acct.Balance, balance)
	}
	if acct.Nonce != nonce {
		t.Fatalf("unexpected nonce for %s: got %d want %d", addr.Hex(), acct.Nonce, nonce)
	}
}

func TestMineBlockBuildsAndSealsBlock(t *testing.T) {
	engine := &testEngine{}
	acct1, priv1, pub1 := newTestAccount(t, 1, 20)
	acct2, _, _ := newTestAccount(t, 2, 5)
	bc := NewBlockchain(engine, []*types.Account{acct1, acct2})
	engine.calls = 0

	tx, err := types.NewTransaction(acct1.Address, acct2.Address, 3, pub1, priv1)
	if err != nil {
		t.Fatalf("new transaction: %v", err)
	}
	bc.TransactionPool.AddTransaction(tx)

	balance1 := bc.State[acct1.Address].Balance
	nonce1 := bc.State[acct1.Address].Nonce
	balance2 := bc.State[acct2.Address].Balance
	nonce2 := bc.State[acct2.Address].Nonce

	expectedState := bc.State.Clone()
	if err := types.ValidateTransactions([]*types.Transaction{tx}, expectedState); err != nil {
		t.Fatalf("validate transactions: %v", err)
	}
	expectedTxRoot := CalculateMerkleRoot([]*types.Transaction{tx})
	expectedAcRoot := CalculateAccountMerkleRoot(expectedState)

	mined := bc.MineBlock(bc.State)
	if mined == nil {
		t.Fatalf("expected mined block")
	}
	if engine.calls != 1 {
		t.Fatalf("expected consensus to run once, got %d", engine.calls)
	}
	if mined.Header == nil || mined.Body == nil {
		t.Fatalf("mined block missing header/body")
	}
	if mined.Header.Hash != "sealed" || mined.Header.Nonce != 7 {
		t.Fatalf("unexpected consensus fields: hash=%s nonce=%d", mined.Header.Hash, mined.Header.Nonce)
	}
	prevHash := bc.Blocks[len(bc.Blocks)-1].Header.Hash
	if mined.Header.PrevBlockHash != prevHash {
		t.Fatalf("unexpected prev hash: got %s want %s", mined.Header.PrevBlockHash, prevHash)
	}
	if mined.Header.TxMerkleRoot != expectedTxRoot {
		t.Fatalf("unexpected tx merkle root: got %s want %s", mined.Header.TxMerkleRoot, expectedTxRoot)
	}
	if mined.Header.AcMerkleRoot != expectedAcRoot {
		t.Fatalf("unexpected account merkle root: got %s want %s", mined.Header.AcMerkleRoot, expectedAcRoot)
	}
	if len(mined.Body.Transactions) != 1 || mined.Body.Transactions[0].Hash != tx.Hash {
		t.Fatalf("unexpected mined transactions")
	}
	if len(bc.TransactionPool.GetTransactions()) != 0 {
		t.Fatalf("transaction pool not cleared")
	}

	assertAccountState(t, bc.State, acct1.Address, balance1, nonce1)
	assertAccountState(t, bc.State, acct2.Address, balance2, nonce2)
}

func TestMineBlockEmptyPoolReturnsNil(t *testing.T) {
	engine := &testEngine{}
	acct1, _, _ := newTestAccount(t, 3, 10)
	bc := NewBlockchain(engine, []*types.Account{acct1})
	engine.calls = 0

	if block := bc.MineBlock(bc.State); block != nil {
		t.Fatalf("expected nil block when pool is empty")
	}
	if engine.calls != 0 {
		t.Fatalf("consensus should not run for empty pool")
	}
}

func TestValidateBlockPrevHashMismatch(t *testing.T) {
	engine := &testEngine{}
	acct1, _, _ := newTestAccount(t, 1, 20)
	bc := NewBlockchain(engine, []*types.Account{acct1})

	fakeBlock := &Block{
		Header: &BlockHeader{
			Height:        1,
			PrevBlockHash: "fake_hash_not_genesis",
			Timestamp:     time.Now().UnixNano(),
			Hash:          "somehash",
		},
		Body: &BlockBody{Transactions: []*types.Transaction{}},
	}

	err := bc.ValidateBlock(fakeBlock, bc.State)
	if err == nil {
		t.Fatalf("expected error on previous hash mismatch, but got nil")
	}
}

func TestValidateBlockTimestampRevert(t *testing.T) {
	engine := &testEngine{}
	acct1, _, _ := newTestAccount(t, 1, 20)
	bc := NewBlockchain(engine, []*types.Account{acct1})
	prevBlock := bc.Blocks[len(bc.Blocks)-1]

	fakeBlock := &Block{
		Header: &BlockHeader{
			Height:        1,
			PrevBlockHash: prevBlock.Header.Hash,
			Timestamp:     prevBlock.Header.Timestamp - 1000, // 时间倒流
			Hash:          "somehash",
		},
		Body: &BlockBody{Transactions: []*types.Transaction{}},
	}

	err := bc.ValidateBlock(fakeBlock, bc.State)
	if err == nil {
		t.Fatalf("expected error on timestamp revert, but got nil")
	}
}

func TestValidateBlockTamperedTxRoot(t *testing.T) {
	engine := &testEngine{}
	acct1, priv1, pub1 := newTestAccount(t, 1, 20)
	acct2, _, _ := newTestAccount(t, 2, 5)
	bc := NewBlockchain(engine, []*types.Account{acct1, acct2})

	tx, _ := types.NewTransaction(acct1.Address, acct2.Address, 3, pub1, priv1)
	bc.TransactionPool.AddTransaction(tx)
	mined := bc.MineBlock(bc.State)

	// 恶意篡改其中一笔交易
	mined.Body.Transactions[0].Amount = 10

	err := bc.validateBlock(mined, bc.Blocks[0], bc.State, bc.Engine)
	if err == nil {
		t.Fatalf("expected error on tampered transaction (merkle mismatch), but got nil")
	}
}

func TestValidateBlockFromFuture(t *testing.T) {
	engine := &testEngine{}
	acct1, _, _ := newTestAccount(t, 1, 20)
	bc := NewBlockchain(engine, []*types.Account{acct1})
	prevBlock := bc.Blocks[len(bc.Blocks)-1]

	fakeBlock := &Block{
		Header: &BlockHeader{
			Height:        1,
			PrevBlockHash: prevBlock.Header.Hash,
			Timestamp:     time.Now().UnixNano() + int64(100*time.Second), // 时间穿越到未来
			Hash:          "somehash",
		},
		Body: &BlockBody{Transactions: []*types.Transaction{}},
	}

	err := bc.ValidateBlock(fakeBlock, bc.State)
	if err == nil {
		t.Fatalf("[时间防线拦截失败] 预期拦截一个带着未来几分钟后时间戳的区块，但您的验证逻辑将其放行了。真实网络中这可能导致时间穿梭攻击！")
	}
}
