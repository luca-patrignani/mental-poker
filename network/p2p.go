package network

// P2P  adatta common.Peer all'interfaccia consensus.NetworkLayer
type P2P struct {
	peer *Peer
}

// NewP2P  crea un nuovo
func NewP2P(peer *Peer) *P2P {
	return &P2P{peer: peer}
}

// Broadcast invia dati a tutti i peer
func (p *P2P) Broadcast(data []byte, root int) ([]byte, error) {
	return p.peer.Broadcast(data, root)
}

// AllToAll invia dati da tutti a tutti
func (p *P2P) AllToAll(data []byte) ([][]byte, error) {
	return p.peer.AllToAll(data)
}

// GetRank ritorna il rank di questo nodo
func (p *P2P) GetRank() int {
	return p.peer.Rank
}

// GetPeerCount ritorna il numero di peer
func (p *P2P) GetPeerCount() int {
	return len(p.peer.Addresses)
}

func (p *P2P) GetAddresses() map[int]string {
	return p.peer.Addresses
}

// Close chiude la connessione
func (p *P2P) Close() error {
	return p.peer.Close()
}
