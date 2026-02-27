package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "sort"
    "strconv"
    "strings"

    "simple_eth/block"
    "simple_eth/consensus"
    "simple_eth/types"
    "simple_eth/wallet"

    "github.com/ethereum/go-ethereum/common"
)

type cli struct {
    chain       *block.Blockchain
    spv         *wallet.SPVService
    ks          *wallet.KeyStore
}

func main() {
    c := &cli{}
    c.run()
}

func (c *cli) run() {
    fmt.Println("简易以太坊 CLI")
    fmt.Println("命令: init [pow|pos], accounts, send <from> <to> <amount>[;...|auto], mine, changePowdifficult <difficulty>, pos-add <address> <stake>, spv-proof <txHash>, spv-verify <proofJSON>, help, exit（地址请使用0x前缀）")
    scanner := bufio.NewScanner(os.Stdin)
    for {
        fmt.Print("> ")
        if !scanner.Scan() {
            break
        }
        line := strings.TrimSpace(scanner.Text())
        if line == "" {
            continue
        }
        lower := strings.ToLower(line)
        switch {
        case strings.HasPrefix(lower, "init"):
            c.handleInit(line)
        case strings.EqualFold(lower, "accounts"):
            c.handleAccounts()
        case strings.HasPrefix(lower, "send"):
            c.handleSend(line)
        case strings.EqualFold(lower, "mine"):
            c.handleMine()
        case strings.HasPrefix(lower, "changepowdifficult"):
            c.handleChangePowDifficulty(line)
        case strings.HasPrefix(lower, "pos-add"):
            c.handlePosAdd(line)
        case strings.HasPrefix(lower, "spv-proof"):
            c.handleSPVProof(line)
        case strings.HasPrefix(lower, "spv-verify"):
            c.handleSPVVerify(line)
        case strings.EqualFold(lower, "help"):
            fmt.Println("命令: init [pow|pos], accounts, send <from> <to> <amount>[;...|auto], mine, changePowdifficult <difficulty>, pos-add <address> <stake>, spv-proof <txHash>, spv-verify <proofJSON>, help, exit（地址请使用0x前缀）")
        case strings.EqualFold(lower, "exit"), strings.EqualFold(lower, "quit"):
            fmt.Println("bye")
            return
        default:
            fmt.Println("无法识别的命令，输入 help 查看用法。")
        }
    }
}

func (c *cli) handleInit(line string) {
    fields := strings.Fields(line)
    mode := "pos"
    if len(fields) >= 2 {
        mode = strings.ToLower(fields[1])
    }
    var engine block.ConsensusEngine
    switch mode {
    case "pow":
        engine = &consensus.PoWEngine{}
    case "pos":
        engine = consensus.NewPoSEngine()
    default:
            fmt.Println("无法识别的命令，输入 help 查看用法。")
        engine = consensus.NewPoSEngine()
    }

    c.ks = wallet.NewKeyStore()
    const accountCount = 100
    const initialBalance = 10000.0
    accounts := make([]*types.Account, 0, accountCount)
    for i := 0; i < accountCount; i++ {
        acct, err := c.ks.CreateAccount(initialBalance)
        if err != nil {
        fmt.Printf("创建账户失败: %v\n", err)
            return
        }
        accounts = append(accounts, acct)
    }

    c.chain = block.NewBlockchain(engine, accounts)
    c.spv = wallet.NewSPVService(c.chain)
    fmt.Printf("区块链已初始化。模式=%s，高度=%d，账户数=%d\n", mode, c.chain.Blocks[0].Header.Height, len(c.chain.BootstrapAccounts))
}

func (c *cli) handleAccounts() {
    if c.chain == nil {
        fmt.Println("请先执行 init")
        return
    }

    fmt.Println("初始账户（使用地址作为唯一标识，send 命令请输0x开头的地址）:")
    bootstrap := make(map[common.Address]struct{})
    for _, acct := range c.chain.BootstrapAccounts {
        bootstrap[acct.Address] = struct{}{}
        stateAcct := c.chain.State[acct.Address]
        balance := 0.0
        nonce := int64(0)
        if stateAcct != nil {
            balance = stateAcct.Balance
            nonce = stateAcct.Nonce
        }
        fmt.Printf("%s 余额: %.2f 交易次数: %d\n", acct.Address.Hex(), balance, nonce)
    }

    extras := make([]common.Address, 0)
    for addr := range c.chain.State {
        if _, ok := bootstrap[addr]; ok {
            continue
        }
        extras = append(extras, addr)
    }
    if len(extras) > 0 {
        sort.Slice(extras, func(i, j int) bool { return strings.Compare(extras[i].Hex(), extras[j].Hex()) < 0 })
        fmt.Println("状态中其他地址:")
        for _, addr := range extras {
            acct := c.chain.State[addr]
            if acct == nil {
                continue
            }
            fmt.Printf("    %s 余额: %.2f 交易次数: %d\n", addr.Hex(), acct.Balance, acct.Nonce)
        }
    }
}

