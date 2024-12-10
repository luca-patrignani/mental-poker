package common

import (
	"bufio"
	"net"
	"time"
)

// Player is an helper struct for communication between nodes.
// the Rank is an identifier of the Player.
// Addresses[i] contains the address to reach the Player with Rank i.
type Player struct {
	Rank      int
	Addresses []string
}

// Each caller of AllToAll sends the content of bufferSend to every node.
// bufferRecv[i] will contain the value sent by the Player with Rank i.
// This function will implicitly synchronize the players.
func (p Player) AllToAll(bufferSend string) (bufferRecv []string, err error) {
	bufferRecv = make([]string, len(p.Addresses))
	for i := 0; i < len(p.Addresses); i++ {
		recv, err := p.broadcastNoBarrier(bufferSend, i)
		if err != nil {
			return nil, err
		}
		bufferRecv[i] = recv
	}
	return
}

// The Player with Rank root sends the content of bufferSend to every node.
// bufferRecv will contain the value sent by the Player with Rank root.
// This function will implicitly synchronize the players.
func (p Player) Broadcast(bufferSend string, root int) (string, error) {
	bufferRecv, err := p.broadcastNoBarrier(bufferSend, root)
	if err != nil {
		return "", err
	}
	err = p.barrier()
	if err != nil {
		return "", nil
	}
	return bufferRecv, nil
}

// barrier sychronizes the players.
// In particular this method guarantees that no Player's control flow will
// leave this function until every player has entered this function.
func (p Player) barrier() error {
	_, err := p.AllToAll("")
	if err != nil {
		return err
	}
	return nil
}

// The Player with Rank root sends the content of bufferSend to every node.
// bufferRecv will contain the value sent by the Player with Rank root.
func (p Player) broadcastNoBarrier(bufferSend string, root int) (string, error) {
	const delim byte = 0x0
	if root == p.Rank {
		for i, addr := range p.Addresses {
			if i != p.Rank {
				conn, err := net.Dial("tcp", addr)
				for err != nil {
					time.Sleep(time.Millisecond)
					conn, err = net.Dial("tcp", addr)
				}
				defer conn.Close()
				_, err = conn.Write(append([]byte(bufferSend), delim))
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
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString(delim)
	if err != nil {
		return "", err
	}
	line = line[:len(line)-1]
	return line, nil
}
