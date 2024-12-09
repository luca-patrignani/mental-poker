package common

import (
	"net"
	"time"
)

type Player struct {
	Rank      int
	Addresses []string
}

func (p Player) AllToAll(bufferSend string) (bufferRecv []string, err error) {
	bufferRecv = make([]string, len(p.Addresses))
	for i := 0; i < len(p.Addresses); i++ {
		recv, err := p.Broadcast(bufferSend, i)
		if err != nil {
			return nil, err
		}
		bufferRecv[i] = recv
	}
	return
}

func (p Player) Broadcast(bufferSend string, root int) (string, error) {
	if root == p.Rank {
		for i, addr := range p.Addresses {
			if i != p.Rank {
				conn, err := net.Dial("tcp", addr)
				for err != nil {
					time.Sleep(time.Millisecond)
					conn, err = net.Dial("tcp", addr)
				}
				defer conn.Close()
				_, err = conn.Write([]byte(bufferSend))
				if err != nil {
					return "", err
				}
			}
		}
		return bufferSend, nil
	}
	listener, err := net.Listen("tcp", p.Addresses[p.Rank])
	if err != nil {
		return "", err
	}
	conn, err := listener.Accept()
	if err != nil {
		return "", err
	}
	listener.Close()
	defer conn.Close()
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return "", err
	}
	buffer = buffer[:n]
	return string(buffer), nil
}