func (c *cli) handleSend(line string) {
    if c.chain == nil {
        fmt.Println("请先执行 init")
        return
    }
    pieces := strings.SplitN(line, " ", 2)
    if len(pieces) < 2 {
        fmt.Println("用法: send <from 0x...> <to 0x...> <amount>[; <from 0x...> <to 0x...> <amount> ...] | send auto")
        return
    }
    raw := strings.TrimSpace(pieces[1])
    if raw == "" {
        fmt.Println("用法: send <from 0x...> <to 0x...> <amount>[; <from 0x...> <to 0x...> <amount> ...] | send auto")
        return
    }

    var txs []*types.Transaction
    if strings.EqualFold(raw, "auto") {
        var err error
        txs, err = c.buildAutoTransactions()
        if err != nil {
            fmt.Printf("自动构造交易失败: %v\n", err)
            return
        }
    } else {
        parts := strings.Split(raw, ";")
        for _, part := range parts {
            fields := strings.Fields(strings.TrimSpace(part))
            if len(fields) != 3 {
                fmt.Println("每条记录格式: <from> <to> <amount>")
                return
            }
            fromAcct, err := c.resolveAccount(fields[0])
            if err != nil {
                fmt.Printf("发送方无效: %v\n", err)
                return
            }
            toAddr, err := c.resolveAddress(fields[1])
            if err != nil {
                fmt.Printf("接收方无效: %v\n", err)
                return
            }
            amt, err := strconv.ParseFloat(fields[2], 64)
            if err != nil {
                fmt.Printf("金额无效: %v\n", err)
                return
            }
            if c.ks == nil {
                fmt.Println("系统尚未初始化密钥")
                return
            }
            tx, err := c.ks.BuildTransaction(fromAcct.Address, toAddr, amt)
            if err != nil {
                fmt.Printf("构造交易失败: %v\n", err)
                return
            }
            txs = append(txs, tx)
        }
    }

    stateCopy := c.chain.State.Clone()
    if err := types.ValidateTransactions(txs, stateCopy); err != nil {
        for _, tx := range txs {
            fmt.Printf("交易 %s: 验证失败 (%v)\n", tx.Hash, err)
        }
        return
    }

    for _, tx := range txs {
        c.chain.TransactionPool.AddTransaction(tx)
        fmt.Printf("交易 %s: 已加入交易池\n", tx.Hash)
    }
}

func (c *cli) buildAutoTransactions() ([]*types.Transaction, error) {
    if c.chain == nil {
        return nil, fmt.Errorf("请先执行 init")
    }
    if c.ks == nil {
        return nil, fmt.Errorf("系统尚未初始化密钥")
    }

    senders := make([]common.Address, 0)
    receivers := make([]common.Address, 0)
    for addr, acct := range c.chain.State {
        if acct == nil {
            continue
        }
        receivers = append(receivers, addr)
        if acct.Balance > 1 && c.ks.PrivateKey(addr) != nil {
            senders = append(senders, addr)
        }
    }
    if len(senders) < 10 {
        return nil, fmt.Errorf("可用发送者不足10个（余额>1 且可用私钥），当前 %d", len(senders))
    }
    if len(receivers) < 10 {
        return nil, fmt.Errorf("可用接收者不足10个，当前 %d", len(receivers))
    }

    sort.Slice(senders, func(i, j int) bool { return strings.Compare(senders[i].Hex(), senders[j].Hex()) < 0 })
    sort.Slice(receivers, func(i, j int) bool { return strings.Compare(receivers[i].Hex(), receivers[j].Hex()) < 0 })
    senders = senders[:10]
    receivers = receivers[:10]

    txs := make([]*types.Transaction, 0, 10)
    for i := 0; i < 10; i++ {
        tx, err := c.ks.BuildTransaction(senders[i], receivers[i], 1)
        if err != nil {
            return nil, fmt.Errorf("构造第 %d 笔交易失败: %w", i+1, err)
        }
        txs = append(txs, tx)
    }
    return txs, nil
}

