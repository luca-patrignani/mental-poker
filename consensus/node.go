package consensus

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"time"
)

// NewConsensusNode crea un nuovo nodo di consenso
func NewConsensusNode(
	id string,
	pub ed25519.PublicKey,
	priv ed25519.PrivateKey,
	peers map[int]ed25519.PublicKey,
	sm StateMachine,
	ledger Ledger,
	network NetworkLayer,
) *ConsensusNode {
	n := len(peers)
	return &ConsensusNode{
		pub:       pub,
		priv:      priv,
		playersPK: peers,
		quorum:    computeQuorum(n), // BFT quorum
		pokerSM:   sm,
		ledger:    ledger,
		network:   network,
		proposal:  nil,
		votes:     nil,
	}
}

func (cn *ConsensusNode) UpdatePeers() error {
	b, err := json.Marshal(cn.pub)
	if err != nil {
		return err
	}
	pkBytes, err := cn.network.AllToAll(b)
	if err != nil {
		return err
	}
	pk := make(map[int]ed25519.PublicKey, len(pkBytes))
	for i, pki := range pkBytes {
		var p ed25519.PublicKey
		if err := json.Unmarshal(pki, &p); err != nil {
			return fmt.Errorf("failed to unmarshal public key: %v\n", err)
		}
		pk[i] = p
	}
	cn.playersPK = pk
	return nil
}

// Funzione all-to-all con timeout
func (node *ConsensusNode) AllToAllwithTimeout(data []byte, timeout time.Duration) ([][]byte, error) {
	expected := node.network.GetPeerCount()
	var responses [][]byte
	start := time.Now()

	for {
		if time.Since(start) > timeout {
			return responses, fmt.Errorf("timeout: received %d of %d messages", len(responses), expected)
		}

		responses, err := node.network.AllToAll(data)
		if err != nil {
			fmt.Printf("Error in broadcasting votes: %v, retry in 5 seconds\n", err)
			time.Sleep(5000 * time.Millisecond)
			continue
		}

		if len(responses) >= expected {
			break
		}
		time.Sleep(5000 * time.Millisecond)
	}

	return responses, nil
}

// Funzione all-to-all con timeout
func (node *ConsensusNode) BroadcastwithTimeout(data []byte, rank int, timeout time.Duration) ([]byte, error) {
	var response []byte
	start := time.Now()

	for {
		if time.Since(start) > timeout {
			return response, fmt.Errorf("timeout: no message received\n")
		}

		response, err := node.network.Broadcast(data, rank)
		if err != nil {
			fmt.Printf("Error in broadcasting votes: %v, retry in 5 seconds\n", err)
			time.Sleep(5000 * time.Millisecond)
			continue
		}

		return response, nil
	}

}

// Ceiling for Byzantine fault tolerance
func computeQuorum(n int) int { return (2*n + 2) / 3 }
