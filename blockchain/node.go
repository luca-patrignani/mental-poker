package blockchain

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/luca-patrignani/mental-poker/common"
	"github.com/luca-patrignani/mental-poker/poker"
)

// Node represents a single peer in the P2P table
type Node struct {
	ID        string
	Pub       ed25519.PublicKey
	Priv      ed25519.PrivateKey
	PlayersPK map[string]ed25519.PublicKey // playerID -> pubkey
	N         int
	quorum    int

	// Session state (shared)
	Session poker.Session

	proposals map[string]ProposalMsg        // proposalID -> proposal
	votes     map[string]map[string]VoteMsg // proposalID -> voterID -> vote

	peer *common.Peer
}

// NewNode constructs a Node. playersPK is the map of all player pubkeys (including this node)
func NewNode(id string, p *common.Peer, pub ed25519.PublicKey, priv ed25519.PrivateKey, playersPK map[string]ed25519.PublicKey) *Node {
	n := len(playersPK)
	return &Node{
		ID:        id,
		Pub:       pub,
		Priv:      priv,
		PlayersPK: playersPK,
		N:         n,
		quorum:    ceil2n3(n),
		proposals: make(map[string]ProposalMsg),
		votes:     make(map[string]map[string]VoteMsg),
		peer:      p,
	}
}

// Ceiling for Byzantine fault tolerance
func ceil2n3(n int) int { return (2*n + 2) / 3 }

// findPlayerIndex helper
func (node *Node) findPlayerIndex(playerID string) int {
	for i, p := range node.Session.Players {
		pID, err := strconv.Atoi(playerID)
		if err != nil {
			return -1
		}
		if p.Rank == pID {
			return i
		}
	}
	return -1
}

// WaitForProposalAndProcess blocks until the barrier returns the proposal sent by the
// current proposer (node.Session.CurrentTurn).
//
// This function is intended to be called by non-proposer nodes when they are in the
// "waiting for proposal" phase.
func (node *Node) WaitForProposalAndProcess() error {
	// compute proposer rank from the session state
	proposerRank := int(node.Session.CurrentTurn)
	// call barrier-style Broadcast to receive the proposal bytes
	recv, err := node.peer.Broadcast(nil, proposerRank)
	if err != nil {
		return fmt.Errorf("wait for proposal failed: %w", err)
	}
	// unmarshal into ProposalMsg
	var p ProposalMsg
	if err := json.Unmarshal(recv, &p); err != nil {
		return fmt.Errorf("invalid proposal bytes: %w", err)
	}
	// process locally (vote)
	err = node.onReceiveProposal(p)
	if err != nil {
		return err
	}
	return nil
}
