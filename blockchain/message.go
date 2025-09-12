package blockchain

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
	// Timestamp in nanoseconds
	ts := time.Now().UnixNano()

	// Add random entropy
	randBytes := make([]byte, 16) // 128 bits entropy
	_, err := rand.Read(randBytes)
	if err != nil {
		return "", err
	}

	// Combine inputs into one string
	raw := fmt.Sprintf("%d-%x", ts, randBytes)

	// Hash it
	hash := sha256.Sum256([]byte(raw))

	// Encode as hex string (can shorten if desired)
	return hex.EncodeToString(hash[:8]), nil
}

// ProposalMsg and VoteMsg types
type ProposalMsg struct {
	Type       string  `json:"type,omitempty"` // "proposal"
	ProposalID string  `json:"proposal_id"`
	Action     *Action `json:"action"`
	Signature  []byte  `json:"sig"` // signature of the action (redundant with Action.Signature but kept for clarity)
}

func makeProposalMsg(a *Action, sig []byte) ProposalMsg {
	id, _ := makeMsgID()
	return ProposalMsg{Type: "proposal", ProposalID: id, Action: a, Signature: sig}
}

type VoteValue string

const (
	VoteAccept VoteValue = "ACCEPT"
	VoteReject VoteValue = "REJECT"
)

type VoteMsg struct {
	Type       string    `json:"type,omitempty"` // "Vote"
	ProposalID string    `json:"proposal_id"`
	VoterID    string    `json:"voter_id"`
	Value      VoteValue `json:"value"`
	Reason     string    `json:"reason,omitempty"`
	Sig        []byte    `json:"sig"`
}

func makeVoteMsg(proposalID string, voterID string, value VoteValue, reason string) VoteMsg {
	return VoteMsg{
		Type:       "vote",
		ProposalID: proposalID,
		VoterID:    voterID,
		Value:      value,
		Reason:     reason,
		Sig:        nil, // to be set after signing
	}
}

// CommitCertificate = Proposal + quorum votes
type CommitCertificate struct {
	Type      string       `json:"type,omitempty"` // "commit"
	Proposal  *ProposalMsg `json:"proposal"`
	Votes     []VoteMsg    `json:"votes"`
	Committed bool         `json:"committed"`
}

func makeCommitCertificate(prop *ProposalMsg, votes []VoteMsg, commit bool) CommitCertificate {
	return CommitCertificate{Type: "commit", Proposal: prop, Votes: votes, Committed: commit}
}

// BanCertificate contains the evidence that a given player behaved maliciously
// w.r.t. a particular proposal. It includes the proposal ID, accused player and
// the rejecting votes (raw VoteMsg) that form the evidence.
type BanCertificate struct {
	Type       string    `json:"type,omitempty"` // "ban"
	ProposalID string    `json:"proposal_id"`
	Accused    string    `json:"accused"`
	Reason     string    `json:"reason"`
	Votes      []VoteMsg `json:"votes"`
}

// makeBanCertificate constructs a BanCertificate from the collected reject votes
func makeBanCertificate(proposalID string, accused string, reason string, votes []VoteMsg) BanCertificate {
	return BanCertificate{Type: "ban", ProposalID: proposalID, Accused: accused, Reason: reason, Votes: votes}
}

// validateBanCertificate checks that:
// - the votes are signed by known players
// - each vote references the same proposalID and has Value==VoteReject
// - there are at least quorum votes
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

// handleBanCertificate is invoked when this node receives a BanCertificate.
// If it's valid, removes the accused player deterministically.
func (node *Node) handleBanCertificate(cert BanCertificate) error {
	fmt.Printf("Node %s: handling ban cert against player %s \n", node.ID, cert.Accused)
	ok, err := node.validateBanCertificate(cert)
	if err != nil || !ok {
		return fmt.Errorf("invalid ban certificate: %w", err)
	}

	err = node.removePlayerByID(cert.Accused, cert.Reason)
	if err != nil {
		return err
	}
	return nil
}
