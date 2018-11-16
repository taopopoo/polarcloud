package core

import (
	"fmt"
	"net"
	//	"runtime"
	"strconv"
	"time"
	gconfig "yunpan/config"
	addrm "yunpan/core/addr_manager"
	"yunpan/core/config"
	"yunpan/core/engine"
	msg "yunpan/core/message_center"
	"yunpan/core/nodeStore"
	_ "yunpan/core/persistence"
	"yunpan/core/utils"
)

var (
	//	privateKey  *rsa.PrivateKey
	isStartCore = false
)

func init() {

	go startUp()
	go read()
	go getNearSuperIP()
}

/*
	有新地址就连接到网络中去
*/
func startUp() {
	//	fmt.Println("这个方法究竟有没有执行啊")
	//	one := make(chan string, 0)
	//	addrm.AddSubscribe(one)
	for addr := range addrm.SubscribesChan {
		//		fmt.Println("这个方法究竟有没有执行啊222")
		//		engine.Log.Debug("开始接收新地址")
		//		fmt.Println("这个方法究竟有没有执行啊333")
		//接收到超级节点地址消息
		//		addr := <-addrm.SubscribesChan
		engine.Log.Debug("有新的地址 %s", addr)
		//		fmt.Println("有新的地址 %s", addr)
		host, portStr, _ := net.SplitHostPort(addr)
		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}
		go connectNet(host, uint16(port))

	}
}

/*
	初始化消息引擎
*/
//func InitEngine() error {
//	engine.InitEngine(string(nodeStore.NodeSelf.IdInfo.Build()))
//	engine.SetAuth(new(Auth))
//	engine.SetCloseCallback(closeConnCallback)
//	msg.InitMsgRouter()

//	return nil
//}

/*
	启动消息服务器
*/
func StartEngine() bool {
	defer func() {
		time.Sleep(time.Second * 5)
	}()
	engine.InitEngine(string(nodeStore.NodeSelf.IdInfo.JSON()))
	engine.SetAuth(new(Auth))
	engine.SetCloseCallback(closeConnCallback)
	msg.RegisterCoreMsg()

	//	engine.ListenByListener(config.TCPListener, true)
	//占用本机一个端口
	var err error
	for i := 0; i < 100; i++ {
		//		fmt.Println(runtime.GOARCH, runtime.GOOS)
		//		if runtime.GOOS == "windows" {
		//			err = engine.Listen(config.Init_LocalIP, uint32(config.Init_LocalPort+uint16(i)), true)
		//		} else {
		//			err = engine.Listen("0.0.0.0", uint32(config.Init_LocalPort+uint16(i)), true)
		//		}
		err = engine.Listen("0.0.0.0", uint32(config.Init_LocalPort+uint16(i)), true)
		if err != nil {
			continue
		} else {
			//得到本机可用端口
			config.Init_LocalPort = config.Init_LocalPort + uint16(i)
			if !config.Init_IsMapping {
				nodeStore.NodeSelf.TcpPort = config.Init_LocalPort
			}

			//加载超级节点ip地址
			go addrm.LoadAddrForAll()
			return true
		}
	}
	return false
}

func StartService() {
	//启动核心组件
	StartUpCore()
}

