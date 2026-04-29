package consensus

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"simple_eth/block"

	"github.com/ethereum/go-ethereum/common"
)

// PoSEngine selects proposer via stake and encapsulates the block.
type PoSEngine struct {
	Validators map[common.Address]float64 // Validators and their stake
}

const (
	slotDurationSeconds int64 = 12
	maxEffectiveBalance       = 32.0
)

type validatorEntry struct {
	Address common.Address
	Stake   float64
}

func NewPoSEngine() *PoSEngine {
	return &PoSEngine{
		Validators: make(map[common.Address]float64),
	}
}

// AddValidatorWithStake adds a validator, requiring stake < 32.
func (e *PoSEngine) AddValidatorWithStake(address common.Address, stake float64) error {
	if stake <= 0 {
		return fmt.Errorf("stake must be positive")
	}
	if stake >= 32 {
		return fmt.Errorf("stake must be less than 32")
	}
	e.Validators[address] = stake
	return nil
}

// RemoveValidator removes a validator.
func (e *PoSEngine) RemoveValidator(address common.Address) {
	delete(e.Validators, address)
}

// ConstructConsensus selects a proposer and populates PoS consensus fields.
func (e *PoSEngine) ConstructConsensus(blockCandidate *block.Block) *block.Block {
	if blockCandidate == nil {
		return nil
	}
	if blockCandidate.Header == nil {
		blockCandidate.Header = &block.BlockHeader{}
	}
	if blockCandidate.Body == nil {
		blockCandidate.Body = &block.BlockBody{}
	}

	// TODO: Lab 3, combine seed generation, shuffling, and proposer selection for PoS block creation.
	entries := e.snapshotValidators()
	if len(entries) == 0 {
		return nil
	}

	slot := computeSlot(blockCandidate.Header.Timestamp)
	seed := generateSeed(blockCandidate.Header.PrevBlockHash, slot)
	shuffled := shuffleValidators(entries, seed)
	proposer := pickProposer(shuffled, seed)

	blockCandidate.Header.Validator = proposer
	blockCandidate.Header.Hash = calculatePoSHash(blockCandidate)
	return blockCandidate
}

// ValidateConsensus verifies proposer selection and block hash.
func (e *PoSEngine) ValidateConsensus(blockCandidate *block.Block, prevBlock *block.Block) bool {
	if blockCandidate == nil || blockCandidate.Header == nil || blockCandidate.Body == nil || prevBlock == nil || prevBlock.Header == nil {
		fmt.Println("区块或前一区块为空")
		return false
	}
	// 1. Verify previous block hash.
	if blockCandidate.Header.PrevBlockHash != prevBlock.Header.Hash {
		fmt.Println("前一区块哈希不匹配")
		return false
	}

	slot := computeSlot(blockCandidate.Header.Timestamp)
	entries := e.snapshotValidators()
	if len(entries) == 0 {
		fmt.Println("没有可用的验证者")
		return false
	}
	seed := generateSeed(blockCandidate.Header.PrevBlockHash, slot)
	shuffled := shuffleValidators(entries, seed)
	expected := pickProposer(shuffled, seed)
	if blockCandidate.Header.Validator != expected {
		fmt.Println("提议者与预期不一致")
		return false
	}
	// 3. Verify block hash.
	if blockCandidate.Header.Hash != calculatePoSHash(blockCandidate) {
		fmt.Println("区块哈希验证失败")
		return false
	}
	// 4. Verify timestamp increment.
	if blockCandidate.Header.Timestamp <= prevBlock.Header.Timestamp {
		fmt.Println("时间戳验证失败")
		return false
	}

	return true
}

func calculatePoSHash(blockCandidate *block.Block) string {
	record := fmt.Sprintf("%d%s%s%s%s",
		blockCandidate.Header.Timestamp,
		blockCandidate.Header.PrevBlockHash,
		blockCandidate.Header.Validator.Hex(),
		blockCandidate.Header.TxMerkleRoot,
		blockCandidate.Header.AcMerkleRoot,
	)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

func (e *PoSEngine) snapshotValidators() []validatorEntry {
	entries := make([]validatorEntry, 0, len(e.Validators))
	for addr, stake := range e.Validators {
		if stake <= 0 {
			continue
		}
		entries = append(entries, validatorEntry{Address: addr, Stake: stake})
	}
	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(entries[i].Address.Bytes(), entries[j].Address.Bytes()) < 0
	})
	return entries
}

func computeSlot(timestamp int64) int64 {
	seconds := timestamp / int64(time.Second)
	return seconds / slotDurationSeconds
}

// generateSeed generates seed (prev_hash + slot + "PROPOSER").
func generateSeed(prevHash string, slot int64) []byte {
	// TODO: Lab 3, deterministically generate a specific shuffle seed for the current slot to prevent fraud.
	prevBytes, _ := hex.DecodeString(prevHash)
	slotBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(slotBytes, uint64(slot))
	combined := append(prevBytes, slotBytes...)
	combined = append(combined, []byte("PROPOSER")...)
	h := sha256.Sum256(combined)
	return h[:]
}

// shuffleValidators performs deterministic shuffling using seed (Fisher-Yates).
func shuffleValidators(entries []validatorEntry, seed []byte) []validatorEntry {
	// TODO: Lab 3, establish a verifiable deterministic shuffling mechanism.
	shuffled := make([]validatorEntry, len(entries))
	copy(shuffled, entries)
	for i := len(shuffled) - 1; i > 0; i-- {
		r := hashUint64(seed, uint64(i), nil)
		j := int(r % uint64(i+1))
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return shuffled
}

// pickProposer selects a proposer based on effective stake using threshold sampling.
func pickProposer(shuffled []validatorEntry, seed []byte) common.Address {
	// TODO: Lab 3, select a valid proposer weighted by account stake to influence selection probability.
	if len(shuffled) == 0 {
		return common.Address{}
	}
	for i := 0; ; i++ {
		entry := shuffled[i%len(shuffled)]
		threshold := hashUint64(seed, uint64(i), []byte("THRESHOLD")) % uint64(maxEffectiveBalance)
		if effectiveBalance(entry.Stake) > float64(threshold) {
			return entry.Address
		}
	}
}

// effectiveBalance directly returns stake, 32 ETH limit is verified on addition.
func effectiveBalance(stake float64) float64 {
	return stake
}

func hashUint64(seed []byte, i uint64, tag []byte) uint64 {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, i)
	combined := append(append([]byte{}, seed...), buf...)
	if len(tag) > 0 {
		combined = append(combined, tag...)
	}
	h := sha256.Sum256(combined)
	return binary.BigEndian.Uint64(h[:8])
}
