package keystore

import (
	"crypto/ecdsa"
	"crypto/md5"
	"polarcloud/core/utils"
)

var (
	NWallet *Wallet
)

//钱包初始化
func init() {
	NWallet = NewWallet()
}

//加载种子
func Load() (int, error) {
	l, err := NWallet.LoadSeeds()
	if err == nil {
		NWallet.SetSeedIndex(0)
	}
	return l, err
}

//设置新的种子
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

//获取钱包地址列表
func GetAddr() []Address {
	addrlist := NWallet.GetAllAddress()
	return addrlist
}

//获取一个新的地址
func GetNewAddr(password string) (*Address, error) {
	pass := md5.Sum([]byte(password))
	addr, err := NWallet.GetNewAddress(pass[:])
	return addr, err
}

//获取基础地址
func GetCoinbase() (*Address, error) {
	addr, err := NWallet.GetCoinbase()
	return addr, err
}

//设置基础地址
func SetCoinbase(index int) {
	NWallet.SetCoinbase(index)
}

//根据地址获取私钥
func GetPriKeyByAddress(address, password string) (prikey *ecdsa.PrivateKey, err error) {
	pass := md5.Sum([]byte(password))
	prikey, err = NWallet.GetPriKey(address, pass[:])
	return
}

//验证地址合法性(Address类型)
func ValidateAddress(address *Address) (validate Validate) {
	validate = NWallet.ValidateAddress(address)
	return
}

//验证地址合法性(Addres类型)
func ValidateByAddress(address string) (validate Validate) {
	validate = NWallet.ValidateByAddress(address)
	return
}

//获取某个地址的扩展地址
func GetNewExpAddr(preAddress *Address) *utils.Multihash {
	addr := NWallet.GetNewExpAddress(preAddress)
	return addr
}

//根据公钥生成地址multihash
func BuildAddrByPubkey(pub []byte) (*utils.Multihash, error) {
	addr, err := buildAddrinfo(pub, Version)
	return addr, err
}
