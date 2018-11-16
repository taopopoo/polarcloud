/*
* 安全协议
 */
package message_center

import (
	"fmt"
	"yunpan/core/engine"
	"yunpan/core/nodeStore"
	"yunpan/core/utils"
)

type Mess struct {
	*Message
}

func (this *Mess) Send(msgId uint64) bool {
	//TODO 这里对消息加密
	//this.BuildHash()
	if nodeStore.NodeSelf.IsSuper {
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 111111111111")
		}
		//收消息人就是自己
		if nodeStore.NodeSelf.IdInfo.Id.B58String() == this.Head.RecvId.B58String() {
			return false
		}
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 333333333333333")
		}
		//查找代理节点
		if _, ok := nodeStore.GetProxyNode(this.Head.RecvId.B58String()); ok {
			//发送给代理节点
			if session, ok := engine.GetSession(this.Head.RecvId.B58String()); ok {
				if msgId == debuf_msgid {
					fmt.Println("-=-=- 4444444444444")
				}
				session.Send(msgId, this.Head.JSON(), this.Body.JSON(), false)
				return true
			} else {
				fmt.Println("这个代理节点的链接断开了")
			}
			return true
		}
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 5555555")
		}

		//		fmt.Println(string(*this.Head.JSON()))
		var targetId *utils.Multihash
		if this.Head.Accurate {
			targetId = nodeStore.FindNearInSuper(this.Head.RecvSuperId, nil, false)
		} else {
			targetId = nodeStore.FindNearInSuper(this.Head.RecvSuperId, nil, true)
		}
		//		fmt.Println("本节点的其他超级节点", msgId, nodeStore.GetAllNodes(), targetId.B58String())
		if targetId == nil {
			fmt.Println("没有可用的邻居节点")
			return false
		}
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 666666666666")
		}
		//收消息人就是自己
		if nodeStore.NodeSelf.IdInfo.Id.B58String() == targetId.B58String() {
			if msgId == debuf_msgid {
				fmt.Println("-=-=- 777777777777777")
			}
			return false
		}

		if msgId == debuf_msgid {
			fmt.Println("-=-=- 88888888888888")
		}

		//转发出去
		if session, ok := engine.GetSession(targetId.B58String()); ok {
			session.Send(msgId, this.Head.JSON(), this.Body.JSON(), false)
			if msgId == debuf_msgid {
				fmt.Println("-=-=- 999999999999")
			}
			return true
		} else {
			fmt.Println("111 这个超级节点的链接断开了", msgId, targetId.B58String())
		}
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 101010101010101010101010")
		}
		return true

		//		return IsSendToOtherSuperToo(this.Head, this.Body.JSON(), msgId, nil)
	} else {
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 22222222222")
		}
		if nodeStore.SuperPeerId == nil {
			fmt.Println("没有可用的超级节点")
			return false
		}
		if session, ok := engine.GetSession(nodeStore.SuperPeerId.B58String()); ok {
			session.Send(msgId, this.Head.JSON(), this.Body.JSON(), false)
			return true
		} else {
			fmt.Println("超级节点的session未找到")
		}
		return true
	}
}
