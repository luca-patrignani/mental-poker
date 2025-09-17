package communication

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

func makeMsgID() (string, error) {
	ts := time.Now().UnixNano()

	randBytes := make([]byte, 16) // 128 bits entropy
	_, err := rand.Read(randBytes)
	if err != nil {
		return "", err
	}
	raw := fmt.Sprintf("%d-%x", ts, randBytes)
	hash := sha256.Sum256([]byte(raw))

	return hex.EncodeToString(hash[:8]), nil
}

type ProposalMsg struct {
	ProposalID string  `json:"proposal_id"`
	Action     *Action `json:"action"`
	Signature  []byte  `json:"sig"`
}

// Construnctor for ProposalMsg
func makeProposalMsg(a *Action, sig []byte) ProposalMsg {
	id, _ := makeMsgID()
	return ProposalMsg{ProposalID: id, Action: a, Signature: sig}
}

func proposalID(a *Action) (string, error) {
	b, err := a.signingBytes()
	if err != nil {
		return "", err
	}
	return Sha256Hex(b), nil
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

func makeVoteMsg(proposalID string, voterID string, value VoteValue, reason string) VoteMsg {
	return VoteMsg{
		ProposalID: proposalID,
		VoterID:    voterID,
		Value:      value,
		Reason:     reason,
		Sig:        nil, // to be set after signing
	}
}

// CommitCertificate = Proposal + quorum votes
type CommitCertificate struct {
	Proposal *ProposalMsg `json:"proposal"`
	Votes    []VoteMsg    `json:"votes"`
}

func makeCommitCertificate(prop *ProposalMsg, votes []VoteMsg) CommitCertificate {
	return CommitCertificate{Proposal: prop, Votes: votes}
}

// BanCertificate contains the evidence that a given player behaved maliciously
type BanCertificate struct {
	ProposalID string    `json:"proposal_id"`
	Accused    string    `json:"accused"`
	Reason     string    `json:"reason"`
	Votes      []VoteMsg `json:"votes"`
}

// makeBanCertificate constructs a BanCertificate from the collected reject votes
func makeBanCertificate(proposalID string, accused string, reason string, votes []VoteMsg) BanCertificate {
	return BanCertificate{ProposalID: proposalID, Accused: accused, Reason: reason, Votes: votes}
}

func (node *Node) validateBanCertificate(cert BanCertificate) (bool, error) {
	if len(cert.Votes) < node.quorum {
		return false, fmt.Errorf("not enough votes in ban cert")
	}
	for _, v := range cert.Votes {
		// check voter pubkey exists
		pub, ok := node.PlayersPK[v.VoterID]
		if !ok {
			return false, fmt.Errorf("unknown voter %s", v.VoterID)
		}
		// reconstruct toSign for vote verification
		toSign, _ := json.Marshal(struct {
			ProposalID string    `json:"proposal_id"`
			VoterID    string    `json:"voter_id"`
			Value      VoteValue `json:"value"`
		}{v.ProposalID, v.VoterID, v.Value})
		if !ed25519.Verify(pub, toSign, v.Sig) {
			return false, fmt.Errorf("bad vote signature from %s", v.VoterID)
		}
		if v.Value != VoteReject {
			return false, fmt.Errorf("vote from %s is not reject", v.VoterID)
		}
		if v.ProposalID != cert.ProposalID {
			return false, fmt.Errorf("vote proposal mismatch")
		}
	}
	return true, nil
}