/*
	启动核心组件
*/
func StartUpCore() {

	//	if len(*nodeStore.NodeSelf.IdInfo.Id.GetId()) == 0 {
	//		//		GetId()
	//		if len(*nodeStore.NodeSelf.IdInfo.Id.GetId()) == 0 {
	//			return
	//		}
	//	}
	if nodeStore.NodeSelf.IdInfo.Id == nil {
		return
	}
	engine.Log.Debug("启动服务器核心组件")
	engine.Log.Debug("本机id为：\n%s", nodeStore.NodeSelf.IdInfo.Id)

	isSuperPeer := config.CheckIsSuperPeer()
	//是超级节点
	node := &nodeStore.Node{
		IdInfo:  nodeStore.NodeSelf.IdInfo,
		IsSuper: isSuperPeer, //是否是超级节点
		//		UdpPort: 0,
	}
	addr, port := config.GetHost()
	node.Addr = addr
	node.TcpPort = uint16(port)

	/*
		启动消息服务器
	*/
	engine.InitEngine(string(nodeStore.NodeSelf.IdInfo.JSON()))
	/*
		生成密钥文件
	*/
	//	var err error
	//	//生成密钥
	//	privateKey, err = rsa.GenerateKey(rand.Reader, 512)
	//	if err != nil {
	//		fmt.Println("生成密钥错误", err.Error())
	//		return
	//	}
	/*
		启动分布式哈希表
	*/
	//	nodeStore.InitNodeStore(node)
	/*
		设置关闭连接回调函数后监听
	*/
	engine.SetAuth(new(Auth))
	engine.SetCloseCallback(closeConnCallback)
	//	engine.ListenByListener(config.TCPListener, true)
	//	engine.Listen(config.TCPListener)
	//自己是超级节点就把自己添加到超级节点地址列表中去
	if isSuperPeer {
		addrm.AddSuperPeerAddr(addr + ":" + strconv.Itoa(int(port)))
	}

	isStartCore = true

	/*
		连接到超级节点
	*/
	ip, port, err := addrm.GetSuperAddrOne(false)
	if err == nil {
		fmt.Println("准备连接到网络中去", ip, port)
		connectNet(ip, port)
	}

	//	go read()

}

/*
	链接到网络中去
*/
func connectNet(ip string, port uint16) {
	//	if !isStartCore {
	//		StartUpCore()
	//		//启动失败
	//		if !isStartCore {
	//			return
	//		}
	//	}
	//	engine.Log.Debug("链接到网络中去")

	session, err := engine.AddClientConn(ip, uint32(port), false)
	if err != nil {
		//		fmt.Println("连接失败", err)
		return
	}
	//TODO 只有最近的节点才能作为超级节点
	//	superPeerIdStr := session.GetName()
	//	superPeerId, _ := hex.DecodeString(superPeerIdStr)
	//	nodeStore.SuperPeerId = nodeStore.NewIdAddress(superPeerId)

	mh, err := utils.FromB58String(session.GetName())
	if err != nil {
		//		fmt.Println("连接失败", err)
		return
	}
	nodeStore.SuperPeerId = &mh

	engine.Log.Debug("超级节点为: %s", nodeStore.SuperPeerId.B58String())
	config.IsOnline = true
	// config.SuperNodeIp = ip
	// config.SuperNodePort = port
	//给目标机器发送自己的名片
	//	introduceSelf()

	//获取邻居节点id地址
	//	nearId := nodeStore.GetNearId(nodeStore.NodeSelf.IdInfo.Id)

	fmt.Println("本节点是否是超级节点", nodeStore.NodeSelf.IsSuper)

	//	mhead := msg.NewMessageHead(nearId, nearId, false)
	//	mbody := msg.NewMessageBody(nil, "", nil, 0)
	//	message := msg.NewMessage(mhead, mbody)
	//	message.BuildHash()

	//	var message msg.Message
	//	//判断用代理方式查找，还是直接查找
	//	if nodeStore.NodeSelf.IsSuper {
	//		//直接查找最近的超级节点
	//		message = msg.Message{
	//			RecvId:        nearId,
	//			RecvSuperId:   nearId,                         //接收者的超级节点id
	//			CreateTime:    utils.TimeFormatToNanosecond(), //消息创建时间unix
	//			SenderSuperId: nodeStore.NodeSelf.IdInfo.Id,   //发送者超级节点id
	//			Sender:        nodeStore.NodeSelf.IdInfo.Id,
	//			Accurate:      false,
	//		}
	//	} else {
	//		//用代理方式查找最近的超级节点
	//		message = msg.Message{
	//			RecvId:        nearId,
	//			RecvSuperId:   nearId,                         //接收者的超级节点id
	//			CreateTime:    utils.TimeFormatToNanosecond(), //消息创建时间unix
	//			Sender:        nodeStore.NodeSelf.IdInfo.Id,   //发送者id
	//			SenderSuperId: nodeStore.SuperPeerId,          //发送者超级节点id
	//			Accurate:      false,
	//		}
	//	}
	//	message.BuildHash()

	//	resultBytes, _ := json.Marshal(message)

	//	session.Send(gconfig.MSGID_checkNodeOnline, mhead.JSON(), mbody.JSON(), false)

	//	go findRootName()

}

