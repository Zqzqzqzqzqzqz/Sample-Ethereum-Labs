package consensus

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"simple_eth/block"
	"simple_eth/types"

	"github.com/ethereum/go-ethereum/common"
)

func buildPowBlock(timestamp int64) *block.Block {
	return &block.Block{
		Header: &block.BlockHeader{
			Timestamp:     timestamp,
			PrevBlockHash: "prevhash",
			TxMerkleRoot:  "txroot",
			AcMerkleRoot:  "acroot",
			Validator:     common.HexToAddress("0x0000000000000000000000000000000000000001"),
		},
		Body: &block.BlockBody{Transactions: []*types.Transaction{}},
	}
}

func TestCalculatePoWHash(t *testing.T) {
	blk := buildPowBlock(1234567890)
	nonce := 42

	headerCopy := *blk.Header
	headerCopy.Nonce = 0
	headerCopy.Hash = ""
	baseHash := block.CalculateBlockHash(&block.Block{
		Header: &headerCopy,
		Body:   blk.Body,
	})
	payload := fmt.Sprintf("%s%d", baseHash, nonce)
	first := sha256.Sum256([]byte(payload))
	second := sha256.Sum256(first[:])
	expected := hex.EncodeToString(second[:])

	if got := calculatePoWHash(blk, nonce); got != expected {
		t.Fatalf("unexpected pow hash: got %s want %s", got, expected)
	}
}

func TestPoWConstructConsensus(t *testing.T) {
	engine := &PoWEngine{Difficulty: 4}
	candidate := buildPowBlock(time.Now().UnixNano())

	mined := engine.ConstructConsensus(candidate)
	if mined == nil || mined.Header == nil {
		t.Fatalf("[PoW 失败] 您的工作量碰撞算法返回了空块。")
	}
	if mined.Header.Hash == "" {
		t.Fatalf("[PoW 失败] 您挖掘出的区块并未挂载有效的哈希戳。")
	}
	if !strings.HasPrefix(mined.Header.Hash, engine.prefix()) {
		t.Fatalf("[PoW 失败] 您的工作量证明算法找出的哈希(%s)不具有所需数量的 0，未能满足规定难度。", mined.Header.Hash)
	}
	expected := calculatePoWHash(mined, mined.Header.Nonce)
	if mined.Header.Hash != expected {
		t.Fatalf("[PoW 失败] 找出的区块哈希欺诈：与其宣称的 Nonce 重新计算所得的真实哈希不符。")
	}
}
