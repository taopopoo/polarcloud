package nodeStore

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"polarcloud/config"
	"polarcloud/core/engine"
	"polarcloud/core/utils"
	"strconv"
	"sync"
	"time"
)

var (
	NodeSelf         *Node                               = new(Node)                         //保存自己的id信息和ip地址和端口号
	Nodes                                                = new(sync.Map)                     //超级节点，id字符串为键 key:string,value:*Node
	OutFindNode      chan *utils.Multihash               = make(chan *utils.Multihash, 1000) //需要查询的逻辑节点
	Groups           *NodeGroup                          = NewNodeGroup()                    //组
	NodeIdLevel      uint                                = 256                               //节点id长度
	Proxys                                               = new(sync.Map)                     //被代理的节点，id字符串为键 key:string,value:*Node
	SuperPeerId      *utils.Multihash                                                        //超级节点名称
	OutCloseConnName = make(chan *utils.Multihash, 1000)                                     //废弃的nodeid，需要询问是否关闭
	once             sync.Once
	Key              *ECCKey
)

func init() {
	//加载本地网络id
	idinfo, err := BuildIdinfo()
	if err != nil {
		panic(err)
	}
	NodeSelf.IdInfo = *idinfo
	fmt.Println("id =", idinfo.Id.B58String())

}

/*
	加载本地私钥生成idinfo
*/
func BuildIdinfo() (*IdInfo, error) {
	var err error
	Key, err = GetKeyPair() //GetKey()
	if err != nil {
		return nil, err
	}
	idinfo := new(IdInfo)
	//	fmt.Println("加载的本地公钥\n", Key.GetPukStrings())
	idinfo.Puk, err = Key.GetPukBytes() //  base64.StdEncoding.DecodeString(Key.GetPukStrings())
	if err != nil {
		return nil, err
	}
	//	fmt.Println(string(idinfo.Puk))

	//生成网络地址
	hashOne := utils.Hash_SHA3_256([]byte(Key.GetPukStrings()))
	hashTwo := utils.Hash_SHA3_256(hashOne)
	bs, err := utils.Encode(hashTwo, config.HashCode)
	if err != nil {
		return nil, err
	}
	id := utils.Multihash(bs)
	idinfo.Id = &id

	//给地址签名
	signText, err := utils.Sign(Key.GetPrk(), *idinfo.Id)
	if err != nil {
		return nil, err
	}
	idinfo.Sign = *signText
	//	fmt.Println("生成的签名:", signText)

	//	fmt.Println("是否验证通过", CheckIdInfo(*idinfo))
	return idinfo, nil
}

//超级节点之间查询的间隔时间
//var SpacingInterval time.Duration = time.Second * 30

//id字符串格式为16进制字符串
//var IdStrBit int = 16

func InitNodeStore() {
	once.Do(run)
}

/*
	定期检查所有节点状态
	一个小时查询所有逻辑节点，非超级节点查询相邻节点
	5分钟清理一次已经不在线的节点
*/
func run() {
	//	fmt.Println("启动循环查询逻辑节点")
	go func() {

		//		bt := utils.NewBackoffTimer(config.Time_find_network_peer...)
		//		//查询和自己相关的逻辑节点
		//		for {
		//			ids := getNodeNetworkNum()
		//			//			if !NodeSelf.IsSuper {
		//			//				ids = ids[:3]
		//			//			}
		//			//			ids = ids[:8]
		//			for _, idOne := range ids {
		//				OutFindNode <- idOne
		//				time.Sleep(time.Second * 1)
		//			}
		//			//			fmt.Println("完成一轮查找")
		//			bt.Wait()
		//		}
	}()
}

//func initIds() {
//	if netNodes == nil {
//		netNodes = NewIds(NodeSelf.IdInfo.Id, NodeIdLevel)
//	}
//}

//添加一个代理节点
func AddProxyNode(node *Node) {
	Proxys.Store(node.IdInfo.Id.B58String(), node)
}

