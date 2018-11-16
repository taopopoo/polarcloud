/*
	1.通过一个随机字符串和密码生成wallet.json文件，保存加密后的种子文件和公钥。
*/
package keystore

import (
	"bytes"
	"crypto/ecdsa"
	//"crypto/elliptic"
	//"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	//"crypto/x509"
	//"encoding/pem"
	"errors"
	"log"
	//"os"
	"strings"
	"sync"
	//"yunpan/config"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"yunpan/config"
	"yunpan/core/utils"

	"golang.org/x/crypto/ripemd160"
)

//https://en.bitcoin.it/wiki/List_of_address_prefixes
var (
	Version       = uint64(0x00)
	Expandversion = uint64(0x15)
)

type Wallet struct {
	PrivateKey  *ecdsa.PrivateKey
	PublicKey   []byte
	Seeds       []Seed   //种子
	SeedsIndex  int      //种子序号
	LinkCode    []byte   //链码
	AddrList    sync.Map //相应种子的地址列表
	ExpAddrList sync.Map //扩展地址
	SeedPath    string   //种子目录
	SeedFile    string   //种子文件名
	AddrFile    string   //地址文件名
	ExpAddrFile string   //扩展地址文件名
}

func NewWallet() *Wallet {
	wallet := &Wallet{SeedPath: config.Wallet_path, SeedFile: config.Wallet_seed, AddrFile: "addr.json", ExpAddrFile: "expaddr.json"}
	return wallet
}

//加载种子
func (w *Wallet) LoadSeeds() (int, error) {
	seeds, err := w.LoadSeed()
	if err == nil {
		w.Seeds = seeds
	}
	return len(seeds), err
}

//设置种子序号并初始化地址
func (w *Wallet) SetSeedIndex(index int) error {
	if len(w.Seeds) == 0 || w.SeedsIndex > len(w.Seeds) {
		return errors.New("no Seed")
	}
	w.SeedsIndex = index
	//如果没有默认coinbase,则加载时设置
	seed := w.Seeds[w.SeedsIndex]
	if len(seed.Coinbase.Pubkey) == 0 {
		w.SetCoinbase(0)
	}
	//为初始址生成公私钥
	//w.firstGenerateSeed()
	//加载地址列表
	w.AddrList = sync.Map{}
	addrlist := w.getAddressList()
	for _, v := range addrlist {
		if w.SeedsIndex == v.SeedIndex {
			w.PublicKey = v.Pubkey
			addr := Address{SeedIndex: v.SeedIndex, Index: v.Index, PriKey: v.PriKey, Pubkey: v.Pubkey, Hash: v.Hash}
			w.syncAddrMap(&w.AddrList, w.SeedsIndex, addr)
		}
	}
	return nil
}

//增加种子
func (w *Wallet) SetSeed(seed Seed) {
	if len(w.Seeds) == 0 {
		seed.Index = 0
	} else {
		seed.Index = len(w.Seeds)
	}
	w.Seeds = append(w.Seeds, seed)
}

//获取码链
func (w *Wallet) GetLinkCode(linkcode *[]byte) {
	rand := sha512.Sum512(w.Seeds[w.SeedsIndex].Data)
	*linkcode = rand[:len(rand)/2]
}

func (w *Wallet) getAddress(index int64, password []byte, prePubKey, linkcode *[]byte) *Address {
	if len(*prePubKey) == 0 {
		//log.Println("first")
		_, pub, _ := w.firstGenerateSeed(password, linkcode)
		*prePubKey = pub
	}
	rand := w.GenerateSeed(index, prePubKey, linkcode)
	pri, pub, _ := w.newKeyPair(rand)
	*prePubKey = pub
	address, _ := buildAddrinfo(pub, Version)
	priencypt, err := Encrypt(pri, password)
	if err != nil {
		return nil
	}
	addr := Address{SeedIndex: w.SeedsIndex, Index: index, PriKey: priencypt, Pubkey: pub, Hash: address}
	w.syncAddrMap(&w.AddrList, w.SeedsIndex, addr)
	return &addr
}

