/*
*	地址主类
 */
package keystore

import (
	"crypto/ecdsa"
	"crypto/md5"
	"polarcloud/core/utils"
)

type KeyStore struct {
	Wallet *Wallet
}

//根据种子新建keystore
func NewKeyStore(seed ...string) *KeyStore {
	ks := new(KeyStore)
	ks.Wallet = NewWallet()
	if len(seed) > 0 && seed[0] != "" {
		ks.Wallet.SeedPath = seed[0]
	}
	if len(seed) > 1 && seed[1] != "" {
		ks.Wallet.SeedFile = seed[1]
	}
	return ks
}

//加载种子
func (ks *KeyStore) Load() (int, error) {
	l, err := ks.Wallet.LoadSeeds()
	if err == nil {
		ks.Wallet.SetSeedIndex(0)
	}
	return l, err
}

//新增种子入keystore
func (ks *KeyStore) NewLoad(seed, password string) error {
	pass := md5.Sum([]byte(password))
	seedData, err := Encrypt([]byte(seed), pass[:])
	if err != nil {
		return err
	}
	seeds := Seed{Data: seedData}
	ks.Wallet.SetSeed(seeds)
	ks.Wallet.SaveSeed(ks.Wallet.Seeds)
	ks.Wallet.SetSeedIndex(0)
	//创建矿工地址
	ks.Wallet.GetNewAddress(pass[:])
	return nil
}

//获取地址列表
func (ks *KeyStore) GetAddr() []Address {
	addrlist := ks.Wallet.GetAllAddress()
	return addrlist
}

//生成一个新的地址，需要密码
func (ks *KeyStore) GetNewAddr(password string) (*Address, error) {
	pass := md5.Sum([]byte(password))
	addr, err := ks.Wallet.GetNewAddress(pass[:])
	return addr, err
}

//获取一个基础地址
func (ks *KeyStore) GetCoinbase() (*Address, error) {
	addr, err := ks.Wallet.GetCoinbase()
	return addr, err
}

//设置一个基础地址
func (ks *KeyStore) SetCoinbase(index int) {
	ks.Wallet.SetCoinbase(index)
}

//根据地址获取私钥
func (ks *KeyStore) GetPriKeyByAddress(address, password string) (prikey *ecdsa.PrivateKey, err error) {
	pass := md5.Sum([]byte(password))
	prikey, err = ks.Wallet.GetPriKey(address, pass[:])
	return
}

//验证地址
func (ks *KeyStore) ValidateAddress(address *Address) (validate Validate) {
	validate = ks.Wallet.ValidateAddress(address)
	return
}

//根据主地址获取一个新的扩展地址
func (ks *KeyStore) GetNewExpAddr(preAddress *Address) *utils.Multihash {
	addr := ks.Wallet.GetNewExpAddress(preAddress)
	return addr
}