//得到一个代理节点
func GetProxyNode(id string) (node *Node, ok bool) {
	var v interface{}
	v, ok = Proxys.Load(id)
	if v == nil {
		return nil, ok
	}
	node = v.(*Node)
	return
}

/*
	获得所有代理节点
*/
func GetProxyAll() []string {
	ids := make([]string, 0)
	Proxys.Range(func(key, value interface{}) bool {
		ids = append(ids, key.(string))
		return true
	})
	return ids
}

/*
	添加一个超级节点
	检查这个节点是否是自己的逻辑节点，如果是，则保存
	不保存自己
*/
func AddNode(node *Node) {
	//	initIds()

	//	fmt.Println("添加一个节点", new(big.Int).SetBytes(node.IdInfo.Id.Data()).Int64())

	//是本身节点不添加
	if node.IdInfo.Id.B58String() == NodeSelf.IdInfo.Id.B58String() {
		return
	}

	idm := NewIds(NodeSelf.IdInfo.Id, NodeIdLevel)
	ids := GetAllNodes()
	for _, one := range ids {
		idm.AddId(one)
	}
	//	idm.AddId(node.IdInfo.Id)

	ok, removeIDs := idm.AddId(node.IdInfo.Id)
	if ok {
		//		fmt.Println("添加成功", new(big.Int).SetBytes(node.IdInfo.Id.Data()).Int64())
		node.lastContactTimestamp = time.Now()
		Nodes.Store(node.IdInfo.Id.B58String(), node)
		//修改超级节点，普通节点经常切换影响网络
		SuperPeerId = idm.GetIndex(0)

		//删除被替换的id
		for _, one := range removeIDs {
			//			idOne := hex.EncodeToString(one)
			//			delete(Nodes, idOne)
			//			OutCloseConnName <- idOne

			Nodes.Delete(one.B58String())
			OutCloseConnName <- one
		}
	}
	//	fmt.Println("添加一个node", node.IdInfo.Id.B58String())
	return

}

/*
	删除一个节点，包括超级节点和代理节点
*/
func DelNode(id *utils.Multihash) {
	//	initIds()
	Nodes.Delete(id.B58String())
	engine.RemoveSession(id.B58String())
	Proxys.Delete(id.B58String())
}

/*
	通过id查找一个节点
*/
func FindNode(id *utils.Multihash) *Node {
	v, ok := Nodes.Load(id.B58String())
	if ok {
		return v.(*Node)
	}
	v, ok = Proxys.Load(id.B58String())
	if ok {
		return v.(*Node)
	}
	return nil
}

/*
	在超级节点中找到最近的节点，不包括代理节点
	@nodeId         要查找的节点
	@outId          排除一个节点
	@includeSelf    是否包括自己
	@return         查找到的节点id，可能为空
*/
func FindNearInSuper(nodeId, outId *utils.Multihash, includeSelf bool) *utils.Multihash {
	kl := NewKademlia()
	if includeSelf {
		kl.Add(new(big.Int).SetBytes(NodeSelf.IdInfo.Id.Data()))
		//		idbs, _ := utils.Encode(NodeSelf.IdInfo.Id.Data(), config.HashCode)
		//		idmh := utils.Multihash(idbs)
		//		fmt.Println("+几几", idmh.B58String())
	}
	outIdStr := ""
	if outId != nil {
		outIdStr = outId.B58String()
	}
	Nodes.Range(func(k, v interface{}) bool {
		if k.(string) == outIdStr {
			return true
		}
		value := v.(*Node)
		//		fmt.Println("+", value.IdInfo.Id.B58String())
		kl.Add(new(big.Int).SetBytes(value.IdInfo.Id.Data()))
		return true
	})

	targetIds := kl.Get(new(big.Int).SetBytes(nodeId.Data()))
	if len(targetIds) == 0 {
		return nil
	}
	//	for _, one := range targetIds {
	//		idbs, _ := utils.Encode(one.Bytes(), config.HashCode)
	//		idmh := utils.Multihash(idbs)
	//		fmt.Println("-", idmh.B58String())
	//	}
	targetId := targetIds[0]
	if targetId == nil {
		return nil
	}
	bs, _ := utils.Encode(targetId.Bytes(), config.HashCode)
	mh := utils.Multihash(bs)
	//	fmt.Println("搜索到的id", mh.B58String())
	return &mh
}