//生成单个地址
func (w *Wallet) GetNewAddress(password []byte) (address *Address, err error) {
	row := w.Seeds[w.SeedsIndex]
	var pubkey []byte
	var linkcode []byte
	w.AddrList = sync.Map{}
	for i := int64(0); i <= row.AddrIndex; i++ {
		address = w.getAddress(i, password, &pubkey, &linkcode)
		if i == 0 {
			err := w.checkPass(address)
			if err != nil {
				//log.Println(err)
				return nil, err
			}
		}
	}
	row.AddrIndex += 1
	w.Seeds[w.SeedsIndex] = row
	w.Save()
	//如果没有默认coinbase,则生成第一个地址时设置
	if len(row.Coinbase.Pubkey) == 0 {
		w.SetCoinbase(0)
	}
	return address, nil
}
func (w *Wallet) GetAllAddress() []Address {
	addrlist, ok := w.AddrList.Load(w.SeedsIndex)
	if !ok {
		return nil
	}
	/*for _, v := range addrlist.([]Address) {
		addr, _ := buildAddrinfo(v.Pubkey, Version)
		addrlists = append(addrlists, addr)
	}*/
	return addrlist.([]Address)
}
func (w *Wallet) SetCoinbase(index int) {
	addlist := w.getAddressList()
	for _, v := range addlist {
		if v.Index == int64(index) {
			row := w.Seeds[w.SeedsIndex]
			cobs := (CoinbaseInfo)(v)
			row.Coinbase = cobs
			w.Seeds[w.SeedsIndex] = row
			w.Save()
		}
	}
}
func (w *Wallet) GetCoinbase() (*Address, error) {
	row := w.Seeds[w.SeedsIndex]
	var addr Address
	var err error
	if len(row.Coinbase.Pubkey) != 0 {
		//hash, _ := buildAddrinfo(row.Coinbase.Pubkey, Version)
		//addr.Hash = hash
		addr = (Address)(row.Coinbase)
	} else {
		err = errors.New("no Coinbase address")
	}
	return &addr, err
}
func buildAddrinfo(pubkey []byte, code uint64) (*utils.Multihash, error) {
	hash := HashPubKey(pubkey)
	bs, err := utils.Encode(hash, code)
	if err != nil {
		return nil, errors.New("encode error")
	}
	addrinfo := utils.Multihash(bs)
	return &addrinfo, nil
}

//生成多个地址
/*func (w *Wallet) GetMoreAddress(start, end int64) [][]byte {
	var address [][]byte
	row := w.Seeds[w.SeedsIndex]
	addrlist, _ := w.AddrList.LoadOrStore(w.SeedsIndex, []Address{})
	//如果地址已经生成过，则直接返回
	if end < row.AddrIndex {

		for _, v := range addrlist.([]Address)[start:end] {
			address = append(address, []byte(v.Address))
		}
	} else {
		addrlist := addrlist.([]Address)[start:row.AddrIndex]
		for _, v := range addrlist {
			address = append(address, []byte(v.Address))
		}
		for i := row.AddrIndex; i < end; i++ {
			address = append(address, w.getAddress(i))
			row.AddrIndex += 1
			w.Seeds[w.SeedsIndex] = row
		}
		w.Save()
	}

	return address
}*/

//获取地址私钥
func (w *Wallet) GetPriKey(address string, password []byte) (prikey *ecdsa.PrivateKey, err error) {
	err = errors.New("no address")
	for k, v := range w.Seeds {
		w.SeedsIndex = k
		var pubkey []byte
		var linkcode []byte
		for i := int64(0); i < v.AddrIndex; i++ {
			addr := w.getAddress(i, password, &pubkey, &linkcode)
			if i == 0 {
				err := w.checkPass(addr)
				if err != nil {
					//log.Println(err)
					return nil, err
				}
			}
			//log.Println(i, addr.B58String(), address)
			if strings.Contains(addr.Hash.B58String(), address) {
				prikey = w.PrivateKey
				err = nil
			}
		}
	}
	return
}

//获取扩展地址私钥
func (w *Wallet) GetExpandPriKey(address string) (prikey *ecdsa.PrivateKey, err error) {
	return nil, nil
}

