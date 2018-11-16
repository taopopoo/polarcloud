package keystore

import (
	"bytes"
	"crypto/md5"
	"errors"
	"polarcloud/core/utils"
)

type Seed struct {
	Index     int
	Data      []byte
	AddrIndex int64
	Coinbase  CoinbaseInfo
}
type CoinbaseInfo Address
type ExpAddress struct {
	PreAddress *Address
	Index      int64
	Pubkey     []byte
	Address    *utils.Multihash
}
type Validate struct {
	IsVerify     bool
	IsMine       bool
	IsType       int //1 mainaddr/2 expandaddr
	Version      uint64
	ExpVersion   uint64
	MainAddrInfo *utils.Multihash
	ExpAddrInfo  ExpAddress
}
type Address struct {
	SeedIndex int
	Index     int64
	PriKey    []byte
	Pubkey    []byte
	Hash      *utils.Multihash
}

func (addr *Address) GetPubKey() []byte {
	return addr.Pubkey
}
func (addr *Address) GetPriKey(password string) ([]byte, error) {
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
	return pri, nil
}
func (addr *Address) Sign(text []byte, password string) (*[]byte, error) {
	pri, err := addr.GetPriKey(password)
	if err != nil {
		return nil, err
	}
	prikey, _ := utils.ParsePrikey(pri)
	sign, err := utils.Sign(prikey, text)
	if err != nil {
		return nil, err
	}
	return sign, nil
}
func (addr *Address) Verify(text []byte, sign string) (bool, error) {
	res, err := utils.Verify(addr.Pubkey, text, sign)
	return res, err
}