func (c *cli) handleMine() {
    if c.chain == nil {
        fmt.Println("请先执行 init")
        return
    }
    newBlock := c.chain.MineBlock(c.chain.State)
    if newBlock == nil {
        fmt.Println("未能产出区块（交易池为空或共识失败）")
        return
    }
    if err := c.chain.ValidateBlock(newBlock, c.chain.State); err != nil {
        fmt.Printf("追加区块失败: %v\n", err)
        return
    }
    hdr := newBlock.Header
    fmt.Printf("新区块已产出。Height=%d Hash=%s PrevHash=%s TxMerkleRoot=%s AcMerkleRoot=%s Timestamp=%d Validator=%s Nonce=%d TxCount=%d\n",
        hdr.Height, hdr.Hash, hdr.PrevBlockHash, hdr.TxMerkleRoot, hdr.AcMerkleRoot, hdr.Timestamp, hdr.Validator.Hex(), hdr.Nonce, len(newBlock.Body.Transactions))
}

func (c *cli) handleChangePowDifficulty(line string) {
	if c.chain == nil {
		fmt.Println("请先执行 init")
		return
	}
	engine, ok := c.chain.Engine.(*consensus.PoWEngine)
	if !ok {
		fmt.Println("当前不是 PoW 模式")
		return
	}
	fields := strings.Fields(line)
	if len(fields) != 2 {
		fmt.Println("用法: changePowdifficult <difficulty>")
		return
	}
	diff, err := strconv.Atoi(fields[1])
	if err != nil || diff < 4 {
		fmt.Println("difficulty 必须为整数且 >= 4")
		return
	}
	engine.Difficulty = diff
	fmt.Printf("PoW 难度已更新为 %d\n", diff)
}

func (c *cli) handlePosAdd(line string) {
	if c.chain == nil {
		fmt.Println("请先执行 init")
		return
	}
	engine, ok := c.chain.Engine.(*consensus.PoSEngine)
	if !ok {
		fmt.Println("当前不是 PoS 模式")
		return
	}
	fields := strings.Fields(line)
	if len(fields) != 3 {
		fmt.Println("用法: pos-add <address> <stake>")
		return
	}
	addr, err := c.resolveAddress(fields[1])
	if err != nil {
		fmt.Printf("地址无效: %v\n", err)
		return
	}
	stake, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		fmt.Printf("stake 无效: %v\n", err)
		return
	}
	if stake <= 0 || stake >= 32 {
		fmt.Println("stake 必须大于 0 且小于 32")
		return
	}
	if err := engine.AddValidatorWithStake(addr, stake); err != nil {
		fmt.Printf("添加质押者失败: %v\n", err)
		return
	}
	fmt.Printf("已添加质押者: %s, stake=%.2f\n", addr.Hex(), stake)
}

func (c *cli) handleSPVProof(line string) {
    if c.chain == nil || c.spv == nil {
        fmt.Println("请先执行 init")
        return
    }
    fields := strings.Fields(line)
    if len(fields) != 2 {
        fmt.Println("用法: spv-proof <txHash>")
        return
    }
    proof, err := c.spv.BuildProof(fields[1])
    if err != nil {
        fmt.Printf("生成证明失败: %v\n", err)
        return
    }
    out, _ := json.MarshalIndent(proof, "", "  ")
    fmt.Printf("SPV 证明:\n%s\n", string(out))
}

func (c *cli) handleSPVVerify(line string) {
    if c.chain == nil || c.spv == nil {
        fmt.Println("请先执行 init")
        return
    }
    parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
    if len(parts) != 2 {
        fmt.Println("用法: spv-verify <proofJSON>")
        return
    }
    var proof wallet.SPVProof
    if err := json.Unmarshal([]byte(parts[1]), &proof); err != nil {
        fmt.Printf("解析 proof JSON 失败: %v\n", err)
        return
    }
    ok, err := c.spv.VerifyProof(&proof)
    if err != nil {
        fmt.Printf("验证出错: %v\n", err)
        return
    }
    fmt.Printf("SPV 验证结果: %t\n", ok)
}

func (c *cli) resolveAccount(input string) (*types.Account, error) {
    if c.chain == nil {
        return nil, fmt.Errorf("区块链未初始化")
    }
    addr, err := c.resolveAddress(input)
    if err != nil {
        return nil, err
    }

    if acct := c.chain.State[addr]; acct != nil {
        return acct, nil
    }
    for _, acct := range c.chain.BootstrapAccounts {
        if acct.Address == addr {
            return acct, nil
        }
    }
    return nil, fmt.Errorf("未知账户 %s", addr.Hex())
}

func (c *cli) resolveAddress(input string) (common.Address, error) {
    trimmed := strings.TrimSpace(input)
    if trimmed == "" {
        return common.Address{}, fmt.Errorf("地址不能为空")
    }
    if !strings.HasPrefix(strings.ToLower(trimmed), "0x") {
        return common.Address{}, fmt.Errorf("地址必须以0x开头")
    }
    addr := common.HexToAddress(trimmed)
    if addr == (common.Address{}) {
        return common.Address{}, fmt.Errorf("无效地址: %s", trimmed)
    }
    return addr, nil
}
