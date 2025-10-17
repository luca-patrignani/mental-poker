package ledger

import (
	"github.com/luca-patrignani/mental-poker/consensus"
	"github.com/luca-patrignani/mental-poker/domain/poker"
)

// Block rappresenta un blocco nella blockchain
type Block struct {
	Index     int               `json:"index"`
	Timestamp int64             `json:"timestamp"`
	PrevHash  string            `json:"prev_hash"`
	Hash      string            `json:"hash"`
	Session   poker.Session     `json:"session"`
	Action    poker.PokerAction `json:"poker_action"` // Generic action data
	Votes     []consensus.Vote  `json:"votes"`        // Quorum votes
	Metadata  Metadata          `json:"metadata"`
}

type Metadata struct {
	ProposerID int               `json:"proposer_id"`
	Quorum     int               `json:"quorum"`
	Extra      map[string]string `json:"extra,omitempty"`
}
