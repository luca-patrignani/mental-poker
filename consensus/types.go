package consensus

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/luca-patrignani/mental-poker/domain/poker"
)

// ConsensusNode è il nodo di consenso generico
type ConsensusNode struct {
	pub       ed25519.PublicKey
	priv      ed25519.PrivateKey
	playersPK map[int]ed25519.PublicKey
	quorum    int

	pokerSM StateMachine
	ledger  Ledger
	network NetworkLayer

	proposal *Action
	votes    map[int]Vote
}

// Action rappresenta un'azione generica nel sistema di consenso
type Action struct {
	Id        string            `json:"id"`
	PlayerID  int               `json:"actor_id"`
	Payload   poker.PokerAction `json:"payload"` // JSON serialized domain action
	Timestamp int64             `json:"ts"`
	Signature []byte            `json:"sig,omitempty"`
}

func makeAction(actorId int, payload poker.PokerAction) (Action, error) {
	randBytes := make([]byte, 16) // 128 bits entropy
	_, err := rand.Read(randBytes)
	if err != nil {
		return Action{}, err
	}
	raw := fmt.Sprintf("%d%x%x", actorId, payload, randBytes)
	b, _ := json.Marshal(raw)
	id := hex.EncodeToString(b[:8])

	return Action{
		Id:       id,
		PlayerID: actorId,
		Payload:  payload,
	}, nil
}

type VoteValue string

const (
	VoteAccept VoteValue = "ACCEPT"
	VoteReject VoteValue = "REJECT"
)

type Vote struct {
	ActionId  string    `json:"proposal_id"`
	VoterID   int       `json:"voter_id"`
	Value     VoteValue `json:"value"`
	Reason    string    `json:"reason,omitempty"`
	Signature []byte    `json:"signature,omitempty"`
}

// Certificate = Proposal + quorum votes
// Certificate for commit action (including banning)
type Certificate struct {
	Proposal *Action `json:"proposal"`
	Votes    []Vote  `json:"votes"`
	Reason   string  `json:"reason,omitempty"`
}