/*
	关闭服务器回调函数
*/
func ShutdownCallback() {
	//回收映射的端口
	Reclaim()
	// addrm.CloseBroadcastServer()
	fmt.Println("Close over")
}

/*
	一个连接断开后的回调方法
*/
func closeConnCallback(name string) {
	engine.Log.Debug("节点下线 %s", name)
	//	fmt.Println("节点下线", name)

	mh, _ := utils.FromB58String(name)

	nodeStore.DelNode(&mh)
	//nodeStore.DelProxyNode(name)
	if nodeStore.SuperPeerId == nil {
		return
	}
	if name == nodeStore.SuperPeerId.B58String() {
		nearId := nodeStore.FindNearInSuper(nodeStore.NodeSelf.IdInfo.Id, nil, false)
		if nearId == nil {
			//该节点没有邻居节点，已经离开了网络，没有连入网站中。
			fmt.Println("该节点没有邻居节点，已经离开了网络，没有连入网站中")
		} else {
			nodeStore.SuperPeerId = nearId
			//			nodeStore.SuperPeerIdStr = hex.EncodeToString(nearId)

		}
		//		if nodeStore.NodeSelf.IsSuper {
		//			//			nodeStore.Get(nodeStore.NodeSelf.IdInfo.Id, false, []byte{})
		//		} else {

		//		}
	}

	//TODO 先判断对方是否真的下线，再广播对方下线

	//	node := nodeStore.Get(nodeStore.ParseId(name), false, "")
	//	// fmt.Println("节点下线", node)

	//	// engine.Log.Debug("目前超级节点是 %s", nodeStore.ParseId(nodeStore.SuperName))

	//	if name == nodeStore.SuperName {
	//		fmt.Println("超级节点断开连接:", name)
	//		targetNode := nodeStore.Get(nodeStore.Root.IdInfo.GetId(), false, nodeStore.Root.IdInfo.GetId())
	//		if targetNode == nil {
	//			return
	//		}

	//		if config.Init_role == config.C_role_client {
	//			session, err := engine.AddClientConn(targetNode.Addr, uint32(targetNode.TcpPort), false)
	//			if err != nil {
	//				fmt.Println("连接节点错误", err)
	//				return
	//			}
	//			nodeStore.SuperName = session.GetName()
	//		} else {
	//			session, _ := engine.GetSession(string(targetNode.IdInfo.Build()))
	//			nodeStore.SuperName = session.GetName()
	//		}
	//		return
	//	}
	//	// session, ok := engine.GetController().GetSession(name)
	//	// if err != nil {
	//	// 	fmt.Println("客户端离线，但找不到这个session")
	//	// }

	//	if node != nil && !node.IsSuper {
	//		fmt.Println("自己代理的节点下线:", nodeStore.ParseId(name))
	//	}
	//	nodeStore.DelNode(nodeStore.ParseId(name))
}

/*
	处理查找节点的请求
	定期查询已知节点是否在线，更新节点信息
*/
func read() {
	for {
		nodeIdStr := <-nodeStore.OutFindNode

		mhead := msg.NewMessageHead(nodeIdStr, nodeIdStr, true)
		mbody := msg.NewMessageBody(nil, "", nil, 0)
		message := msg.NewMessage(mhead, mbody)
		message.Send(gconfig.MSGID_checkNodeOnline)

	}
}

/*
	定时获得相邻节点的超级节点ip地址
*/
func getNearSuperIP() {
	for {
		for _, key := range nodeStore.GetAllNodes() {
			//直接查找最近的超级节点
			mhead := msg.NewMessageHead(key, key, false)
			mbody := msg.NewMessageBody(nil, "", nil, 0)
			message := msg.NewMessage(mhead, mbody)
			message.BuildHash()

			session, ok := engine.GetSession(key.B58String())
			if ok {
				//				fmt.Println("给这个session发送消息成功", key.B58String())
				session.Send(gconfig.MSGID_getNearSuperIP, mhead.JSON(), mbody.JSON(), false)
				time.Sleep(time.Second * 1)
			}
		}
		//		fmt.Println("完成一轮查找邻居节点地址")
		time.Sleep(time.Second * config.Time_getNear_super_ip)
	}
}
