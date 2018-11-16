package utils

import (
	"net"
	"strconv"
)

/*
	获得一个TCP监听
*/
func GetTCPListener(ip string, port int) (*net.TCPListener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", ip+":"+strconv.Itoa(int(port)))
	if err != nil {
		// Log.Error("这个地址不符合规范：%s", ip+":"+strconv.Itoa(int(port)))
		return nil, err
	}
	var listener *net.TCPListener
	listener, err = net.ListenTCP("tcp4", tcpAddr)
	if err != nil {
		// Log.Error("监听一个地址失败：%s", ip+":"+strconv.Itoa(int(port)))
		// Log.Error("%v", err)
		return nil, err
	}
	// Log.Debug("监听一个地址：%s", ip+":"+strconv.Itoa(int(port)))
	// fmt.Println("监听一个地址：", ip+":"+strconv.Itoa(int(port)))
	// fmt.Println(ip + ":" + strconv.Itoa(int(port)) + "成功启动服务器")
	return listener, nil
}
