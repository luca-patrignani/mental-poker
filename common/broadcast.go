package common

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"
)

// Player is an helper struct for communication between nodes.
// the Rank is an identifier of the Player.
// Addresses[i] contains the address to reach the Player with Rank i.
type Player struct {
	Rank      int
	Addresses []net.TCPAddr
	// listener net.Listener
}

func NewPlayer(rank int, addresses []net.TCPAddr) (Player, error) {
	// listener, err := net.Listen("tcp", addresses[rank])
	// if err != nil {
	// 	return Player{}, err
	// }
	p := Player{
		Rank:      rank,
		Addresses: addresses,
		// listener: listener,
	}
	return p, nil
}

// Each caller of AllToAll sends the content of bufferSend to every node.
// bufferRecv[i] will contain the value sent by the Player with Rank i.
// This function will implicitly synchronize the players.
func (p Player) AllToAll(bufferSend []byte) (bufferRecv [][]byte, err error) {
	bufferRecv = make([][]byte, len(p.Addresses))
	bufferRecv[p.Rank] = bufferSend
	s := http.ServeMux{}
	fatal := make(chan error)
	s.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		remoteAddr, err := net.ResolveTCPAddr("tcp", r.RemoteAddr)
		if err != nil {
			fatal <- err
			return
		}
		i := -1
		for j := 0; j < len(p.Addresses); j++ {
			if tcpAddrEqual(p.Addresses[j], *remoteAddr) {
				i = j
				break
			}
		}
		recv, err := io.ReadAll(r.Body)
		if err != nil {
			fatal <- err
			return
		}
		bufferRecv[i] = recv
		w.WriteHeader(http.StatusAccepted)
	})
	// Create a custom transport with a DialContext
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			LocalAddr: &p.Addresses[p.Rank], // Set the local address
		}).DialContext,
	}
	// Create an HTTP client with the custom transport
	client := &http.Client{
		Transport: transport,
	}
	for i, addr := range p.Addresses {
		if i != p.Rank {
			resp, err := client.Post("http://"+addr.String(), "application/octet-stream", strings.NewReader(string(bufferSend)))
			for err != nil || resp.StatusCode != http.StatusAccepted {
				resp, err = client.Post("http://"+addr.String(), "application/octet-stream", strings.NewReader(string(bufferSend)))
			}
		}
	}

	for slices.ContainsFunc(bufferRecv, func(b []byte) bool { return b == nil }) {
	}
	return
}

type myHandler struct {
	RootAddr       net.TCPAddr
	ContentChannel chan []byte
	ErrChannel     chan error
}

func (h *myHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
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
	_, err := p.AllToAll(nil)
	if err != nil {
		return err
	}
	return nil
}

// The Player with Rank root sends the content of bufferSend to every node.
// bufferRecv will contain the value sent by the Player with Rank root.
func (p Player) broadcastNoBarrier(bufferSend string, root int) (string, error) {
	if root == p.Rank {
		// Create a custom transport with a DialContext
		transport := &http.Transport{
			DialContext: (&net.Dialer{
				LocalAddr: &p.Addresses[p.Rank], // Set the local address
			}).DialContext,
		}
		// Create an HTTP client with the custom transport
		client := &http.Client{
			Transport: transport,
		}
		for i, addr := range p.Addresses {
			if i != p.Rank {
				resp, err := client.Post("http://"+addr.String(), "application/octet-stream", strings.NewReader(bufferSend))
				for err != nil || resp.StatusCode != http.StatusAccepted {
					time.Sleep(time.Millisecond)
					resp, err = client.Post("http://"+addr.String(), "application/octet-stream", strings.NewReader(bufferSend))
				}
				if resp.StatusCode != http.StatusAccepted {
					return "", fmt.Errorf("unsuccessful status code %d", resp.StatusCode)
				}
			}
		}
		return bufferSend, nil
	}
	errChan := make(chan error)
	handler := myHandler{
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
		return "", err
	case recv = <-handler.ContentChannel:
		break
	}
	s.Shutdown(context.Background())
	return string(recv), nil
}

func tcpAddrEqual(a, b net.TCPAddr) bool {
	return a.IP.Equal(b.IP) && a.Port == b.Port
}
