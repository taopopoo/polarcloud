package keystore

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/md5"
	"errors"
	"polarcloud/core/utils"
)

//种子
type Seed struct {
	Index     int
	Data      []byte
	AddrIndex int64
	Coinbase  CoinbaseInfo
}

//基础地址
type CoinbaseInfo Address

//扩展地址
type ExpAddress struct {
	PreAddress *Address
	Index      int64
	Pubkey     []byte
	Address    *utils.Multihash
}

//地址验证结果
type Validate struct {
	IsVerify     bool
	IsMine       bool
	IsType       int //1 mainaddr/2 expandaddr
	Version      uint64
	ExpVersion   uint64
	MainAddrInfo *utils.Multihash
	ExpAddrInfo  ExpAddress
}

//地址
type Address struct {
	SeedIndex int
	Index     int64
	PriKey    []byte
	Pubkey    []byte
	Hash      *utils.Multihash
}

//根据地址获取公钥
func (addr *Address) GetPubKey() []byte {
	return addr.Pubkey
}

//根据地址获取私钥
func (addr *Address) GetPriKey(password string) (*ecdsa.PrivateKey, error) {
	pass := md5.Sum([]byte(password))
	pri, err := Decrypt(addr.PriKey, pass[:])
	if err != nil {
		return nil, err
	}
	prikey, _ := utils.ParsePrikey(pri)
	pub, _ := utils.MarshalPubkey(&prikey.PublicKey)
	if !bytes.Equal(addr.Pubkey, pub) {
		return nil, errors.New("password is wrong")
	}
	return prikey, nil
}

//签名
func (addr *Address) Sign(text []byte, password string) (*[]byte, error) {
	pri, err := addr.GetPriKey(password)
	if err != nil {
		return nil, err
	}
	//	prikey, _ := utils.ParsePrikey(pri)
	sign, err := utils.Sign(pri, text)
	if err != nil {
		return nil, err
	}
	return sign, nil
}

//验证签名
func (addr *Address) Verify(text []byte, sign []byte) (bool, error) {
	res, err := utils.Verify(addr.Pubkey, text, sign)
	return res, err
}
