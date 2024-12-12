package common

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Player is an helper struct for communication between nodes.
// the Rank is an identifier of the Player.
// Addresses[i] contains the address to reach the Player with Rank i.
type Player struct {
	Rank      int
	Addresses []net.TCPAddr
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
		return nil, err
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

type broadcastHandler struct {
	RootRank       int
	ContentChannel chan []byte
	ErrChannel     chan error
}

func (h *broadcastHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	senderRankS, ok := req.Header["Rank"]
	if !ok {
		rw.WriteHeader(http.StatusNotAcceptable)
		h.ErrChannel <- fmt.Errorf("from handler: Rank field is not present in request")
		return
	}
	senderRank, err := strconv.Atoi(senderRankS[0])
	if err != nil {
		rw.WriteHeader(http.StatusNotAcceptable)
		h.ErrChannel <- fmt.Errorf("from handler: Rank field is not a number")
		return
	}
	if senderRank != h.RootRank {
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
func (p Player) broadcastNoBarrier(bufferSend []byte, root int) ([]byte, error) {
	if root == p.Rank {
		client := http.Client{}
		defer client.CloseIdleConnections()
		for i, addr := range p.Addresses {
			if i != p.Rank {
				req, err := http.NewRequest("POST", "http://"+addr.String(), strings.NewReader(string(bufferSend)))
				if err != nil {
					return nil, err
				}
				req.Header["Rank"] = []string{fmt.Sprint(p.Rank)}
				resp, err := client.Do(req)
				for err != nil || resp.StatusCode != http.StatusAccepted {
					time.Sleep(time.Millisecond)
					resp, err = client.Do(req)
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
		RootRank:       root,
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
