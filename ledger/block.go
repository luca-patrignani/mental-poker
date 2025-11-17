package ledger

import (
	"github.com/luca-patrignani/mental-poker/consensus"
	"github.com/luca-patrignani/mental-poker/domain/poker"
)

// Block represents a single consensus decision in the blockchain.
// Each block contains the complete game state after an action, the votes
// that approved it, and cryptographic links to the previous block.
type Block struct {
	Index     int               `json:"index"`        // Sequential block number (0 = genesis)
	Timestamp int64             `json:"timestamp"`    // Unix timestamp when block was created
	PrevHash  string            `json:"prev_hash"`    // SHA256 hash of previous block
	Hash      string            `json:"hash"`         // SHA256 hash of this block
	Session   poker.Session     `json:"session"`      // Complete game state after action
	Action    poker.PokerAction `json:"poker_action"` // The action that was executed
	Votes     []consensus.Vote  `json:"votes"`        // Quorum votes approving this action
	Metadata  Metadata          `json:"metadata"`     // Additional consensus metadata
}

// Metadata contains consensus-specific information about a block.
type Metadata struct {
	ProposerID int               `json:"proposer_id"`    // ID of player who proposed the action
	Quorum     int               `json:"quorum"`         // Required votes for consensus
	Extra      map[string]string `json:"extra,omitempty"` // Optional metadata (e.g., ban reasons)
}