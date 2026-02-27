package wallet

import (
	"simple_eth/block"
	"simple_eth/types"
	"testing"
)

func TestSPV_BuildProofBlackBox(t *testing.T) {
	// 手工捏造一条包含了两笔交易的长链
	targetTxHashStr := "tx_wallet_data_2"

	// mock transaction
	tx1 := &types.Transaction{Hash: "tx_wallet_data_1"}
	tx2 := &types.Transaction{Hash: targetTxHashStr}

	b := &block.Block{
		Header: &block.BlockHeader{
			// 不需要在测试用例强行写入真实的 Merkle Root，因为如果不测验证端，只调用 BuildProof，主要断言路径生成，不关心其一致。
		},
		Body: &block.BlockBody{
			Transactions: []*types.Transaction{tx1, tx2},
		},
	}

	chain := &block.Blockchain{
		Blocks: []*block.Block{b},
	}

	spvSvc := &SPVService{Chain: chain}

	// 正向查询
	proof, err := spvSvc.BuildProof(targetTxHashStr)
	if err != nil || proof == nil {
		t.Fatalf("[SPV 查询失败] 您的轻节点查询服务未能成功返回有效区块打包的默克尔路径证明。错误：%v", err)
	}

	// 反向（错误）查询
	// 任何会导致 Panic 的情况都将由 Grader 主体捕获判作 false。
	proofErr, err2 := spvSvc.BuildProof("non_existent_hash")
	if err2 == nil || proofErr != nil {
		t.Fatalf("[SPV 查询边界错误] 给予不存在的查询凭证，服务应当反馈错误拦截，而不是返回一个似乎'正确'但非法的路径堆栈。")
	}
}
