package ecc

import (
	"math/big"
	"testing"
)

func TestECC_HappyPath(t *testing.T) {
	seckey, err := NewPrivateKey()
	if err != nil {
		t.Fatalf("[环境错误] 无法生成私钥: %v", err)
	}
	pubkey := GeneratePublicKey(seckey)
	ecc := MyECC{}

	msg := []byte("学生评测专用测试语句：Hello Ethereum!")

	// 测试正常签名
	sign, err := ecc.Sign(msg, seckey)
	if err != nil || sign == nil {
		t.Fatalf("[密码学失败] 您的 Sign 方法未能成功返回签名对象。")
	}

	// 测试正常验签
	if !ecc.VerifySignature(msg, sign, pubkey) {
		t.Fatalf("[密码学失败] 错误：正常的验证签名逻辑无法通过测试。请检查您处理 ECC 方程的坐标生成或者逆运算是否严谨。")
	}
}

func TestECC_NegativePath(t *testing.T) {
	seckey, _ := NewPrivateKey()
	pubkey := GeneratePublicKey(seckey)
	ecc := MyECC{}
	msg := []byte("原始数据")

	sign, _ := ecc.Sign(msg, seckey)

	// 篡改 1：数据被篡改
	fakeMsg := []byte("被篡改的数据")
	if ecc.VerifySignature(fakeMsg, sign, pubkey) {
		t.Errorf("[密码验证失败] 您的代码返回了 true，未能识别出消息本身的哈希已经被篡改。")
	}

	// 篡改 2：签名 R 值被篡改
	tamperedRSign := &Signature{
		r: new(big.Int).Add(sign.r, big.NewInt(1)),
		s: sign.s,
	}
	if ecc.VerifySignature(msg, tamperedRSign, pubkey) {
		t.Errorf("[密码验证失败] 您的代码返回了 true，未能识别出签名对象中的 R 坐标已被篡改。")
	}

	// 篡改 3：使用错误的公钥验签
	wrongSecKey, _ := NewPrivateKey()
	wrongPubKey := GeneratePublicKey(wrongSecKey)
	if ecc.VerifySignature(msg, sign, wrongPubKey) {
		t.Errorf("[密码验证失败] 您的代码返回了 true，未能拦截使用不匹配公钥进行的验签。")
	}
}

func TestECC_EdgeCases(t *testing.T) {
	seckey, _ := NewPrivateKey()
	pubkey := GeneratePublicKey(seckey)
	ecc := MyECC{}

	// 空数据测试
	emptyMsg := []byte{}
	_, err := ecc.Sign(emptyMsg, seckey)
	if err != nil {
		t.Fatalf("[密码学边界错误] 给予空字节数据时签名失败: %v", err)
	}

	// 极端造假测试 (空结构体拦截，如果 panic 则交由总控拦截)
	emptySign := &Signature{
		r: big.NewInt(0),
		s: big.NewInt(0),
	}
	if ecc.VerifySignature(emptyMsg, emptySign, pubkey) {
		t.Errorf("[密码验证失败] 对于异常的 0 值签名，程序不应该判定为有效。")
	}
}
