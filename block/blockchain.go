package block

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"simple_eth/types"

	"github.com/ethereum/go-ethereum/common"
)

// Blockchain represents the entire blockchain structure.
type Blockchain struct {
	Blocks            []*Block
	TransactionPool   *types.TransactionPool
	Engine            ConsensusEngine
	Metadata          *ChainMetadata
	State             types.State
	BootstrapAccounts []*types.Account
}

// ChainMetadata records genesis information for easy export.
type ChainMetadata struct {
	Accounts   []*ChainAccount `json:"accounts"`
	Consensus  string          `json:"consensus"`
	Validators []string        `json:"validators,omitempty"`
}

// ChainAccount records address and balance during genesis.
type ChainAccount struct {
	Address string  `json:"address"`
	Balance float64 `json:"balance"`
}

// NewBlockchain creates a new blockchain, receiving a consensus engine and initial accounts.
func NewBlockchain(engine ConsensusEngine, accounts []*types.Account) *Blockchain {
	chain := &Blockchain{
		TransactionPool: types.NewTransactionPool(),
		Engine:          engine,
	}
	if len(accounts) == 0 {
		panic("需要至少一个初始账户")
	}

	state := make(types.State, len(accounts))
	metaAccounts := make([]*ChainAccount, 0, len(accounts))
	addresses := make([]common.Address, 0, len(accounts))
	for _, acct := range accounts {
		if acct == nil {
			continue
		}
		copyAcct := *acct
		copyAcct.Nonce = 0
		state[acct.Address] = &copyAcct
		metaAccounts = append(metaAccounts, &ChainAccount{Address: acct.Address.Hex(), Balance: acct.Balance})
		addresses = append(addresses, acct.Address)
	}

	chain.Metadata = &ChainMetadata{Accounts: metaAccounts}
	chain.State = state
	chain.BootstrapAccounts = accounts

	body := &BlockBody{Transactions: []*types.Transaction{}}
	header := &BlockHeader{
		Height:        0,
		PrevBlockHash: "",
		Timestamp:     time.Now().UnixNano(),
		TxMerkleRoot:  CalculateMerkleRoot(body.Transactions),
		AcMerkleRoot:  CalculateAccountMerkleRoot(state),
	}
	genesisBlock := &Block{Header: header, Body: body}
	sealed := engine.ConstructConsensus(genesisBlock)
	if sealed == nil {
		panic("无法生成创世区块")
	}
	chain.Blocks = append(chain.Blocks, sealed)
	chain.Metadata.Consensus = consensusName(engine)
	if validators := configureValidators(engine, addresses, state); len(validators) > 0 {
		chain.Metadata.Validators = validators
	}
	outputMetadata(chain.Metadata)
	return chain
}

func (bc *Blockchain) MineBlock(state types.State) *Block {
	// TODO: Lab 2, assemble new block by executing valid txs from pool and linking headers.
	txs := bc.TransactionPool.GetTransactions() // 获取交易池中的交易
	if len(txs) == 0 {
		return nil
	}

	// 验证交易的有效性
	cloned := state.Clone()
	if err := types.ValidateTransactions(txs, cloned); err != nil {
		return nil
	}

	blockCandidate := bc.buildBlockTemplate(txs, cloned)
	sealed := bc.Engine.ConstructConsensus(blockCandidate)
	bc.TransactionPool.ClearTransactions()
	return sealed
}

// ValidateBlock acts as a validating node to verify a received block and append it to the chain.
func (bc *Blockchain) ValidateBlock(block *Block, state types.State) error {
	if block == nil {
		return fmt.Errorf("区块为空")
	}
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	validationState := state.Clone()
	if err := bc.validateBlock(block, prevBlock, validationState, bc.Engine); err != nil {
		return err
	}
	block.Header.Height = int64(len(bc.Blocks))
	bc.Blocks = append(bc.Blocks, block)
	for addr, acct := range validationState {
		if acct == nil {
			continue
		}
		copyAcct := *acct
		state[addr] = &copyAcct
	}
	return nil
}

// IsValid verifies the entire chain block by block.
func (bc *Blockchain) IsValid(state types.State) bool {
	workingState := state.Clone()
	for i := 1; i < len(bc.Blocks); i++ {
		currentBlock := bc.Blocks[i]
		prevBlock := bc.Blocks[i-1]

		if err := bc.validateBlock(currentBlock, prevBlock, workingState, bc.Engine); err != nil {
			fmt.Println(err.Error())
			return false
		}
	}
	return true
}

