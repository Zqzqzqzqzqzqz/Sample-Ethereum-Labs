package block

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"simple_eth/types"
)

// Block 由区块头和区块体组成。
type Block struct {
	Header *BlockHeader
	Body   *BlockBody
}

// BlockHeader 保存用于共识与链路的元数据。
type BlockHeader struct {
	Height        int64
	Hash          string
	PrevBlockHash string
	Timestamp     int64
	TxMerkleRoot  string // 交易 Merkle 根
	AcMerkleRoot  string // 账户 Merkle 根
	Validator     common.Address
	Nonce         int
}

// BlockBody 保存交易数据。
type BlockBody struct {
	Transactions []*types.Transaction
}

// CalculateBlockHash 基于区块头与交易列表生成确定性哈希。
func CalculateBlockHash(block *Block) string {
	if block == nil || block.Header == nil {
		return ""
	}
	transactionsJSON, _ := json.Marshal(block.Body.Transactions)
	record := fmt.Sprintf("%d%s%s%s%d%s%s",
		block.Header.Timestamp,
		transactionsJSON,
		block.Header.PrevBlockHash,
		block.Header.Validator.Hex(),
		block.Header.Nonce,
		block.Header.TxMerkleRoot,
		block.Header.AcMerkleRoot,
	)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}
