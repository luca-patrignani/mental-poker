package consensus

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"

	"github.com/luca-patrignani/mental-poker/domain/poker"
)

type StateManager interface {
	Validate(payload poker.PokerAction) error

	Apply(payload poker.PokerAction) error

	GetCurrentPlayer() int

	FindPlayerIndex(id int) int

	NotifyBan(id int) (poker.PokerAction, error)

	GetSession() *poker.Session
}

type Ledger interface {
	Append(session poker.Session, action poker.PokerAction, votes []Vote, proposerID int, quorum int, extra ...map[string]string) error

	Verify() error
}

// NetworkLayer abstract P2P
type NetworkLayer interface {
	Broadcast(data []byte, root int) ([]byte, error)

	AllToAll(data []byte) ([][]byte, error)

	GetRank() int

	GetPeerCount() int

	Close() error
}

type ConsensusNode struct {
	pub       ed25519.PublicKey
	priv      ed25519.PrivateKey
	playersPK map[int]ed25519.PublicKey
	quorum    int

	pokerSM StateManager
	ledger  Ledger
	network NetworkLayer

	proposal *Action
	votes    map[int]Vote
}

// NewConsensusNode creates and initializes a new consensus node with the given cryptographic keys,
// peer public keys, state machine, ledger, and network layer. It sets up the node's quorum
// threshold using Byzantine Fault Tolerance calculations (2n+2)/3.
func NewConsensusNode(
	pub ed25519.PublicKey,
	priv ed25519.PrivateKey,
	peers map[int]ed25519.PublicKey,
	sm StateManager,
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
		votes:     map[int]Vote{},
	}
}

func (node ConsensusNode) GetPriv() ed25519.PrivateKey {
	return node.priv
}

func (node *ConsensusNode) RemoveNode(leaver int) {
	delete(node.playersPK, leaver)
	node.quorum = computeQuorum(len(node.playersPK))
}

// UpdatePeers exchanges public keys with all peers in an AllToAll operation and updates
// the node's peer mapping and quorum threshold accordingly.
func (node *ConsensusNode) UpdatePeers() error {
	b, err := json.Marshal(node.pub)
	if err != nil {
		return err
	}
	pkBytes, err := node.network.AllToAll(b)
	if err != nil {
		return err
	}
	pk := make(map[int]ed25519.PublicKey, len(pkBytes))
	for i, pki := range pkBytes {
		var p ed25519.PublicKey
		if err := json.Unmarshal(pki, &p); err != nil {
			return fmt.Errorf("failed to unmarshal public key: %v", err)
		}
		pk[i] = p
	}
	node.playersPK = pk
	node.quorum = computeQuorum(len(pk))
	return nil
}

// computeQuorum calculates the minimum number of votes required to reach Byzantine Fault
// Tolerance consensus. It returns ceiling((2n+2)/3) where n is the number of nodes.
func computeQuorum(n int) int { return (2*n + 2) / 3 }
