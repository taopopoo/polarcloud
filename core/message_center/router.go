package message_center

//import (
//	"encoding/json"
//	"fmt"
//	"polarcloud/core/engine"
//	"sync"
//	"time"
//)

////******************************************
////  注册所有消息
////      1.提供所有消息注册
////      2.保存所有消息编码和回调方法的映射
////      3.通过消息编码获得回调方法
////******************************************

//type MsgHandler func(c engine.Controller, packet engine.Packet, msg *Message) (bool, string)

//var handlersMapping = make(map[int]MsgHandler)
//var router_lock = new(sync.RWMutex)

//func addRouter(msgId int, handler MsgHandler) {
//	router_lock.Lock()
//	defer router_lock.Unlock()
//	handlersMapping[msgId] = handler
//}

//func getHandler(msgId int) MsgHandler {
//	router_lock.Lock()
//	defer router_lock.Unlock()
//	handler := handlersMapping[msgId]
//	return handler
//}

///*
//	消息分发程序
//*/
//func handlerProcess(c engine.Controller, packet engine.Packet) {
//	defer func() {
//		if err := recover(); err != nil {
//			e, ok := err.(error)
//			if ok {
//				fmt.Println("网络库：", e.Error())
//			}
//		}
//	}()
//	msg := new(Message)
//	err := json.Unmarshal(packet.Data, msg)
//	if err != nil {
//		fmt.Println(err)
//		panic(err)
//	}
//	handler := getHandler(msg.ProtoId)
//	if handler == nil {
//		fmt.Println("消息中心：未注册的消息编号-", packet.MsgID)
//		return
//	}
//	ok, str := handler(c, packet, msg)
//	fire(msg, ok, str)
//}

////******************************************
////  消息管理中心
////      1.消息超时机制
////******************************************
//var timeoutMapping = make(map[string]*Pipe) //
//var timeoutLock *sync.RWMutex = new(sync.RWMutex)

///*
//	添加一个映射
//	@sendProtoId   发送消息协议号id
//	@recvProtoId   接收消息协议号id
//	@ticker        消息发送时间unix
//	@id            发送给目标id
//*/
//func addTimeoutMapping(hash string, pipe *Pipe) {
//	timeoutLock.Lock()
//	defer timeoutLock.Unlock()
//	timeoutMapping[hash] = pipe
//}

///*
//	删除一个映射
//*/
//func removeTimeoutMapping(hash string) {
//	timeoutLock.Lock()
//	defer timeoutLock.Unlock()
//	delete(timeoutMapping, hash)
//}

///*
//	得到一个映射
//*/
//func getTimeoutMapping(hash string) (pipe *Pipe, ok bool) {
//	timeoutLock.Lock()
//	defer timeoutLock.Unlock()
//	pipe, ok = timeoutMapping[hash]
//	return
//}

//type Pipe struct {
//	hash string      //
//	C    chan string //
//	Done chan bool   //完成这个管道的消息队列
//}

///*
//	设置超时
//*/
//func (this *Pipe) runTimeout(timeout time.Duration) {
//	select {
//	case <-this.Done:
//		//完成任务
//	case <-time.NewTicker(timeout).C:
//		//超时了
//		this.C <- "timeout"
//	}
//	//在消息中心删除这个管道超时任务
//	removeTimeoutMapping(this.hash)
//}

///*
//	完成这个管道
//*/
//func (this *Pipe) done(data string) {
//	//完成这个管道使命
//	this.Done <- true
//	//给管道监听者发送数据
//	this.C <- data
//}

///*
//	注册一个消息超时任务
//	@id    超时时间内需要返回的消息id
//	@msg   发送出去的消息
//*/
//func SendTimeOutMsg(msg *Message, timeout time.Duration) (c chan string) {
//	pipe := &Pipe{
//		hash: GetHash(msg),
//		C:    make(chan string, 1),
//		Done: make(chan bool, 1),
//	}
//	go pipe.runTimeout(timeout)
//	addTimeoutMapping(pipe.hash, pipe)
//	SendMessage(msg)
//	return pipe.C
//}

///*
//	回调函数执行成功后，给此方法一个事件
//	用于消除消息的超时
//*/
//func fire(msg *Message, retOK bool, result string) {
//	//不是回复消息
//	if msg.ReplyTime == 0 {
//		return
//	}
//	pipe, ok := getTimeoutMapping(msg.ReplyHash)
//	if ok {
//		pipe.done(result)
//	}
//}
