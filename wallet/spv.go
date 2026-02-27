package wallet

import (
    "encoding/hex"
    "fmt"

    "simple_eth/block"
)

// SPVService provides helper functions for Merkle proof generation/verification.
type SPVService struct {
    Chain *block.Blockchain
}

// SPVProof is a JSON-friendly Merkle proof bundle.
type SPVProof struct {
    Height int             `json:"height"`
    TxHash string          `json:"tx_hash"`
    Path   []ProofNodeJSON `json:"path"`
}

// ProofNodeJSON captures a single Merkle sibling in hex form.
type ProofNodeJSON struct {
    Hash          string `json:"hash"`
    SiblingOnLeft bool   `json:"sibling_on_left"`
}

func NewSPVService(chain *block.Blockchain) *SPVService {
    return &SPVService{Chain: chain}
}

// BuildProof walks the chain to find a transaction and returns its proof packet.
func (s *SPVService) BuildProof(txHash string) (*SPVProof, error) {
    height, blk, idx, err := s.locateTransaction(txHash)
    if err != nil {
        return nil, err
    }
    tree := block.NewMerkleTreeFromTransactions(blk.Body.Transactions)
    if tree.Root == nil {
        return nil, fmt.Errorf("block %d has an empty transaction list", height)
    }
    steps, err := tree.SPVProof(idx)
    if err != nil {
        return nil, err
    }
    return encodeProof(height, txHash, steps), nil
}

// VerifyProof validates proof information against the local chain.
func (s *SPVService) VerifyProof(proof *SPVProof) (bool, error) {
    if proof == nil {
        return false, fmt.Errorf("proof is nil")
    }
    if s.Chain == nil {
        return false, fmt.Errorf("blockchain not initialized")
    }
    if proof.Height < 0 || proof.Height >= len(s.Chain.Blocks) {
        return false, fmt.Errorf("block height out of range")
    }
    blk := s.Chain.Blocks[proof.Height]
    if blk == nil || blk.Header == nil {
        return false, fmt.Errorf("block header missing")
    }
    rootBytes, err := hex.DecodeString(blk.Header.TxMerkleRoot)
    if err != nil {
        return false, fmt.Errorf("cannot decode block merkle root: %w", err)
    }
    leafBytes, err := hex.DecodeString(proof.TxHash)
    if err != nil {
        return false, fmt.Errorf("cannot decode transaction hash: %w", err)
    }
    steps, err := decodeProof(proof.Path)
    if err != nil {
        return false, err
    }
    return block.VerifyProof(leafBytes, steps, rootBytes), nil
}

func (s *SPVService) locateTransaction(txHash string) (int, *block.Block, int, error) {
    if s.Chain == nil {
        return -1, nil, -1, fmt.Errorf("blockchain not initialized")
    }
    for height, blk := range s.Chain.Blocks {
        if blk == nil || blk.Body == nil {
            continue
        }
        for idx, tx := range blk.Body.Transactions {
            if tx != nil && tx.Hash == txHash {
                return height, blk, idx, nil
            }
        }
    }
    return -1, nil, -1, fmt.Errorf("transaction %s not found in chain", txHash)
}

func encodeProof(height int, txHash string, proof []block.ProofStep) *SPVProof {
    path := make([]ProofNodeJSON, 0, len(proof))
    for _, step := range proof {
        path = append(path, ProofNodeJSON{
            Hash:          hex.EncodeToString(step.Hash),
            SiblingOnLeft: step.SiblingOnLeft,
        })
    }
    return &SPVProof{
        Height: height,
        TxHash: txHash,
        Path:   path,
    }
}

func decodeProof(path []ProofNodeJSON) ([]block.ProofStep, error) {
    steps := make([]block.ProofStep, 0, len(path))
    for _, node := range path {
        hashBytes, err := hex.DecodeString(node.Hash)
        if err != nil {
            return nil, fmt.Errorf("invalid proof node: %w", err)
        }
        steps = append(steps, block.ProofStep{
            Hash:          hashBytes,
            SiblingOnLeft: node.SiblingOnLeft,
        })
    }
    return steps, nil
}
