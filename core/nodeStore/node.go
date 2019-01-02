package nodeStore

import (
	//"crypto/ecdsa"
	//"crypto/x509"
	//"encoding/base64"
	"encoding/json"
	//"encoding/pem"
	//"fmt"
	"polarcloud/config"
	"polarcloud/core/utils"
	"time"
)

/*
	保存节点的id
	ip地址
	不同协议的端口
*/
type Node struct {
	// NodeId               *big.Int  //节点id的10进制字符串
	IdInfo  IdInfo `json:"idinfo"`  //节点id信息，id字符串以16进制显示
	IsSuper bool   `json:"issuper"` //是不是超级节点，超级节点有外网ip地址，可以为其他节点提供代理服务
	Addr    string `json:"addr"`    //外网ip地址
	TcpPort uint16 `json:"tcpport"` //TCP端口
	IsApp   bool   `json:"isapp"`   //是不是手机端节点
	//	UdpPort              uint16    `json:"udpport"` //UDP端口
	lastContactTimestamp time.Time //最后检查的时间戳
	// NodeIdShould         *big.Int  //影子id
	// Status               int       //节点状态，1：在线，2：正在查询中，3：下线
	// Out                  chan *Node //需要查询是否在线的节点
	// OverTime             time.Duration `1 * 60 * 60` //超时时间，单位为秒
	// SelectTime           time.Duration `5 * 60`      //查询时间，单位为秒
	// Key                  *rsa.PrivateKey //保存的公钥和私钥信息
}

func (this *Node) FlashOnlineTime() {
	this.lastContactTimestamp = time.Now()

}

func (this *Node) Marshal() []byte {
	nodeBs, err := json.Marshal(this)
	if err != nil {
		return nil
	}
	return nodeBs
}

func ParseNode(bs []byte) (*Node, error) {
	node := new(Node)
	err := json.Unmarshal(bs, node)
	//	fmt.Printf("dddd%+v %v", node, err)
	return node, err
}

//Id信息
type IdInfo struct {
	Id    *utils.Multihash `json:"id"`    //id，网络地址，临时地址，由updatetime字符串产生MD5值
	Puk   []byte           `json:"puk"`   //公钥
	Sign  []byte           `json:"sign"`  //签名
	Ctype string           `json:"ctype"` //签名方法 如ecdsa256 ecdsa512
}

//func (this *IdInfo) GetId() []byte {
//	return this.Id
//}

//func (this *IdInfo) GetBigIntId() *big.Int {
//	bigInt, _ := new(big.Int).SetString(this.Id, IdStrBit)
//	return bigInt
//}

/*
	解析一个idInfo
*/
func (this *IdInfo) Parse(code []byte) (err error) {
	err = json.Unmarshal(code, this)
	return
}

//将此节点id详细信息构建为标准code
func (this *IdInfo) JSON() []byte {
	str, _ := json.Marshal(this)
	return str
}

/*
	检查idInfo是否合法
	@return   true:合法;false:不合法;
*/
func CheckIdInfo(idInfo IdInfo) bool {
	//检查地址是否是公钥生成
	/*fmt.Println("ddddd", idInfo.Id.B58String(), idInfo.Puk, idInfo.Sign)
	bs := utils.BuildKeyToByte(config.Core_addr_puk_type, base64.StdEncoding.EncodeToString(idInfo.Puk))

	b, _ := pem.Decode([]byte(bs))
	pukItr, err := utils.ParsePubkey(b.Bytes)
	if err != nil {
		fmt.Println("111 err", err)
		return false
	}
	//puk := pukItr.(*ecdsa.PublicKey)
	pubkey, _ := utils.MarshalPubkey(pukItr)
	fmt.Println("xxxxx", pubkey)*/
	ok, _ := utils.Verify(idInfo.Puk, *idInfo.Id, idInfo.Sign)
	return ok

}

/*
	解析IdInfo得到纯id
*/
//func ParseId(idInfoStr string) (id []byte) {
//	idInfo := IdInfo{}
//	idInfo.Parse([]byte(idInfoStr))
//	return *idInfo.Id.GetId()
//}

func Parse(idInfoByte []byte) IdInfo {
	idInfo := IdInfo{}
	idInfo.Parse(idInfoByte)
	return idInfo
}

//userName      用户名，最大长度100
//email         email，最大长度100
//local         地址，最大长度100
//superNodeId   超级节点id，最大长度
//superNodeKey  超级节点密钥
//rerutn idInfo
//return err
func NewIdInfo(name, email, domain string) (idInfo IdInfo) {
	//	createTime := time.Now().Format("2006-01-02 15:04:05.999999999")
	//	hash := sha256.New()
	//	hash.Write([]byte(name + "#" + email + "#" + domain + "#" + superNodeId + "#" + createTime))
	//	md := hash.Sum(nil)
	//	mdStr := hex.EncodeToString(md)

	now := time.Now().Format("2006-01-02 15:04:05.999999999")
	//	hash := sha256.New()
	//	hash.Write([]byte(now))
	//	md := hash.Sum(nil)
	//	mdStr := hex.EncodeToString(md)
	idBs := utils.GetHashForDomain(now)
	idmhBs, _ := utils.Encode(idBs, config.HashCode)
	idMh := utils.Multihash(idmhBs)
	//	var domainStr utils.Multihash
	//	if domain != "" {
	//		domainStr = utils.GetHashForDomain(domain)
	//	}

	//	prk, err := GetKey()

	idInfo = IdInfo{
		Id: &idMh,
		//			Puk  []byte           `json:"puk"`  //公钥
		//	Sign []byte           `json:"sign"` //签名
		//		UpdateTime: now,
		//		CreateTime: "",
		//		Name:       name,
		//		Email:      email,
		//		Domain:     domain,
		//		DomainHash: &domainStr,
		//		SuperNodeId: superNodeId,
		// SuperNodeKey: superNodeKey,
	}
	return
}

//type idAddress struct {
//	//	local *sync.RWMutex
//	Id    []byte
//	idStr string
//}

//func (this *idAddress) GetId() (id *[]byte) {
//	//	this.local.RLock()
//	id = &this.Id
//	//	this.local.RUnlock()
//	return
//}
//func (this *idAddress) GetIdStr() (idstr string) {
//	//	this.local.RLock()
//	if this.idStr == "" {
//		this.idStr = hex.EncodeToString(this.Id)
//	}
//	idstr = this.idStr
//	//	this.local.RUnlock()
//	return
//}

///*
//	创建一个
//*/
//func NewIdAddress(id []byte) *idAddress {
//	ida := idAddress{
//		//		local: new(sync.RWMutex),
//		Id:    id,
//		idStr: hex.EncodeToString(id),
//	}
//	return &ida
//}

/*
	临时id
*/
type TempId struct {
	SuperPeerId *utils.Multihash `json:"superpeerid"` //更新在线时间
	PeerId      *utils.Multihash `json:"peerid"`      //更新在线时间
	UpdateTime  int64            `json:"updatetime"`  //更新在线时间
}

/*
	创建一个临时id
*/
func NewTempId(superId, peerId *utils.Multihash) *TempId {
	return &TempId{
		SuperPeerId: superId,
		PeerId:      peerId,
		UpdateTime:  time.Now().Unix(),
	}
}
