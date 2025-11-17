package consensus

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"time"

	"github.com/luca-patrignani/mental-poker/domain/poker"
)

// StateManager defines the interface for managing poker game state transitions.
// Implementations must provide methods to validate actions, apply state changes,
// track current players, and handle player banning.
type StateManager interface {
	// Validate checks if the given poker action is valid in the current game state.
	// It verifies turn order, action legality, and game rules compliance.
	// Returns an error describing why the action is invalid, or nil if valid.
	Validate(payload poker.PokerAction) error

	// Apply executes a validated poker action, modifying the game state.
	// This method should only be called after successful validation.
	// Returns an error if the state transition fails.
	Apply(payload poker.PokerAction) error

	// GetCurrentPlayer returns the player ID whose turn it is to act.
	// Returns -1 if no valid player can be determined.
	GetCurrentPlayer() int

	// FindPlayerIndex returns the session index of the player with the given ID.
	// Returns -1 if the player is not found in the current session.
	FindPlayerIndex(id int) int

	// NotifyBan creates a ban action for the specified player.
	// This is called when consensus determines a player should be removed.
	// Returns the ban action or an error if the player cannot be banned.
	NotifyBan(id int) (poker.PokerAction, error)

	// GetSession returns a pointer to the current poker session state.
	GetSession() *poker.Session
}

// Ledger defines the interface for maintaining an immutable log of consensus decisions.
// Implementations should provide append-only semantics with cryptographic verification.
type Ledger interface {
	// Append adds a new consensus decision to the ledger.
	// The decision includes the game state, action, votes, proposer, and quorum threshold.
	// Optional extra metadata can be included (e.g., ban reasons).
	Append(session poker.Session, action poker.PokerAction, votes []Vote, proposerID int, quorum int, extra ...map[string]string) error

	// Verify checks the integrity of the entire ledger.
	// Returns an error if any tampering or inconsistency is detected.
	Verify() error
}

// NetworkLayer abstracts peer-to-peer communication primitives.
// Implementations must provide reliable broadcast and all-to-all communication
// patterns with optional timeout support.
type NetworkLayer interface {
	// Broadcast sends data from a specific node (identified by root) to all peers.
	// Returns the data received from the root node or an error if communication fails.
	Broadcast(data []byte, root int) ([]byte, error)

	// BroadcastwithTimeout performs a broadcast with a specified timeout.
	// Returns the received data or an error if the timeout is exceeded.
	BroadcastwithTimeout(data []byte, rank int, timeout time.Duration) ([]byte, error)

	// AllToAll sends data from this node to all peers and receives data from all peers.
	// Returns a slice where index i contains the data sent by peer i.
	AllToAll(data []byte) ([][]byte, error)

	// AllToAllwithTimeout performs an all-to-all exchange with a specified timeout.
	// Returns partial results if the timeout is exceeded.
	AllToAllwithTimeout(data []byte, timeout time.Duration) ([][]byte, error)

	// GetRank returns this node's unique identifier in the network.
	GetRank() int

	// GetPeerCount returns the total number of nodes including this one.
	GetPeerCount() int

	// Close gracefully shuts down the network layer and releases resources.
	Close() error
}

// ConsensusNode represents a node participating in the BFT consensus protocol.
// Each node maintains cryptographic keys, tracks peer public keys, manages
// consensus state, and coordinates with the state machine and ledger.
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
// peer public keys, state machine, ledger, and network layer.
//
// The quorum threshold is computed using Byzantine Fault Tolerance formula: ceiling((2n+2)/3),
// where n is the number of peers. This ensures safety with up to (n-1)/3 Byzantine failures.
//
// Parameters:
//   - pub: This node's Ed25519 public key for signature verification
//   - priv: This node's Ed25519 private key for signing actions and votes
//   - peers: Map of player IDs to their public keys
//   - sm: State machine implementing poker game logic
//   - ledger: Ledger for recording consensus decisions
//   - network: Network layer for P2P communication
//
// Returns a fully initialized consensus node ready to participate in the protocol.
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

// GetPriv returns this node's private key.
// Warning: Handle with care as this exposes sensitive cryptographic material.
func (node ConsensusNode) GetPriv() ed25519.PrivateKey {
	return node.priv
}

// RemoveNode removes a player from the consensus group and recalculates the quorum.
// This is called when a player leaves the game or is banned.
//
// Parameters:
//   - leaver: The player ID to remove from the consensus group
//
// The quorum is automatically adjusted to maintain BFT guarantees with the reduced peer set.
func (node *ConsensusNode) RemoveNode(leaver int) {
	delete(node.playersPK, leaver)
	node.quorum = computeQuorum(len(node.playersPK))
}

// UpdatePeers performs a full peer discovery by exchanging public keys with all nodes
// via an AllToAll operation. This synchronizes the peer mapping across all nodes.
//
// Returns an error if the key exchange fails or if any received key cannot be unmarshaled.
//
// This method should be called during initialization to ensure all nodes have
// consistent views of the network topology and cryptographic identities.
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