func (bc *Blockchain) buildBlockTemplate(transactions []*types.Transaction, state types.State) *Block {
	prevHash := ""
	if len(bc.Blocks) > 0 {
		prevHash = bc.Blocks[len(bc.Blocks)-1].Header.Hash
	}
	txList := make([]*types.Transaction, len(transactions))
	copy(txList, transactions)
	body := &BlockBody{Transactions: txList}
	header := &BlockHeader{
		PrevBlockHash: prevHash,
		Timestamp:     time.Now().UnixNano(),
		TxMerkleRoot:  CalculateMerkleRoot(txList),
		AcMerkleRoot:  CalculateAccountMerkleRoot(state),
	}
	return &Block{Header: header, Body: body}
}

func (bc *Blockchain) validateBlock(block *Block, prev *Block, state types.State, engine ConsensusEngine) error {
	if err := validateBlockStructure(block, prev); err != nil {
		return err
	}
	if engine == nil {
		return fmt.Errorf("共识引擎不存在")
	}
	if !engine.ValidateConsensus(block, prev) {
		return fmt.Errorf("共识验证失败")
	}
	if err := types.ValidateTransactions(block.Body.Transactions, state); err != nil {
		return err
	}

	expectedACRoot := CalculateAccountMerkleRoot(state)
	if expectedACRoot != block.Header.AcMerkleRoot {
		return fmt.Errorf("账户 Merkle Root 不匹配")
	}
	return nil
}

func validateBlockStructure(current *Block, prev *Block) error {
	if current == nil {
		return fmt.Errorf("区块为空")
	}
	if current.Header == nil {
		return fmt.Errorf("区块头为空")
	}
	if current.Header.Hash == "" {
		return fmt.Errorf("区块哈希为空")
	}
	if current.Header.Timestamp <= 0 {
		return fmt.Errorf("区块时间戳非法")
	}
	if current.Body == nil {
		return fmt.Errorf("区块体为空")
	}
	expectedRoot := CalculateMerkleRoot(current.Body.Transactions)
	if expectedRoot != current.Header.TxMerkleRoot {
		return fmt.Errorf("交易 Merkle Root 不匹配")
	}
	if prev == nil {
		if current.Header.PrevBlockHash != "" {
			return fmt.Errorf("创世区块前一个哈希必须为空")
		}
		return nil
	}
	if current.Header.PrevBlockHash != prev.Header.Hash {
		return fmt.Errorf("区块前一个哈希不匹配，期望 %s 实际 %s", prev.Header.Hash, current.Header.PrevBlockHash)
	}
	if current.Header.Timestamp < prev.Header.Timestamp {
		return fmt.Errorf("区块时间戳回退")
	}
	// 容忍度 10 秒
	if current.Header.Timestamp > time.Now().UnixNano()+int64(10*time.Second) {
		return fmt.Errorf("区块时间戳来自遥远的未来")
	}
	return nil
}

func configureValidators(engine ConsensusEngine, addresses []common.Address, state types.State) []string {
	vc, ok := engine.(interface {
		AddValidatorWithStake(common.Address, float64) error
	})
	if !ok || len(addresses) == 0 {
		return nil
	}
	sorted := make([]common.Address, len(addresses))
	copy(sorted, addresses)
	sort.Slice(sorted, func(i, j int) bool {
		return bytes.Compare(sorted[i].Bytes(), sorted[j].Bytes()) < 0
	})
	count := 32
	if len(sorted) < count {
		count = len(sorted)
	}
	result := make([]string, 0, count)
	for i := 0; i < count; i++ {
		addr := sorted[i]
		if acct := state[addr]; acct != nil {
			_ = vc.AddValidatorWithStake(addr, acct.Balance)
		}
		result = append(result, addr.Hex())
	}
	return result
}

func consensusName(engine ConsensusEngine) string {
	name := fmt.Sprintf("%T", engine)
	switch {
	case strings.Contains(name, "PoW"):
		return "PoW"
	case strings.Contains(name, "PoS"):
		return "PoS"
	default:
		return name
	}
}

func outputMetadata(meta *ChainMetadata) {
	if meta == nil {
		return
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return
	}
	fmt.Println(string(data))
}
