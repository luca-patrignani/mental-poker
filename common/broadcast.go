package common

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
)

// Player is an helper struct for communication between nodes.
// the Rank is an identifier of the Player.
// Addresses[i] contains the address to reach the Player with Rank i.
type Player struct {
	Rank      int
	Addresses []net.TCPAddr
}

type broadcastHandler struct {
	RootAddr       net.TCPAddr
	ContentChannel chan []byte
	ErrChannel     chan error
}

func (h *broadcastHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	remoteAddr, err := net.ResolveTCPAddr("tcp", req.RemoteAddr)
	if err != nil {
		rw.WriteHeader(http.StatusNotAcceptable)
		h.ErrChannel <- fmt.Errorf("from handler: %v", err)
		return
	}
	if !tcpAddrEqual(*remoteAddr, h.RootAddr) {
		rw.WriteHeader(http.StatusNotAcceptable)
		return
	}
	content, err := io.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		h.ErrChannel <- fmt.Errorf("from handler: %v", err)
		return
	}
	h.ContentChannel <- content
	rw.WriteHeader(http.StatusAccepted)
}

// The Player with Rank root sends the content of bufferSend to every node.
// bufferRecv will contain the value sent by the Player with Rank root.
// This function will implicitly synchronize the players.
func (p Player) Broadcast(bufferSend []byte, root int) ([]byte, error) {
	bufferRecv, err := p.broadcastNoBarrier(bufferSend, root)
	if err != nil {
		return nil, err
	}
	err = p.barrier()
	if err != nil {
		return nil, nil
	}
	return bufferRecv, nil
}

// Each caller of AllToAll sends the content of bufferSend to every node.
// bufferRecv[i] will contain the value sent by the Player with Rank i.
// This function will implicitly synchronize the players.
func (p Player) AllToAll(bufferSend []byte) (bufferRecv [][]byte, err error) {
	bufferRecv = make([][]byte, len(p.Addresses))
	for i := 0; i < len(p.Addresses); i++ {
		recv, err := p.broadcastNoBarrier(bufferSend, i)
		if err != nil {
			return nil, err
		}
		bufferRecv[i] = recv
	}
	return
}

// barrier synchronizes the players.
// In particular this method guarantees that no Player's control flow will
// leave this function until every player has entered this function.
func (p Player) barrier() error {
	_, err := p.AllToAll(nil)
	if err != nil {
		return err
	}
	return nil
}

// The Player with Rank root sends the content of bufferSend to every node.
// bufferRecv will contain the value sent by the Player with Rank root.
func (p Player) broadcastNoBarrier(bufferSend []byte, root int) ([]byte, error) {
	if root == p.Rank {
		// Create a custom transport with a DialContext
		transport := &http.Transport{
			DialContext: (&net.Dialer{
				LocalAddr: &p.Addresses[p.Rank], // Set the local address
				Control: func(network, address string, c syscall.RawConn) error {
					var err error
					outerErr := c.Control(func(fd uintptr) {
						err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
					})
					return errors.Join(err, outerErr)
				},
			}).DialContext,
		}
		// Create an HTTP client with the custom transport
		client := &http.Client{
			Transport: transport,
		}
		defer client.CloseIdleConnections()
		for i, addr := range p.Addresses {
			if i != p.Rank {
				resp, err := client.Post("http://"+addr.String(), "application/octet-stream", strings.NewReader(string(bufferSend)))
				for err != nil || resp.StatusCode != http.StatusAccepted {
					time.Sleep(time.Millisecond)
					resp, err = client.Post("http://"+addr.String(), "application/octet-stream", strings.NewReader(string(bufferSend)))
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusAccepted {
					return nil, fmt.Errorf("unsuccessful status code %d", resp.StatusCode)
				}
			}
		}
		return bufferSend, nil
	}
	errChan := make(chan error)
	handler := broadcastHandler{
		RootAddr:       p.Addresses[root],
		ContentChannel: make(chan []byte),
		ErrChannel:     errChan,
	}
	s := http.Server{
		Addr:        p.Addresses[p.Rank].AddrPort().String(),
		Handler:     &handler,
		IdleTimeout: time.Second,
	}
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			errChan <- err
			return
		}
	}()
	var recv []byte
	select {
	case err := <-errChan:
		return nil, err
	case recv = <-handler.ContentChannel:
		break
	}
	err := s.Shutdown(context.Background())
	return recv, err
}

func tcpAddrEqual(a, b net.TCPAddr) bool {
	return a.IP.Equal(b.IP) && a.Port == b.Port
}
