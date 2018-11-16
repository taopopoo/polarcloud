package engine

import (
	"encoding/json"
	"net"
	"time"
)

//其他计算机对本机的连接
type ServerConn struct {
	sessionBase
	conn           net.Conn
	Ip             string
	Connected_time string
	CloseTime      string
	packet         Packet
	engine         *Engine
	controller     Controller
}

func (this *ServerConn) run() {

	this.packet.Session = this

	go this.recv()
}

//接收客户端消息协程
func (this *ServerConn) recv() {
	defer PrintPanicStack()
	//处理客户端主动断开连接的情况
	var err error
	var handler MsgHandler
	for {
		err = RecvPackage(this.conn, &this.packet)
		if err != nil {
			break
		} else {
			//			if this.packet.MsgID == 1007 {
			//				NLog.Debug(LOG_file, "conn recv: %d, %d, %d, %d",
			//					this.packet.MsgID, this.packet.Size, len(this.packet.Data), len(this.packet.Dataplus))
			//				//				NLog.Debug(LOG_file, "conn recv: %d", len(this.packet.Data))
			//				//				NLog.Debug(LOG_file, "conn recv: %d, %v", this.packet.Data)
			//				//				NLog.Debug(LOG_file, "conn recv: %d", len(this.packet.Dataplus))
			//				//				NLog.Debug(LOG_file, "conn recv: %d, %v", this.packet.Dataplus)
			//			}
			//			if this.packet.MsgID == 111 {
			//				Log.Debug("conn recv: %d, %s, %d", this.packet.MsgID, this.Ip, len(this.packet.Data)+16)
			//			}
			//			Log.Debug("conn recv: %d, %s, %d", this.packet.MsgID, this.Ip, len(this.packet.Data)+16)
			//			Log.Debug("conn recv: %d, %s, %d %v", this.packet.MsgID, this.Ip, len(this.packet.Data)+16, this.packet.Data)

			if this.packet.IsWait {
				Log.Debug("开始等待")
				this.packet.IsWait = false
				this.packet.WaitChan <- true
				Log.Debug("开始执行")
				<-this.packet.WaitChan
				Log.Debug("执行完成")
			} else {
				handler = this.engine.router.GetHandler(this.packet.MsgID)
				if handler == nil {
					Log.Warn("server该消息未注册，消息编号：%d", this.packet.MsgID)
					//					if this.packet.MsgID == 16 {
					//						fmt.Println(string(this.packet.Data))
					//					}
					//					break
				} else {
					//这里决定了消息是否异步处理
					this.handlerProcess(handler, &this.packet)
				}
			}

			//				copy(this.cache, this.cache[this.packet.Size:this.cacheindex])
			//				this.cacheindex = this.cacheindex - uint32(n)

		}
	}

	this.Close()

	//最后一个包接收了之后关闭chan
	//如果有超时包需要等超时了才关闭，目前未做处理
	// close(this.outData)
	// fmt.Println("关闭连接")
}

func (this *ServerConn) Waite(du time.Duration) *Packet {
	if this.packet.Wait(du) {
		return &this.packet
	}
	return nil
}

func (this *ServerConn) FinishWaite() {
	this.packet.FinishWait()
}

func (this *ServerConn) handlerProcess(handler MsgHandler, msg *Packet) {
	//消息处理模块报错将不会引起宕机
	defer PrintPanicStack()
	//消息处理前先通过拦截器
	itps := this.engine.interceptor.getInterceptors()
	itpsLen := len(itps)
	for i := 0; i < itpsLen; i++ {
		isIntercept := itps[i].In(this.controller, *msg)
		//
		if isIntercept {
			return
		}
	}
	handler(this.controller, *msg)
	//消息处理后也要通过拦截器
	for i := itpsLen; i > 0; i-- {
		itps[i-1].Out(this.controller, *msg)
	}
}

//给客户端发送数据
func (this *ServerConn) Send(msgID uint64, data, dataplus *[]byte, waite bool) (err error) {
	defer PrintPanicStack()
	this.packet.IsWait = waite
	buff := MarshalPacket(msgID, data, dataplus)
	_, err = this.conn.Write(*buff)
	if err != nil {
		Log.Debug("conn send err: %s", err.Error())
	} else {
		//		if msgID == 1007 {
		//			Log.Debug("conn send: %d, %s, %d", msgID, this.Ip, len(*dataplus))
		//		}
		//		if msgID == 1007 {
		//			NLog.Debug(LOG_file, "conn send: %d, %s, %d, %d, %d", msgID, this.Ip, len(*buff), len(*data), len(*dataplus))
		//			//			NLog.Debug(LOG_file, "conn send: %d, %s, %d, %d", msgID, this.Ip, len(*data), len(*dataplus))
		//			//			NLog.Debug(LOG_file, "conn send: %s", string(*buff))
		//			//			NLog.Debug(LOG_file, "conn send: %v", *buff)
		//		}
		//		Log.Debug("conn send: %d, %s, %d", msgID, this.Ip, len(*buff))
		//		Log.Debug("conn send: %d, %s, %d %v", msgID, this.Ip, len(*buff), buff)
	}
	return
}

//给客户端发送数据
func (this *ServerConn) SendJSON(msgID uint64, data interface{}, waite bool) (err error) {
	defer PrintPanicStack()
	this.packet.IsWait = waite
	var f []byte
	f, err = json.Marshal(data)
	if err != nil {
		return
	}
	buff := MarshalPacket(msgID, &f, nil)
	_, err = this.conn.Write(*buff)
	Log.Debug("conn send: %d, %s, %d", msgID, this.conn.RemoteAddr(), len(*buff))
	return
}

//关闭这个连接
func (this *ServerConn) Close() {
	if this.engine.closecallback != nil {
		this.engine.closecallback(this.GetName())
	}
	this.engine.sessionStore.removeSession(this.GetName())
	err := this.conn.Close()
	if err != nil {
	}
}

func (this *ServerConn) SetName(name string) {
	this.engine.sessionStore.renameSession(this.name, name)
	this.name = name
}

//获取远程ip地址和端口
func (this *ServerConn) GetRemoteHost() string {
	return this.conn.RemoteAddr().String()
}
