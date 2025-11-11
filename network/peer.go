package network

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Peer is an helper struct for communication between nodes.
// the Rank is an identifier of the Peer.
// Addresses[i] contains the address to reach the Peer with Rank i.
type Peer struct {
	Rank      int
	Addresses map[int]string
	clock     uint64
	server    *http.Server
	handler   *broadcastHandler
	timeout   time.Duration
}

type ConnectionError struct {
	FaultAdresses map[int]string
	Errors        map[int]error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection error with addresses: %v, error: %v", e.FaultAdresses, e.Errors)
}

// NewPeer creates a new Peer with given rank, addresses and listener.
// The listener should be already bound to the address corresponding to the rank.
// The timeout parameter specifies the duration to wait for responses during communication.
func NewPeer(rank int, addresses map[int]string, l net.Listener, timeout time.Duration) Peer {
	handler := &broadcastHandler{
		contentChannel: make(chan []byte),
		errChannel:     make(chan error),
	}
	p := Peer{
		Rank:      rank,
		Addresses: copyMap(addresses),
		clock:     0,
		server:    &http.Server{Addr: addresses[rank], Handler: handler},
		handler:   handler,
		timeout:   timeout,
	}
	go func() {
		err := p.server.Serve(l)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	return p
}

func (p Peer) Close() error {
	return p.server.Shutdown(context.Background())
}

type broadcastHandler struct {
	active         atomic.Bool
	clock          uint64
	contentChannel chan []byte
	errChannel     chan error
}

func (h *broadcastHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if !h.active.Load() {
		rw.WriteHeader(http.StatusNotAcceptable)
		return
	}
	senderClockS, ok := req.Header["Clock"]
	if !ok {
		rw.WriteHeader(http.StatusNotAcceptable)
		h.errChannel <- fmt.Errorf("from handler: Clock field is not present in request")
		return
	}
	senderClock, err := strconv.ParseUint(senderClockS[0], 10, 64)
	if err != nil {
		rw.WriteHeader(http.StatusNotAcceptable)
		h.errChannel <- fmt.Errorf("from handler: Clock field is not a number")
		return
	}
	if senderClock != h.clock {
		rw.WriteHeader(http.StatusNotAcceptable)
		return
	}
	content, err := io.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		h.errChannel <- fmt.Errorf("from handler: %v", err)
		return
	}
	h.contentChannel <- content
	rw.WriteHeader(http.StatusAccepted)
}

// Peer with Rank root sends the content of bufferSend to every node.
// bufferRecv will contain the value sent by the Peer with Rank root.
// This function will implicitly synchronize the peers.
func (p *Peer) Broadcast(bufferSend []byte, root int) ([]byte, error) {
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
// bufferRecv[i] will contain the value sent by the Peer with Rank i.
// This function will implicitly synchronize the peers.
func (p *Peer) AllToAll(bufferSend []byte) (bufferRecv [][]byte, err error) {
	size, b := maxKey(p.Addresses)
	if !b {
		return nil, fmt.Errorf("no addresses found")
	}

	var orderedRanks []int
	for k := range p.Addresses {
		orderedRanks = append(orderedRanks, k)
	}
	sort.Ints(orderedRanks)

	bufferRecv = make([][]byte, size+1)
	for _, i := range orderedRanks {
		recv, err := p.broadcastNoBarrier(bufferSend, i)
		if err != nil {
			return nil, err
		}
		bufferRecv[i] = recv
	}
	return
}

// barrier synchronizes the peers.
// In particular this method guarantees that no Peer's control flow will
// leave this function until every peer has entered this function.
func (p Peer) barrier() error {
	_, err := p.AllToAll(nil)
	if err != nil {
		return err
	}
	return nil
}

// helper function for creating n addresses localhost:PORT
func CreateAddresses(n int) map[int]string {
	addresses := make(map[int]string)
	for i := 0; i < n; i++ {
		l, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			panic(err)
		}
		addresses[i] = l.Addr().String()
		if err := l.Close(); err != nil {
			panic(err)
		}
	}
	return addresses
}

func CreateListeners(n int) (map[int]net.Listener, map[int]string) {
	listeners := make(map[int]net.Listener)
	addresses := make(map[int]string)
	for i := 0; i < n; i++ {
		l, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			panic(err)
		}
		listeners[i] = l
		addresses[i] = l.Addr().String()
	}
	return listeners, addresses
}

// Peer with Rank root sends the content of bufferSend to every node.
// bufferRecv will contain the value sent by the Peer with Rank root.
func (p *Peer) broadcastNoBarrier(bufferSend []byte, root int) ([]byte, error) {
	p.clock++
	connErr := &ConnectionError{
		FaultAdresses: make(map[int]string),
		Errors:        make(map[int]error),
	}
	if root == p.Rank {
		client := http.Client{Timeout: p.timeout}
		for i, addr := range p.Addresses {
			if i != p.Rank {
				//fmt.Printf("Node %d requesting post to %d\n",p.Rank,i)
				req, err := http.NewRequest("POST", "http://"+addr, strings.NewReader(string(bufferSend)))
				if err != nil {
					connErr.FaultAdresses[i] += addr
					connErr.Errors[i] = errors.Join(connErr.Errors[i], err)
					continue
				}
				req.Header["Clock"] = []string{fmt.Sprint(p.clock)}
				req.Header["SenderRank"] = []string{fmt.Sprint(p.Rank)}
				req.Header["ReceiverRank"] = []string{fmt.Sprint(i)}
				resp, err := client.Do(req)
				start := time.Now()
				for err != nil || resp.StatusCode != http.StatusAccepted {
					resp, err = client.Do(req)
					if p.timeout > 0 && time.Since(start) > p.timeout {
						if err != nil {
							connErr.FaultAdresses[i] += addr
							connErr.Errors[i] = errors.Join(connErr.Errors[i], fmt.Errorf("connection attempts timed out with error %w", err))
							break
						}
						connErr.FaultAdresses[i] += addr
						connErr.Errors[i] = errors.Join(connErr.Errors[i], fmt.Errorf("connection attempts timed out with status code %d", resp.StatusCode))
						break
					}
				}
				if resp == nil {
					continue
				}
				if err := resp.Body.Close(); err != nil {
					connErr.FaultAdresses[i] += addr
					connErr.Errors[i] = errors.Join(connErr.Errors[i], err)
					continue
				}
			}
		}
		if len(connErr.FaultAdresses) > 0 {
			return nil, connErr
		}
		return bufferSend, nil
	}
	p.handler.clock = p.clock
	p.handler.active.Store(true)
	defer p.handler.active.Store(false)
	var recv []byte
	timeoutTicker := make(<-chan time.Time)
	if p.timeout > 0 {
		timeoutTicker = time.Tick(p.timeout)
	}
	select {
	case recv = <-p.handler.contentChannel:
		break
	case err := <-p.handler.errChannel:
		connErr.FaultAdresses[root] = p.Addresses[root]
		connErr.Errors[root] = err
		return nil, connErr
	case <-timeoutTicker:
		err := p.Close()
		connErr.FaultAdresses[root] = p.Addresses[root]
		connErr.Errors[root] = errors.Join(err, fmt.Errorf("the peer waiting for connection timed out"))
		return nil, connErr
	}
	return recv, nil
}

func maxKey(m map[int]string) (max int, ok bool) {
	ok = false
	for k := range m {
		if !ok || k > max {
			max = k
			ok = true
		}
	}
	return
}

func copyMap(original map[int]string) map[int]string {
	copied := make(map[int]string)
	for k, v := range original {
		copied[k] = v
	}
	return copied
}
