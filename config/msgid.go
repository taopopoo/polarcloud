package config

const (
	//---------------- base --------------------------
	//	MSGID_Text = 101 //显示文本消息

	MSGID_checkNodeOnline       = 110 //检查节点是否在线
	MSGID_checkNodeOnline_recv  = 111 //检查节点是否在线_返回
	MSGID_TextMsg               = 112 //接收文本消息
	MSGID_getNearSuperIP        = 113 //从邻居节点得到自己的逻辑节点
	MSGID_getNearSuperIP_recv   = 114 //从邻居节点得到自己的逻辑节点_返回
	MSGID_multicast_online_recv = 122 //接收节点上线广播
	MSGID_ask_close_conn_recv   = 128 //询问关闭连接

	//---------------- name 域名模块 --------------------------
	MSGID_register_name        = 102 //注册一个域名
	MSGID_register_name_recv   = 103 //注册一个域名_返回
	MSGID_build_name           = 104 //构建一个域名
	MSGID_build_name_recv      = 105 //构建一个域名_返回
	MSGID_check_temp_name      = 106 //检查刚构建的域名是否成功
	MSGID_check_temp_name_recv = 107 //检查刚构建的域名是否成功 返回
	//	MSGID_findDomain               = 108 //查找这个域名是否存在
	//	MSGID_recv_domain              = 109 //返回这个域名是否存在
	MSGID_find_name                = 115 //查找一个域名的地址
	MSGID_find_name_recv           = 116 //查找一个域名的地址 返回
	MSGID_name_add_address_recv    = 117 //收到域名添加新地址
	MSGID_name_sync_multicast_recv = 118 //接收需要同步的域名广播

	MSGID_ROOT_register_name    = 123 //申请注册一个域名
	MSGID_ROOT_RECV_create_name = 124 //创建一个域名

	//---------------- 公钥 模块 --------------------------
	MSGID_key_sync_multicast_recv = 119 //接收需要同步的公钥广播
	MSGID_key_find_keyname        = 120 //查找公钥对应的域名
	MSGID_key_find_keyname_recv   = 121 //查找公钥对应的域名 返回
	MSGID_ROOT_RECV_save_key_name = 125 //接收保存的公钥key对应的域名

	//---------------- web 模块 --------------------------
	MSGID_http_request  = 126 //http请求
	MSGID_http_response = 127 //http返回

	//---------------- wallet 模块 --------------------------
	MSGID_multicast_vote_recv   = 200 //接收见证人投票广播
	MSGID_multicast_blockhead   = 201 //接收区块头广播
	MSGID_heightBlock           = 202 //查询邻居节点区块高度
	MSGID_heightBlock_recv      = 203 //查询邻居节点区块高度_返回
	MSGID_getBlockHead          = 204 //查询邻居节点的起始区块头
	MSGID_getBlockHead_recv     = 205 //查询邻居节点的起始区块头_返回
	MSGID_getTransaction        = 206 //查询交易
	MSGID_getTransaction_recv   = 207 //查询交易_返回
	MSGID_multicast_transaction = 208 //接收交易广播
)