//初始化扩展地址
func (w *Wallet) InitExpAddress() {
	expaddr := w.GetAllExpAddress()
	for _, v := range expaddr {
		w.syncExpAddrMap(&w.ExpAddrList, v.PreAddress, v)
	}
}
func (w *Wallet) getExpAddress(preaddress *Address, pubkey []byte, index int64) *utils.Multihash {
	ids := intToBytes(index)
	ids = append(ids, pubkey...)
	pubKeyHash := HashPubKey(ids)
	address, _ := buildAddrinfo(pubKeyHash, Expandversion)
	addr := ExpAddress{PreAddress: preaddress, Index: index, Pubkey: pubkey, Address: address}
	w.syncExpAddrMap(&w.ExpAddrList, preaddress, addr)
	return address
}

//生成新的扩展地址
func (w *Wallet) GetNewExpAddress(preaddress *Address) (expaddress *utils.Multihash) {
	addr, ok := w.getAddressInfo(preaddress)
	if ok {
		index := 0
		expaddr, ok := w.ExpAddrList.Load(preaddress)
		if ok {
			index = len(expaddr.([]ExpAddress))
		}
		expaddress = w.getExpAddress(preaddress, addr.Pubkey, int64(index))
	}
	w.saveExpAddr(w.ExpAddrList)
	return
}

func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)

	RIPEMD160Hasher := ripemd160.New()
	_, err := RIPEMD160Hasher.Write(publicSHA256[:])
	if err != nil {
		log.Panic(err)
	}
	publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)

	return publicRIPEMD160
}
func (w *Wallet) ValidateAddress(address *Address) (validate Validate) {
	return w.Validate(address)
}
func (w *Wallet) ValidateByAddress(address string) (validate Validate) {
	return w.Validate(address)
}
func (w *Wallet) Validate(addr interface{}) (validate Validate) {
	address := utils.Multihash{}
	switch addr.(type) {
	case *Address:
		address = *addr.(*Address).Hash
	case string:
		address, _ = utils.FromB58String(addr.(string))
	}
	dhash, err := utils.Decode(address)
	if err != nil {
		log.Println(err)
		return
	}
	if dhash.Code == Version && err == nil {
		validate.IsVerify = true
		validate.Version = dhash.Code
		validate.IsType = 1
	}
	dhash, err = utils.Decode(address)
	if err != nil {
		log.Println(err)
		return
	}
	if dhash.Code == Expandversion && err == nil {
		validate.IsVerify = true
		validate.Version = dhash.Code
		validate.IsType = 2
	}
	if !validate.IsVerify {
		return
	}
	if validate.IsType == 2 {
		addlist := w.GetAllExpAddress()
		for _, v := range addlist {
			if v.Address.B58String() == address.B58String() {
				validate.IsMine = true
				validate.ExpAddrInfo = v
				addr = v.PreAddress
			}
		}
	}
	addlist := w.getAddressList()
	for _, v := range addlist {
		if v.Hash.B58String() == address.B58String() {
			validate.IsMine = true
			validate.MainAddrInfo = v.Hash
		}
	}

	return
}

//生成初始公私钥
func (w *Wallet) firstGenerateSeed(password []byte, linkcode *[]byte) ([]byte, []byte, error) {
	seedData, err := Decrypt(w.Seeds[w.SeedsIndex].Data, password)
	if err != nil {
		return nil, nil, err
	}
	pri, pub, _ := w.newKeyPair(seedData)
	w.GetLinkCode(linkcode)
	return pri, pub, nil
}

/*
生成加密源
sindex seed index
index address index
*/
func (w *Wallet) GenerateSeed(index int64, prePubKey, linkcode *[]byte) []byte {
	rand := intToBytes(index)
	rand = append(rand, append(*linkcode, *prePubKey...)...)
	newrand := sha512.Sum512(rand)
	return newrand[:]
}

