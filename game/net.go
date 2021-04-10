package game

import (
	"fmt"
	"net"
)

type Net struct {
}

type Server struct {
	Listener    *net.TCPListener
	AcceptError error
}

func NServer(port int) (s *Server, err error) {
	s = &Server{}

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return
	}

	s.Listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return
	}

	return
}