//在节点中找到最近的节点，包括代理节点
func FindNearNodeId(nodeId, outId *utils.Multihash, includeSelf bool) *utils.Multihash {
	kl := NewKademlia()
	if includeSelf {
		kl.Add(new(big.Int).SetBytes(NodeSelf.IdInfo.Id.Data()))
		//		idbs, _ := utils.Encode(NodeSelf.IdInfo.Id.Data(), config.HashCode)
		//		idmh := utils.Multihash(idbs)
		//		fmt.Println("+几几", idmh.B58String())
	}
	outIdStr := ""
	if outId != nil {
		outIdStr = outId.B58String()
	}
	Nodes.Range(func(k, v interface{}) bool {
		if k.(string) == outIdStr {
			return true
		}
		value := v.(*Node)
		//		fmt.Println("+", value.IdInfo.Id.B58String())
		kl.Add(new(big.Int).SetBytes(value.IdInfo.Id.Data()))
		return true
	})
	//代理节点
	Proxys.Range(func(k, v interface{}) bool {
		if k.(string) == outIdStr {
			return true
		}
		value := v.(*Node)
		//过滤APP节点
		if value.IsApp {
			return true
		}
		//		fmt.Println("+", value.IdInfo.Id.B58String())
		kl.Add(new(big.Int).SetBytes(value.IdInfo.Id.Data()))
		return true
	})

	targetIds := kl.Get(new(big.Int).SetBytes(nodeId.Data()))
	if len(targetIds) == 0 {
		return nil
	}
	//	for _, one := range targetIds {
	//		idbs, _ := utils.Encode(one.Bytes(), config.HashCode)
	//		idmh := utils.Multihash(idbs)
	//		fmt.Println("-", idmh.B58String())
	//	}
	targetId := targetIds[0]
	if targetId == nil {
		return nil
	}
	bs, _ := utils.Encode(targetId.Bytes(), config.HashCode)
	mh := utils.Multihash(bs)
	//	fmt.Println("搜索到的id", mh.B58String())
	return &mh
}

/*
	根据节点id得到一个距离最短节点的信息，不包括代理节点
	@nodeId         要查找的节点
	@includeSelf    是否包括自己
	@outId          排除一个节点
	@return         查找到的节点id，可能为空
*/
//func Get(nodeId string, includeSelf bool, outId string) *Node {
//	nodeIdInt, b := new(big.Int).SetString(nodeId, IdStrBit)
//	if !b {
//		fmt.Println("节点id格式不正确，应该为十六进制字符串:")
//		fmt.Println(nodeId)
//		return nil
//	}
//	kl := NewKademlia()
//	if includeSelf {
//		//		temp := new(big.Int).SetBytes(Root.IdInfo.Id)
//		kl.add(new(big.Int).SetBytes(NodeSelf.IdInfo.Id))
//	}
//	for key, value := range Nodes {
//		if outId != "" && key == outId {
//			continue
//		}
//		kl.add(new(big.Int).SetBytes(value.IdInfo.Id))
//	}
//	// TODO 不安全访问
//	targetId := kl.get(nodeIdInt)[0]

//	if targetId == nil {
//		return nil
//	}
//	if hex.EncodeToString(targetId.Bytes()) == hex.EncodeToString(NodeSelf.IdInfo.Id) {
//		return NodeSelf
//	}
//	return Nodes[hex.EncodeToString(targetId.Bytes())]
//}

