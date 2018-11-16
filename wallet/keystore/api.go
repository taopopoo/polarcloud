package keystore

import (
	"crypto/ecdsa"
	"crypto/md5"
	"yunpan/core/utils"
)

var (
	NWallet *Wallet
)

func init() {
	NWallet = NewWallet()
}
func Load() (int, error) {
	l, err := NWallet.LoadSeeds()
	if err == nil {
		NWallet.SetSeedIndex(0)
	}
	return l, err
}
func NewLoad(seed, password string) error {
	pass := md5.Sum([]byte(password))
	seedData, err := Encrypt([]byte(seed), pass[:])
	if err != nil {
		return err
	}
	seeds := Seed{Data: seedData}
	NWallet.SetSeed(seeds)
	NWallet.SaveSeed(NWallet.Seeds)
	NWallet.SetSeedIndex(0)
	//创建矿工地址
	NWallet.GetNewAddress(pass[:])
	return nil
}
func GetAddr() []Address {
	addrlist := NWallet.GetAllAddress()
	return addrlist
}
func GetNewAddr(password string) (*Address, error) {
	pass := md5.Sum([]byte(password))
	addr, err := NWallet.GetNewAddress(pass[:])
	return addr, err
}
func GetCoinbase() (*Address, error) {
	addr, err := NWallet.GetCoinbase()
	return addr, err
}
func SetCoinbase(index int) {
	NWallet.SetCoinbase(index)
}
func GetPriKeyByAddress(address, password string) (prikey *ecdsa.PrivateKey, err error) {
	pass := md5.Sum([]byte(password))
	prikey, err = NWallet.GetPriKey(address, pass[:])
	return
}

func ValidateAddress(address *Address) (validate Validate) {
	validate = NWallet.ValidateAddress(address)
	return
}
func ValidateByAddress(address string) (validate Validate) {
	validate = NWallet.ValidateByAddress(address)
	return
}
func GetNewExpAddr(preAddress *Address) *utils.Multihash {
	addr := NWallet.GetNewExpAddress(preAddress)
	return addr
}
func BuildAddrByPubkey(pub []byte) (*utils.Multihash, error) {
	addr, err := buildAddrinfo(pub, Version)
	return addr, err
}
