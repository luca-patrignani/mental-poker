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

	"github.com/pterm/pterm"
)

// Peer represents a node in the peer-to-peer network.
// It maintains connections to other nodes and provides communication primitives
// for distributed consensus protocols.
type Peer struct {
	Rank      int               // Unique identifier for this peer
	Addresses map[int]string    // Map of rank to network address
	clock     uint64            // Logical clock for message ordering
	server    *http.Server      // HTTP server for receiving messages
	handler   *broadcastHandler // Handler for incoming broadcasts
	timeout   time.Duration     // Communication timeout
}

// NewPeer creates and starts a new peer.
//
// Parameters:
//   - rank: Unique identifier for this peer
//   - addresses: Map of all peer ranks to their network addresses
//   - l: Network listener for this peer
//   - timeout: Maximum time to wait for responses
//
// The peer's HTTP server starts immediately in a background goroutine.
//
// Returns the initialized peer.
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

// Broadcast performs a one-to-all communication pattern.
//
// The peer with rank 'root' sends bufferSend to all other peers.
// All peers (including root) receive the same data in bufferRecv.
//
// This operation includes an implicit barrier synchronization, ensuring
// all peers have completed the broadcast before any peer returns.
//
// Parameters:
//   - bufferSend: Data to send (only used by the root peer)
//   - root: Rank of the peer that broadcasts
//
// Returns the broadcast data or an error if communication fails.
//
// Thread-safety: This method synchronizes all participating peers.
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

// BroadcastwithTimeout executes a Broadcast communication to a specific peer rank with a
// specified timeout duration. It retries every 5 seconds until a response is received or
// the timeout is exceeded.
func (p *Peer) BroadcastwithTimeout(data []byte, rank int, timeout time.Duration) ([]byte, error) {
	var response []byte
	start := time.Now()

	for {
		if time.Since(start) > timeout {
			return response, fmt.Errorf("timeout: no message received")
		}

		response, err := p.Broadcast(data, rank)
		if err == nil {

			return response, nil
		}
		fmt.Printf("Error in broadcasting votes: %s, retry in 5 seconds\n", err)
		time.Sleep(5000 * time.Millisecond)

	}

}

// AllToAll performs an all-to-all communication pattern.
//
// Each peer sends its bufferSend to all other peers.
// bufferRecv[i] contains the data sent by peer with rank i.
//
// This operation includes an implicit barrier synchronization.
//
// Parameters:
//   - bufferSend: Data to send from this peer
//
// Returns a slice where index i contains data from peer i, or an error.
//
// Thread-safety: This method synchronizes all participating peers.
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

// AllToAllwithTimeout executes an AllToAll communication with a specified timeout duration.
// It retries every 5 seconds until either all expected responses are received or the timeout
// is exceeded. Returns partial results if timeout occurs.
func (p *Peer) AllToAllwithTimeout(data []byte, timeout time.Duration) ([][]byte, error) {
	expected := len(p.Addresses)
	var responses [][]byte
	start := time.Now()

	for {
		if time.Since(start) > timeout {
			return responses, fmt.Errorf("timeout: received %d of %d messages", len(responses), expected)
		}

		responses, err := p.AllToAll(data)
		if err != nil {
			fmt.Printf("Error in broadcasting: %v\n", err)
		}

		if responses == nil {
			msg := fmt.Sprintf("Error in broadcasting: responses of length %d instead of %d", len(responses), expected)
			pterm.Warning.Println(msg)
		}
		if len(responses) >= expected {
			return responses, nil
		}
		pterm.Info.Println("Retry in 5 seconds. . .")
		time.Sleep(5000 * time.Millisecond)
	}

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
	if root == p.Rank {
		client := http.Client{Timeout: p.timeout}
		for i, addr := range p.Addresses {
			if i != p.Rank {
				//fmt.Printf("Node %d requesting post to %d\n",p.Rank,i)
				req, err := http.NewRequest("POST", "http://"+addr, strings.NewReader(string(bufferSend)))
				if err != nil {
					return nil, err
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
							return nil, fmt.Errorf("connection attempts timed out with error %w", err)
						}
						return nil, fmt.Errorf("connection attempts timed out with status code %d", resp.StatusCode)
					}
				}
				if err := resp.Body.Close(); err != nil {
					return nil, err
				}
			}
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
		return nil, err
	case <-timeoutTicker:
		err := p.Close()
		return nil, errors.Join(err, fmt.Errorf("the peer waiting for connection timed out"))
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
