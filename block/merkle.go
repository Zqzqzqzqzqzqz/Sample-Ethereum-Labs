package block

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math"
	"sort"
	"fmt"
	"crypto/sha256"

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
	if len(data) == 0 {
		return nil
	}

	//建立叶子节点
	nodes := make([]*MerkleNode, len(data))
	for i, d := range data {
		h := sha256.Sum256(d)
		nodes[i] = &MerkleNode{Data: h[:]}
	}
	leaves := make([]*MerkleNode, len(nodes))
	copy(leaves, nodes)

	// 自底向上构建树，合并
	for len(nodes) > 1 {
		if len(nodes)%2 == 1 {
			nodes = append(nodes, duplicateNode(nodes[len(nodes)-1])) //如果节点数为奇数，复制最后一个节点以保持完全二叉树结构
		}
		level := make([]*MerkleNode, 0, len(nodes)/2)
		for i := 0; i < len(nodes); i += 2 { 
			left := nodes[i]
			right := nodes[i+1]
			left.isLeft = true
			right.isLeft = false
			combined := append(left.Data, right.Data...) //左右节点的哈希值拼接后再哈希得到父节点的哈希值
			h := sha256.Sum256(combined)
			parent := &MerkleNode{Left: left, Right: right, Data: h[:]}
			left.Parent = parent
			right.Parent = parent
			level = append(level, parent)
		}
		nodes = level
	}

	return &MerkleTree{Root: nodes[0], leaves: leaves}
}

// SPVProof returns the Merkle path for the leaf at index.
func (t *MerkleTree) SPVProof(index int) ([]ProofStep, error) {
	// TODO: Lab 2, provide bottom-up sibling hashes for SPV proof.
	if t == nil || t.Root == nil { //空树
		return nil, fmt.Errorf("empty tree")
	}
	if index < 0 || index >= len(t.leaves) {
		return nil, fmt.Errorf("index %d out of range (leaves: %d)", index, len(t.leaves))
	}

	var path []ProofStep
	current := t.leaves[index]
	for current.Parent != nil { //从叶子节点向上遍历到根节点
		parent := current.Parent
		var sibling *MerkleNode
		var siblingOnLeft bool
		if current.isLeft { //如果当前节点是左子节点，则兄弟节点在右边
			sibling = parent.Right
			siblingOnLeft = false
		} else {
			sibling = parent.Left
			siblingOnLeft = true
		}
		path = append(path, ProofStep{Hash: sibling.Data, SiblingOnLeft: siblingOnLeft}) 
		current = parent
	}
	return path, nil
}

// VerifyProof verifies an SPV proof against the expected root.
func VerifyProof(leaf []byte, path []ProofStep, expectedRoot []byte) bool {
	// TODO: Lab 2, verify SPV computed root against expected root.
	h := sha256.Sum256(leaf)
	current := h[:]
	for _, step := range path { //遍历SPV路径，逐步计算当前节点的哈希值
		var combined []byte
		if step.SiblingOnLeft {
			combined = append(step.Hash, current...)
		} else {
			combined = append(current, step.Hash...)
		}
		h := sha256.Sum256(combined)
		current = h[:]
	}
	return bytes.Equal(current, expectedRoot)
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