//得到所有的节点，不包括本节点，也不包括代理节点
func GetAllNodes() []*utils.Multihash {
	ids := make([]*utils.Multihash, 0)
	Nodes.Range(func(k, v interface{}) bool {
		value := v.(*Node)
		ids = append(ids, value.IdInfo.Id)
		return true
	})
	return ids
}

/*
	获得本机所有逻辑节点的ip地址
*/
func GetSuperNodeIps() (ips []string) {
	ips = make([]string, 0)
	Nodes.Range(func(k, v interface{}) bool {
		value := v.(*Node)
		ips = append(ips, value.Addr+":"+strconv.Itoa(int(value.TcpPort)))
		return true
	})

	return
}

/*
	检查节点是否是本节点的逻辑节点
	只检查，不保存
*/
func CheckNeedNode(nodeId *utils.Multihash) (isNeed bool) {
	/*
		1.找到已有节点中与本节点最近的节点
		2.计算两个节点是否在同一个网络
		3.若在同一个网络，计算谁的值最小
	*/
	if len(GetAllNodes()) == 0 {
		return true
	}
	//是本身节点不添加
	//	if hex.EncodeToString(nodeId) == NodeSelf.IdInfo.Id.GetIdStr() {
	if nodeId.B58String() == NodeSelf.IdInfo.Id.B58String() {
		//		fmt.Println("2不添加")
		return false
	}

	ids := NewIds(NodeSelf.IdInfo.Id, NodeIdLevel)
	for _, one := range GetAllNodes() {
		ids.AddId(one)
	}
	ok, _ := ids.AddId(nodeId)
	return ok

	//	//	consHash := NewKademlia()
	//	//	nodesLock.RLock()
	//	//	for _, value := range Nodes {
	//	//		consHash.add(new(big.Int).SetBytes(value.IdInfo.Id))
	//	//	}
	//	//	nodesLock.RUnlock()

	//	netIDs := getNodeNetworkNum()
	//	newNetNodes := [256]*utils.Multihash{}
	//	//		nodesLock.Lock()
	//	for i, one := range netNodes {
	//		newNetNodes[i] = one
	//	}
	//	//	copy(newNetNodes, netNodes)
	//	//		nodesLock.Unlock()

	//	for i, one := range newNetNodes {
	//		kl := NewKademlia()
	//		kl.Add(new(big.Int).SetBytes(one.Data()))
	//		kl.Add(new(big.Int).SetBytes(nodeId.Data()))
	//		nearId := kl.Get(new(big.Int).SetBytes(netIDs[i].Data()))
	//		if hex.EncodeToString(one.Data()) == hex.EncodeToString(nearId[0].Bytes()) {
	//			continue
	//		}
	//		return true
	//	}
	//	return false

}

type LogicNumBuider struct {
	lock  *sync.RWMutex
	id    *utils.Multihash
	level uint
	idStr string
	ids   []*utils.Multihash
}

/*
	得到每个节点网络的网络号，不包括本节点
	@id        *utils.Multihash    要计算的id
	@level     int                 深度
*/
func (this *LogicNumBuider) GetNodeNetworkNum() []*utils.Multihash {

	this.lock.RLock()
	if this.idStr != "" && this.idStr == this.id.B58String() {
		this.lock.RUnlock()
		return this.ids
	}
	this.lock.RUnlock()

	this.lock.Lock()
	this.idStr = this.id.B58String()

	root := new(big.Int).SetBytes(this.id.Data())

	this.ids = make([]*utils.Multihash, 0)
	for i := 0; i < int(this.level); i++ {
		//---------------------------------
		//将后面的i位置零
		//---------------------------------
		//		startInt := new(big.Int).Lsh(new(big.Int).Rsh(root, uint(i)), uint(i))
		//---------------------------------
		//第i位取反
		//---------------------------------
		networkNum := new(big.Int).Xor(root, new(big.Int).Lsh(big.NewInt(1), uint(i)))

		bs, err := utils.Encode(networkNum.Bytes(), config.HashCode)
		if err != nil {
			fmt.Println("格式化muhash错误")
			continue
		}
		mhbs := utils.Multihash(bs)
		this.ids = append(this.ids, &mhbs)
	}
	this.lock.Unlock()

	//	NodeSelf.IdInfo = NewIdInfo("", "", "tao")
	//	//	idMH, _ := utils.FromB58String(one)
	//	NodeSelf.IdInfo.Id = this.id

	//	ids := getNodeNetworkNum()
	//	for i, one := range this.ids {
	//		if one.B58String() != ids[i].B58String() {
	//			fmt.Println("我们不一样，每个人都有不同的境遇")
	//			break
	//		}
	//	}

	return this.ids
}

