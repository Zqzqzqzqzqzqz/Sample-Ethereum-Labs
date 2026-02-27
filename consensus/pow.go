package consensus

import (
	"fmt"
	"strings"
	"time"

	"simple_eth/block"
)

// PoWEngine mines by adjusting nonce to satisfy the leading zeros rule.
type PoWEngine struct {
	// Difficulty represents the required number of leading zeros (minimum 4).
	Difficulty int
}

const powMinLeadingZeros = 4

func (e *PoWEngine) prefix() string {
	if e == nil {
		return strings.Repeat("0", powMinLeadingZeros)
	}
	if e.Difficulty < powMinLeadingZeros {
		return strings.Repeat("0", powMinLeadingZeros)
	}
	return strings.Repeat("0", e.Difficulty)
}

// ConstructConsensus performs PoW mining and populates consensus fields.
func (e *PoWEngine) ConstructConsensus(blockCandidate *block.Block) *block.Block {
	if blockCandidate == nil {
		return nil
	}
	if blockCandidate.Header == nil {
		blockCandidate.Header = &block.BlockHeader{}
	}
	if blockCandidate.Body == nil {
		blockCandidate.Body = &block.BlockBody{}
	}

	// Ensure unique timestamp to prevent rapid repetition in tests.
	time.Sleep(10 * time.Millisecond)

	blockCandidate.Header.Hash = ""
	blockCandidate.Header.Nonce = 0

	targetPrefix := e.prefix()
	_ = targetPrefix // Prevent unused variable error

	// TODO: Lab 3, implement brute-force loop to find a nonce satisfying the difficulty hash prefix.
	panic("Not implemented yet")
}

// ValidateConsensus verifies PoW rules.
func (e *PoWEngine) ValidateConsensus(blockCandidate *block.Block, prevBlock *block.Block) bool {
	if blockCandidate == nil || blockCandidate.Header == nil || blockCandidate.Body == nil || prevBlock == nil || prevBlock.Header == nil {
		fmt.Println("区块或前一区块为空")
		return false
	}

	targetPrefix := e.prefix()

	// 1. Verify previous block hash.
	if blockCandidate.Header.PrevBlockHash != prevBlock.Header.Hash {
		fmt.Println("前一区块哈希不匹配")
		return false
	}

	// 2. Verify PoW hash.
	hash := calculatePoWHash(blockCandidate, blockCandidate.Header.Nonce)
	if blockCandidate.Header.Hash != hash || !strings.HasPrefix(blockCandidate.Header.Hash, targetPrefix) {
		fmt.Println("PoW 验证失败")
		return false
	}

	// 3. Verify monotonic timestamp increment.
	if blockCandidate.Header.Timestamp <= prevBlock.Header.Timestamp {
		fmt.Println("时间戳验证失败")
		return false
	}

	return true
}

func calculatePoWHash(blockCandidate *block.Block, nonce int) string {
	// TODO: Lab 3, implement double hash algorithm combining block header and nonce.
	panic("Not implemented yet")
}

func calculateBlockBaseHash(blockCandidate *block.Block) string {
	if blockCandidate == nil || blockCandidate.Header == nil {
		return ""
	}
	headerCopy := *blockCandidate.Header
	headerCopy.Nonce = 0
	headerCopy.Hash = ""
	clone := &block.Block{
		Header: &headerCopy,
		Body:   blockCandidate.Body,
	}
	return block.CalculateBlockHash(clone)
}
