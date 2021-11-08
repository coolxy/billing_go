package billing

import (
	"errors"
	"net"
	"strconv"
)

// initListener 初始化Tcp Listener
func (s *Server) initListener() error {
	//监听的TCP地址端口
	listenAddress := s.Config.IP + ":" + strconv.Itoa(s.Config.Port)
	serverEndpoint, err := net.ResolveTCPAddr("tcp", listenAddress)
	if err != nil {
		return errors.New("resolve TCPAddr failed: " + err.Error())
	}
	//监听TCP连接
	listener, err := net.ListenTCP("tcp", serverEndpoint)
	if err != nil {
		return err
	}
	s.Listener = listener
	return nil
}
