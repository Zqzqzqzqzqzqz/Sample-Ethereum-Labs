package consensus

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"reflect"
	"testing"
	"time"

	"simple_eth/block"
	"simple_eth/types"

	"github.com/ethereum/go-ethereum/common"
)

func manualShuffle(entries []validatorEntry, seed []byte) []validatorEntry {
	shuffled := make([]validatorEntry, len(entries))
	copy(shuffled, entries)
	for i := len(shuffled) - 1; i > 0; i-- {
		r := hashUint64(seed, uint64(i), nil)
		j := int(r % uint64(i+1))
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return shuffled
}

func manualPickProposer(shuffled []validatorEntry, seed []byte) common.Address {
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

func TestGenerateSeed(t *testing.T) {
	prevHash := "aabbcc"
	slot := int64(12)
	got := generateSeed(prevHash, slot)

	prevBytes, _ := hex.DecodeString(prevHash)
	slotBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(slotBytes, uint64(slot))
	combined := append(prevBytes, slotBytes...)
	combined = append(combined, []byte("PROPOSER")...)
	expected := sha256.Sum256(combined)

	if !bytes.Equal(got, expected[:]) {
		t.Fatalf("unexpected seed")
	}
}

func TestShuffleValidatorsDeterministic(t *testing.T) {
	entries := []validatorEntry{
		{Address: common.HexToAddress("0x0000000000000000000000000000000000000001"), Stake: 10},
		{Address: common.HexToAddress("0x0000000000000000000000000000000000000002"), Stake: 20},
		{Address: common.HexToAddress("0x0000000000000000000000000000000000000003"), Stake: 30},
		{Address: common.HexToAddress("0x0000000000000000000000000000000000000004"), Stake: 5},
	}
	original := make([]validatorEntry, len(entries))
	copy(original, entries)
	seed := []byte("seed")

	expected := manualShuffle(entries, seed)
	got := shuffleValidators(entries, seed)

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("[PoS 洗牌失败] 给定特定的验证者集合与种子，您的洗牌逻辑产生的结果与一致性随机算法预期不相符。")
	}
	// 【黑盒容错】我们原先这里有对于 entries 是否被原址覆写（修改了底层切片）的约束，现已根据黑盒要求屏蔽，允许学生写出 inplace shuffle 的代码。
}

func TestPickProposerDeterministic(t *testing.T) {
	entries := []validatorEntry{
		{Address: common.HexToAddress("0x0000000000000000000000000000000000000001"), Stake: 12},
		{Address: common.HexToAddress("0x0000000000000000000000000000000000000002"), Stake: 8},
		{Address: common.HexToAddress("0x0000000000000000000000000000000000000003"), Stake: 16},
	}
	seed := []byte("pick-seed")

	expected := manualPickProposer(entries, seed)
	got := pickProposer(entries, seed)
	if got != expected {
		t.Fatalf("[PoS 选举失败] 给予特定随机种子时，您的 pickProposer 并没有以代币权重抽中预定的记账人。")
	}
}

func TestPickProposerEmpty(t *testing.T) {
	if got := pickProposer(nil, []byte("seed")); got != (common.Address{}) {
		t.Fatalf("expected empty address for empty proposer list")
	}
}

func TestPoSConstructConsensus(t *testing.T) {
	engine := NewPoSEngine()
	addr1 := common.HexToAddress("0x0000000000000000000000000000000000000001")
	addr2 := common.HexToAddress("0x0000000000000000000000000000000000000002")
	if err := engine.AddValidatorWithStake(addr1, 10); err != nil {
		t.Fatalf("add validator: %v", err)
	}
	if err := engine.AddValidatorWithStake(addr2, 20); err != nil {
		t.Fatalf("add validator: %v", err)
	}

	candidate := &block.Block{
		Header: &block.BlockHeader{
			Timestamp:     int64(24 * time.Second),
			PrevBlockHash: "abcd",
			TxMerkleRoot:  "txroot",
			AcMerkleRoot:  "acroot",
		},
		Body: &block.BlockBody{Transactions: []*types.Transaction{}},
	}

	sealed := engine.ConstructConsensus(candidate)
	if sealed == nil || sealed.Header == nil {
		t.Fatalf("expected sealed block")
	}

	slot := computeSlot(sealed.Header.Timestamp)
	entries := engine.snapshotValidators()
	seed := generateSeed(sealed.Header.PrevBlockHash, slot)
	shuffled := shuffleValidators(entries, seed)
	expected := pickProposer(shuffled, seed)

	if sealed.Header.Validator != expected {
		t.Fatalf("[PoS 组装失败] 区块打出的 Proposer 签名者并非此轮的当选者。")
	}
	if sealed.Header.Hash != calculatePoSHash(sealed) {
		t.Fatalf("[PoS 封包失败] 块哈希未能使用正确的签名函数结算。")
	}
}

func TestPoSConstructConsensusNoValidators(t *testing.T) {
	engine := NewPoSEngine()
	candidate := &block.Block{
		Header: &block.BlockHeader{
			Timestamp:     int64(time.Second),
			PrevBlockHash: "abcd",
		},
		Body: &block.BlockBody{Transactions: []*types.Transaction{}},
	}
	if got := engine.ConstructConsensus(candidate); got != nil {
		t.Fatalf("expected nil block when no validators are set")
	}
}

func TestPoSValidateConsensusWrongProposer(t *testing.T) {
	engine := NewPoSEngine()
	addr1 := common.HexToAddress("0x0000000000000000000000000000000000000001")
	addr2 := common.HexToAddress("0x0000000000000000000000000000000000000002")
	_ = engine.AddValidatorWithStake(addr1, 10)
	_ = engine.AddValidatorWithStake(addr2, 20)

	prevBlock := &block.Block{
		Header: &block.BlockHeader{
			Timestamp: time.Now().UnixNano(),
			Hash:      "genesisHash",
		},
	}

	candidate := &block.Block{
		Header: &block.BlockHeader{
			Timestamp:     int64(24*time.Second) + prevBlock.Header.Timestamp,
			PrevBlockHash: prevBlock.Header.Hash,
		},
		Body: &block.BlockBody{Transactions: []*types.Transaction{}},
	}

	// 正确的封装流程会填入正确的 proposer
	sealed := engine.ConstructConsensus(candidate)

	// 我们恶意修改 Proposer (李代桃僵)
	sealed.Header.Validator = common.HexToAddress("0xDEADBEEF00000000000000000000000000000000")
	// 并伪造新的签名哈希
	sealed.Header.Hash = calculatePoSHash(sealed)

	// 此时 validate 应该返回 false 因为虽然 hash 对了，但该 proposer 并非种子计算出的预期轮值节点
	if engine.ValidateConsensus(sealed, prevBlock) {
		t.Fatalf("[共识验证拦截失败] 恶意的非授权 Proposer 节点伪造的新区块未受到你们代码的 Validate 阻击。")
	}
}
