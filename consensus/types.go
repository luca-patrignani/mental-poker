package consensus

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// ConsensusNode è il nodo di consenso generico
type ConsensusNode struct {
	id     string
	pub    ed25519.PublicKey
	priv   ed25519.PrivateKey
	peers  []ed25519.PublicKey
	quorum int

	stateMachine StateMachine
	ledger       Ledger
	network      NetworkLayer

	proposals []ProposalMsg
	votes     []VoteMsg
}

// Action rappresenta un'azione generica nel sistema di consenso
type Action struct {
	Id        string `json:"id"`
	PlayerID   int    `json:"actor_id"`
	Payload   []byte `json:"payload"` // JSON serialized domain action
	Timestamp int64  `json:"ts"`
	Signature []byte `json:"sig,omitempty"`
}

func makeAction(actorId int, payload []byte) (Action, error) {
	randBytes := make([]byte, 16) // 128 bits entropy
	_, err := rand.Read(randBytes)
	if err != nil {
		return Action{}, err
	}
	raw := fmt.Sprintf("%d%x%x", actorId,payload, randBytes)
	b,_ := json.Marshal(raw)
	id := hex.EncodeToString(b[:8])

	return Action{
		Id: id,
		PlayerID: actorId,
		Payload: payload,
	}, nil
}

type ProposalMsg struct {
	ProposalID string  `json:"proposal_id"`
	Action     *Action `json:"action"`
	Signature  []byte  `json:"sig"`
}
type VoteValue string

const (
	VoteAccept VoteValue = "ACCEPT"
	VoteReject VoteValue = "REJECT"
)

type VoteMsg struct {
	ProposalID string    `json:"proposal_id"`
	VoterID    string    `json:"voter_id"`
	Value      VoteValue `json:"value"`
	Reason     string    `json:"reason,omitempty"`
	Sig        []byte    `json:"sig"`
}

// CommitCertificate = Proposal + quorum votes
type CommitCertificate struct {
	Proposal *ProposalMsg `json:"proposal"`
	Votes    []VoteMsg    `json:"votes"`
}

// BanCertificate contains the evidence that a given player behaved maliciously
type BanCertificate struct {
	ProposalID string    `json:"proposal_id"`
	Accused    string    `json:"accused"`
	Reason     string    `json:"reason"`
	Votes      []VoteMsg `json:"votes"`
}
