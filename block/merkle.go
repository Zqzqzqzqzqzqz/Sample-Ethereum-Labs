package block

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math"
	"sort"

	"simple_eth/types"

	"github.com/ethereum/go-ethereum/common"
)

// ProofStep describes a single SPV proof element.
type ProofStep struct {
	Hash          []byte
	SiblingOnLeft bool
}

// MerkleTree represents a classic binary Merkle tree.
type MerkleTree struct {
	Root   *MerkleNode
	leaves []*MerkleNode
}

// MerkleNode is a node in the Merkle tree.
type MerkleNode struct {
	Left   *MerkleNode
	Right  *MerkleNode
	Parent *MerkleNode
	Data   []byte
	isLeft bool
}

// NewMerkleTree builds a Merkle tree from arbitrary byte slices.
func NewMerkleTree(data [][]byte) *MerkleTree {
	// TODO: Lab 2, build Merkle tree bottom-up by hashing pairs.
	panic("Not implemented yet")
}

// SPVProof returns the Merkle path for the leaf at index.
func (t *MerkleTree) SPVProof(index int) ([]ProofStep, error) {
	// TODO: Lab 2, provide bottom-up sibling hashes for SPV proof.
	panic("Not implemented yet")
}

// VerifyProof verifies an SPV proof against the expected root.
func VerifyProof(leaf []byte, path []ProofStep, expectedRoot []byte) bool {
	// TODO: Lab 2, verify SPV computed root against expected root.
	panic("Not implemented yet")
}

// CalculateMerkleRoot hashes all transactions and returns the root hex string.
func CalculateMerkleRoot(transactions []*types.Transaction) string {
	data := transactionsToData(transactions)
	if len(data) == 0 {
		return ""
	}
	tree := NewMerkleTree(data)
	if tree.Root == nil {
		return ""
	}
	return hex.EncodeToString(tree.Root.Data)
}

// NewMerkleTreeFromTransactions builds a tree directly from transactions.
func NewMerkleTreeFromTransactions(transactions []*types.Transaction) *MerkleTree {
	data := transactionsToData(transactions)
	return NewMerkleTree(data)
}

func duplicateNode(node *MerkleNode) *MerkleNode {
	if node == nil {
		return nil
	}
	dataCopy := make([]byte, len(node.Data))
	copy(dataCopy, node.Data)
	return &MerkleNode{Data: dataCopy}
}

func transactionsToData(transactions []*types.Transaction) [][]byte {
	if len(transactions) == 0 {
		return nil
	}
	data := make([][]byte, 0, len(transactions))
	for _, tx := range transactions {
		if tx == nil {
			continue
		}
		bytes, err := hex.DecodeString(tx.Hash)
		if err != nil {
			bytes = []byte(tx.Hash)
		}
		data = append(data, bytes)
	}
	return data
}

// CalculateAccountMerkleRoot hashes all accounts (sorted by address) and returns the root hex string.
func CalculateAccountMerkleRoot(state types.State) string {
	if len(state) == 0 {
		return ""
	}
	data := accountsToData(state)
	if len(data) == 0 {
		return ""
	}
	tree := NewMerkleTree(data)
	if tree.Root == nil {
		return ""
	}
	return hex.EncodeToString(tree.Root.Data)
}

func accountsToData(state types.State) [][]byte {
	addresses := make([]common.Address, 0, len(state))
	for addr, acct := range state {
		if acct == nil {
			continue
		}
		addresses = append(addresses, addr)
	}
	if len(addresses) == 0 {
		return nil
	}
	sort.Slice(addresses, func(i, j int) bool {
		return bytes.Compare(addresses[i].Bytes(), addresses[j].Bytes()) < 0
	})

	data := make([][]byte, 0, len(addresses))
	for _, addr := range addresses {
		acct := state[addr]
		if acct == nil {
			continue
		}
		entry := make([]byte, 0, len(addr.Bytes())+16)
		entry = append(entry, addr.Bytes()...)

		balanceBits := math.Float64bits(acct.Balance)
		tmp := make([]byte, 8)
		binary.BigEndian.PutUint64(tmp, balanceBits)
		entry = append(entry, tmp...)

		binary.BigEndian.PutUint64(tmp, uint64(acct.Nonce))
		entry = append(entry, tmp...)

		data = append(data, entry)
	}
	return data
}
