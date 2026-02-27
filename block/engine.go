package block

// ConsensusEngine 负责在共识层补充区块信息并校验共识规则。
type ConsensusEngine interface {
	ConstructConsensus(block *Block) *Block
	ValidateConsensus(block *Block, prevBlock *Block) bool
}
