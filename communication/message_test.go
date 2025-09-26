package communication

import (
	"crypto/ed25519"
	"encoding/json"
	"testing"
)

// Test makeMsgID is non-empty and produces different values across calls.
func TestMakeMsgID(t *testing.T) {
	id1, err := makeMsgID()
	if err != nil {
		t.Fatalf("makeMsgID failed: %v", err)
	}
	if id1 == "" {
		t.Fatalf("makeMsgID returned empty id")
	}
	id2, err := makeMsgID()
	if err != nil {
		t.Fatalf("makeMsgID failed second call: %v", err)
	}
	if id1 == id2 {
		t.Fatalf("makeMsgID returned same id twice: %s", id1)
	}
}

// Test validateBanCertificate positive and negative cases
func TestValidateBanCertificate(t *testing.T) {
	// create two keypairs (accuser and voter)
	pubA, _ := mustKeypair(t)
	pubB, privB := mustKeypair(t)

	// node with known players (voter is "voter")
	playersPK := map[string]ed25519.PublicKey{
		"accused": pubA,
		"voter":   pubB,
	}
	node := &Node{PlayersPK: playersPK, N: 2, quorum: 1, proposals: make(map[string]ProposalMsg), votes: make(map[string]map[string]VoteMsg)}

	// make a proposal id
	pid, err := makeMsgID()
	if err != nil {
		t.Fatalf("makeMsgID failed: %v", err)
	}

	// create a valid reject vote signed by voter
	toSign, _ := json.Marshal(struct {
		ProposalID string    `json:"proposal_id"`
		VoterID    string    `json:"voter_id"`
		Value      VoteValue `json:"value"`
	}{pid, "voter", VoteReject})
	sig := ed25519.Sign(privB, toSign)
	vote := VoteMsg{ProposalID: pid, VoterID: "voter", Value: VoteReject, Reason: "test", Sig: sig}
	ban := makeBanCertificate(pid, "accused", "reason", []VoteMsg{vote})

	ok, err := node.validateBanCertificate(ban)
	if err != nil || !ok {
		t.Fatalf("expected valid ban cert, got ok=%v err=%v", ok, err)
	}

	// Negative: not enough votes
	node.quorum = 2
	ok, err = node.validateBanCertificate(ban)
	if err == nil || ok {
		t.Fatalf("expected error for not enough votes")
	}
	node.quorum = 1

	// Negative: unknown voter
	badVote := vote
	badVote.VoterID = "unknown"
	ok, err = node.validateBanCertificate(makeBanCertificate(pid, "accused", "reason", []VoteMsg{badVote}))
	if err == nil || ok {
		t.Fatalf("expected unknown voter error")
	}

	// Negative: bad signature
	badVote2 := vote
	badVote2.Sig = []byte("bad")
	ok, err = node.validateBanCertificate(makeBanCertificate(pid, "accused", "reason", []VoteMsg{badVote2}))
	if err == nil || ok {
		t.Fatalf("expected bad signature error")
	}

	// Negative: wrong value (ACCEPT instead of REJECT)
	toSignAccept, _ := json.Marshal(struct {
		ProposalID string    `json:"proposal_id"`
		VoterID    string    `json:"voter_id"`
		Value      VoteValue `json:"value"`
	}{pid, "voter", VoteAccept})
	sigAccept := ed25519.Sign(privB, toSignAccept)
	voteAccept := VoteMsg{ProposalID: pid, VoterID: "voter", Value: VoteAccept, Reason: "test", Sig: sigAccept}
	ok, err = node.validateBanCertificate(makeBanCertificate(pid, "accused", "reason", []VoteMsg{voteAccept}))
	if err == nil || ok {
		t.Fatalf("expected wrong value error")
	}

	// Negative: proposal id mismatch
	toSign2, _ := json.Marshal(struct {
		ProposalID string    `json:"proposal_id"`
		VoterID    string    `json:"voter_id"`
		Value      VoteValue `json:"value"`
	}{"other", "voter", VoteReject})
	sig2 := ed25519.Sign(privB, toSign2)
	voteMismatch := VoteMsg{ProposalID: "other", VoterID: "voter", Value: VoteReject, Reason: "test", Sig: sig2}
	ok, err = node.validateBanCertificate(makeBanCertificate(pid, "accused", "reason", []VoteMsg{voteMismatch}))
	if err == nil || ok {
		t.Fatalf("expected proposal id mismatch error")
	}
}