func NewLogicNumBuider(id *utils.Multihash, level uint) *LogicNumBuider {
	return &LogicNumBuider{
		lock:  new(sync.RWMutex),
		id:    id,
		level: level,
		idStr: "",
		ids:   make([]*utils.Multihash, 0),
	}
}

var (
	networkIDsLock   = new(sync.RWMutex)
	nodeNetworkIDStr = ""
	networkIDs       []*utils.Multihash
)

//得到每个节点网络的网络号，不包括本节点
func getNodeNetworkNum() []*utils.Multihash {
	networkIDsLock.RLock()
	if nodeNetworkIDStr != "" && nodeNetworkIDStr == NodeSelf.IdInfo.Id.B58String() {
		networkIDsLock.RUnlock()
		return networkIDs
	}
	networkIDsLock.RUnlock()

	// rootInt, _ := new(big.Int).SetString(, IdStrBit)
	networkIDsLock.Lock()
	nodeNetworkIDStr = NodeSelf.IdInfo.Id.B58String()

	root := new(big.Int).SetBytes(NodeSelf.IdInfo.Id.Data())

	networkIDs = make([]*utils.Multihash, 0)
	for i := 0; i < int(NodeIdLevel); i++ {
		//---------------------------------
		//将后面的i位置零
		//---------------------------------
		//		startInt := new(big.Int).Lsh(new(big.Int).Rsh(root, uint(i)), uint(i))
		//---------------------------------
		//第i位取反
		//---------------------------------
		networkNum := new(big.Int).Xor(root, new(big.Int).Lsh(big.NewInt(1), uint(i)))

		bs, err := utils.Encode(networkNum.Bytes(), config.HashCode)
		if err != nil {
			fmt.Println("格式化muhash错误")
			continue
		}
		mhbs := utils.Multihash(bs)
		networkIDs = append(networkIDs, &mhbs)
	}
	networkIDsLock.Unlock()
	return networkIDs
}

/*
	获得一个节点更远的节点中，比自己更远的节点
*/
func GetIdsForFar(id *utils.Multihash) []*utils.Multihash {
	//计算来源的逻辑节点地址
	kl := NewKademlia()
	kl.Add(new(big.Int).SetBytes(NodeSelf.IdInfo.Id.Data()))
	kl.Add(new(big.Int).SetBytes(id.Data()))

	Nodes.Range(func(k, v interface{}) bool {
		value := v.(*Node)
		kl.Add(new(big.Int).SetBytes(value.IdInfo.Id.Data()))
		return true
	})

	list := kl.Get(new(big.Int).SetBytes(id.Data()))

	out := make([]*utils.Multihash, 0)
	find := false
	for _, one := range list {

		if hex.EncodeToString(one.Bytes()) == hex.EncodeToString(NodeSelf.IdInfo.Id.Data()) {
			find = true
		} else {
			if find {
				bs, err := utils.Encode(one.Bytes(), config.HashCode)
				if err != nil {
					fmt.Println("编码失败")
					continue
				}
				mh := utils.Multihash(bs)
				out = append(out, &mh)
			}
		}

	}

	return out
}
