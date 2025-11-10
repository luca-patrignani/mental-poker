package network

import "time"

// P2P is an adapter of Peer to the interface NetworkLayer
type P2P struct {
	peer *Peer
}

// NewP2P creates a new P2P adapter wrapping the provided Peer.
func NewP2P(peer *Peer) *P2P {
	return &P2P{peer: peer}
}

// Broadcast sends data from this node (identified by rank root) to all peers.
// It delegates to the underlying Peer's Broadcast method.
func (p *P2P) Broadcast(data []byte, root int) ([]byte, error) {
	return p.peer.Broadcast(data, root)
}

// BroadcastwithTimeout executes a Broadcast communication to a specific peer rank with a
// specified timeout duration. It retries every 5 seconds until a response is received or
// the timeout is exceeded.
func (p *P2P) BroadcastwithTimeout(data []byte, rank int, timeout time.Duration) ([]byte, error) {
	return p.peer.BroadcastwithTimeout(data, rank, timeout)
}

// AllToAll sends data from this node to all peers and receives data from all peers.
// It delegates to the underlying Peer's AllToAll method.
func (p *P2P) AllToAll(data []byte) ([][]byte, error) {
	return p.peer.AllToAll(data)
}

// Each caller of AllToAll sends the content of bufferSend to every node.
// bufferRecv[i] will contain the value sent by the Peer with Rank i.
// This function will implicitly synchronize the peers.
func (p *P2P) AllToAllwithTimeout(data []byte, timeout time.Duration) ([][]byte, error) {
	return p.peer.AllToAllwithTimeout(data, timeout)
}

// GetRank returns the rank (unique identifier) of this node.
func (p *P2P) GetRank() int {
	return p.peer.Rank
}

// GetPeerCount returns the number of peers including this node.
func (p *P2P) GetPeerCount() int {
	return len(p.peer.Addresses)
}

// GetAddresses returns the map of rank to address for all peers.
func (p *P2P) GetAddresses() map[int]string {
	return p.peer.Addresses
}

// Close closes the underlying peer connection.
func (p *P2P) Close() error {
	return p.peer.Close()
}

func (p *P2P) RemovePeer(leaver int) {
	delete(p.peer.Addresses, leaver)
}
