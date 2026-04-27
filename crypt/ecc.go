package ecc

import (
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	s256    = btcec.S256()
	P, _    = new(big.Int).SetString("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F", 0)
	N, _    = new(big.Int).SetString("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 0)
	B, _    = new(big.Int).SetString("0x0000000000000000000000000000000000000000000000000000000000000007", 0)
	Gx, _   = new(big.Int).SetString("0x79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798", 0)
	Gy, _   = new(big.Int).SetString("0x483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8", 0)
	BitSize = 256
	G       = &Point{Gx, Gy}
)

type Point struct {
	X *big.Int
	Y *big.Int
}

type Signature struct {
	s *big.Int
	r *big.Int
}

type ECC interface {
	Sign(msg []byte, secKey *big.Int) (*Signature, error)
	VerifySignature(msg []byte, signature *Signature, pubkey *Point) bool
}

type MyECC struct {
}

func NewPrivateKey() (*big.Int, error) {
	k, err := newRand()
	if err != nil {
		return nil, err
	}
	if err := checkBigIntSize(k); err != nil {
		return nil, fmt.Errorf("k error: %s", err)
	}

	return k, nil
}

func GeneratePublicKey(secKey *big.Int) *Point {
	return Multi(G, secKey)
}

func (ecc *MyECC) Sign(msg []byte, secKey *big.Int) (*Signature, error) {
	z := hashMessage(msg)
	k, err := NewPrivateKey()

	if err != nil {
		return nil, fmt.Errorf("failed to generate k: %w", err)
	}

	// R = kG， r = R.x mod N
	R := Multi(G, k)
	r := new(big.Int).Mod(R.X, N)

	// s = (z + r * e) / k mod N
	// k 的逆元 (k^-1 mod N)
	k_Inv := new(big.Int).ModInverse(k, N)

	//  r * e mod N
	re := new(big.Int).Mul(r, secKey)
	re.Mod(re, N)

	// z + re mod N
	z_Plus_re := new(big.Int).Add(z, re)
	z_Plus_re.Mod(z_Plus_re, N)

	// s = (z + re) * k^-1 mod N
	s := new(big.Int).Mul(z_Plus_re, k_Inv)
	s.Mod(s, N)

	return &Signature{
		s: s,
		r: r,
	}, nil
}

// >>> point = S256Point(px, py)
// >>> s_inv = pow(s, N-2, N)  ❶
// >>> u = z * s_inv % N  ❷
// >>> v = r * s_inv % N  ❸
// >>> print((u*G + v*point).x.num == r)
func (ecc *MyECC) VerifySignature(msg []byte, signature *Signature, pubkey *Point) bool {
	// TODO: Lab 1, verify signature authenticity by inferring uG + vP = R with public key.
	z := hashMessage(msg)
	r := signature.r
	s := signature.s
	if r.Sign() <= 0 || r.Cmp(N) >= 0 || s.Sign() <= 0 || s.Cmp(N) >= 0 { //check r and s are in the range [1, N-1]
		return false
	}
	//  s_inv = s^-1 mod N
	sInv := new(big.Int).ModInverse(s, N)
	// u = z * s^-1 mod N
	u := new(big.Int).Mul(z, sInv)
	u.Mod(u, N)
	// v = r * s^-1 mod N
	v := new(big.Int).Mul(r, sInv)
	v.Mod(v, N)

	// R' = uG + vP
	uG := Multi(G, u)
	vP := Multi(pubkey, v)
	Rx, _ := s256.Add(uG.X, uG.Y, vP.X, vP.Y)
	if Rx == nil { //验证失败
		return false
	}
	RxMod := new(big.Int).Mod(Rx, N)
	return RxMod.Cmp(r) == 0  //如果 Rx mod N == r，则验证成功

}

func hashMessage(msg []byte) *big.Int {
	hash := crypto.Keccak256(msg)
	z := new(big.Int).SetBytes(hash)
	z.Mod(z, N)
	return z
}

func main() {
	seckey, err := NewPrivateKey()
	if err != nil {
		fmt.Println("error!")
	}
	pubkey := GeneratePublicKey(seckey)

	ecc := MyECC{}
	msg := []byte("test1")
	msg2 := []byte("test2")

	sign, err := ecc.Sign(msg, seckey)
	if err != nil {
		fmt.Printf("err %v\n", err)
		return
	}

	fmt.Printf("verify %v\n", ecc.VerifySignature(msg, sign, pubkey))
	fmt.Printf("verify %v\n", ecc.VerifySignature(msg2, sign, pubkey))

}