//生成私/公钥对
func (w *Wallet) newKeyPair(rand []byte) ([]byte, []byte, error) {
	//curve := elliptic.P256()
	curve := utils.S256()
	//private, err := ecdsa.GenerateKey(curve, rand.Reader)
	private, err := ecdsa.GenerateKey(curve, bytes.NewReader(rand))
	if err != nil {
		log.Println(err)
		return nil, nil, err
	}
	//pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	priKey, err := utils.MarshalPrikey(private)
	if err != nil {
		return nil, nil, err
	}
	pubKey, err := utils.MarshalPubkey(&private.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	/*buf := bytes.NewBuffer(nil)

	err = pem.Encode(buf, &pem.Block{Type: config.Wallet_addr_puk_type, Bytes: pubKey})
	if err != nil {
		return nil, err
	}

	bss := bytes.Split(buf.Bytes(), []byte("\n"))
	pubkey := bytes.Join(bss[1:len(bss)-2], nil)*/
	w.PrivateKey = private
	return priKey, pubKey, nil
}
func (w *Wallet) Save() error {
	if err := w.SaveSeed(w.Seeds); err != nil {
		return err
	}
	if err := w.saveAddr(w.AddrList); err != nil {
		return err
	}
	return nil
}
func (w *Wallet) checkPass(addr *Address) error {
	addrlist := w.getAddressList()
	if len(addrlist) == 0 {
		return nil
	}
	//log.Println("checkpass:", addrlist[0].Address.B58String(), addr.B58String())
	if addrlist[0].Hash.B58String() == addr.Hash.B58String() {
		return nil
	}
	return errors.New("password is wrong")
}
func (w *Wallet) SaveSeed(seed []Seed) error {
	s, err := json.Marshal(seed)
	err = writeToFile(w.SeedPath, w.SeedFile, s)
	return err
}

func (w *Wallet) LoadSeed() ([]Seed, error) {
	var seed []Seed
	bs, err := ioutil.ReadFile(filepath.Join(w.SeedPath, w.SeedFile))
	if err != nil {
		return seed, err
	}
	err = json.Unmarshal(bs, &seed)
	return seed, err
}

func (w *Wallet) saveAddr(addrs sync.Map) error {
	var addr []Address
	addrs.Range(func(k, v interface{}) bool {
		addr = append(addr, v.([]Address)...)
		return true
	})
	s, err := json.Marshal(addr)
	err = writeToFile(w.SeedPath, w.AddrFile, s)
	return err
}

func (w *Wallet) getAddressList() (addrs []Address) {
	bs, err := ioutil.ReadFile(filepath.Join(w.SeedPath, w.AddrFile))
	if err != nil {
		return []Address{}
	}
	err = json.Unmarshal(bs, &addrs)
	return
}
func (w *Wallet) getAddressInfo(address *Address) (addr Address, b bool) {
	preaddrs := w.getAddressList()
	for _, v := range preaddrs {
		if v.Hash.B58String() == address.Hash.B58String() {
			addr = v
			b = true
		}
	}
	return
}
func (w *Wallet) GetAllExpAddress() (addrs []ExpAddress) {
	bs, err := ioutil.ReadFile(filepath.Join(w.SeedPath, w.ExpAddrFile))
	if err != nil {
		return addrs
	}
	err = json.Unmarshal(bs, &addrs)
	return
}
func (w *Wallet) getExpAddressIndex(preaddr *Address) (index int64) {
	addrs := w.GetAllExpAddress()
	for _, v := range addrs {
		if v.PreAddress.Hash.B58String() == preaddr.Hash.B58String() {
			index++
		}
	}
	return
}
func (w *Wallet) saveExpAddr(addrs sync.Map) error {
	var addr []ExpAddress
	addrs.Range(func(k, v interface{}) bool {
		addr = append(addr, v.([]ExpAddress)...)
		return true
	})
	s, err := json.Marshal(addr)
	err = writeToFile(w.SeedPath, w.ExpAddrFile, s)
	return err
}
func (w *Wallet) syncAddrMap(s *sync.Map, key int, addr Address) {
	addrs, ok := s.LoadOrStore(key, []Address{addr})
	if ok {
		addrs = append(addrs.([]Address), addr)
		s.Store(key, addrs)
	}
}
func (w *Wallet) syncExpAddrMap(s *sync.Map, key *Address, addr ExpAddress) {
	addrs, ok := s.LoadOrStore(key.Hash.B58String(), []ExpAddress{addr})
	if ok {
		addrs = append(addrs.([]ExpAddress), addr)
		s.Store(key.Hash.B58String(), addrs)
	}
}
