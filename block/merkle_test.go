package block

import (
	"testing"
)

func TestMerkleTree_BlackBoxIntegration(t *testing.T) {
	data := [][]byte{
		[]byte("tx_data_1"),
		[]byte("tx_data_2"),
		[]byte("tx_data_3"),
		[]byte("tx_data_4"),
	}

	tree := NewMerkleTree(data)
	if tree == nil || tree.Root == nil {
		t.Fatalf("[默克尔链错误] 您生成的树为空，未能正确组装叶子节点。")
	}

	// 抽出第 2 笔交易尝试 SPV
	proof, err := tree.SPVProof(1)
	if err != nil {
		t.Fatalf("[默克尔链错误] 获取 SPV 证明失败: %v", err)
	}

	// 拿着学生自己产出的 proof 自举调用验证
	ok := VerifyProof(data[1], proof, tree.Root.Data)
	if !ok {
		t.Fatalf("[默克尔链错误] 您生成的树无法通过 SPV 自举验证。这说明构建的过程、哈希方向或者兄弟节点的提取有严重逻辑错误。")
	}

	// 恶意篡改验证
	fakeData := []byte("fake_tx_data_2")
	if VerifyProof(fakeData, proof, tree.Root.Data) {
		t.Fatalf("[默克尔链错误] 您的验证函数未能防范针对叶子数据的恶意篡改，由于哈希抗传递，它本该验证失败的。")
	}
}

func TestMerkleTree_OddAndEdgeLeaves(t *testing.T) {
	// 测试 1: 0 个叶子节点的极限容错
	var emptyData [][]byte
	emptyTree := NewMerkleTree(emptyData)
	if emptyTree != nil && emptyTree.Root != nil {
		t.Fatalf("[默克尔链结构错误] 当叶子数量为 0 时，树应该返回 nil 或是无根节点对象，而不应该强行构建出无意义的哈希头。")
	}

	// 测试 2: 奇数个叶子节点 (例如 7 个)
	var oddData [][]byte
	for i := 0; i < 7; i++ {
		oddData = append(oddData, []byte("odd_tx_"+string(rune('1'+i))))
	}
	oddTree := NewMerkleTree(oddData)
	if oddTree == nil || oddTree.Root == nil {
		t.Fatalf("[默克尔容错降级] 输入奇数 (7) 个叶子时出错，未能建立或返回了一棵空树。如果遇到单节点无法凑对时，您的代码是否做到了将该单个节点的哈希克隆自身并拼凑计算？")
	}

	// 取最后一个叶子 (下标 6) 进行 SPV 查询
	proof, err := oddTree.SPVProof(6)
	if err != nil {
		t.Fatalf("[默克尔链错误] 奇数节点树环境下 SPV 切点提取崩溃。%v", err)
	}

	ok := VerifyProof(oddData[6], proof, oddTree.Root.Data)
	if !ok {
		t.Fatalf("[默克尔链边界错误] 在奇数个叶子节点（最后一个必定发生了自我复制哈希）的边界情形下生成了错误的树拓扑导致 SPV 签名自举验证不过。")
	}
}
