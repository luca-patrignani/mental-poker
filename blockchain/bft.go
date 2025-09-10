package blockchain

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/luca-patrignani/mental-poker/poker"
)

// ProposalMsg and VoteMsg types
type ProposalMsg struct {
	Action    *Action `json:"action"`
	Signature []byte  `json:"sig"` // signature of the action (redundant with Action.Signature but kept for clarity)
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
	Proposal  *ProposalMsg `json:"proposal"`
	Votes     []VoteMsg    `json:"votes"`
	Committed bool         `json:"committed"`
}

// Node represents a single peer in the P2P table
type Node struct {
	ID        string
	Pub       ed25519.PublicKey
	Priv      ed25519.PrivateKey
	PlayersPK map[string]ed25519.PublicKey // playerID -> pubkey
	N         int
	quorum    int

	// Session state (shared FSM)
	Session poker.Session

	// in-memory caches
	mtx           sync.Mutex
	proposals     map[string]ProposalMsg           // proposalID -> proposal
	votes         map[string]map[string]VoteMsg    // proposalID -> voterID -> vote
	commitHandler func(cert CommitCertificate)    // optional hook

	// network hook: you must implement Broadcast to send bytes to peers
	Broadcast func(msgType string, payload []byte) error
}

// NewNode constructs a Node. playersPK is the map of all player pubkeys (including this node)
func NewNode(id string, pub ed25519.PublicKey, priv ed25519.PrivateKey, playersPK map[string]ed25519.PublicKey) *Node {
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
	}
}

func ceil2n3(n int) int { return (2*n + 2) / 3 }

func Sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// ProposeAction is called by the player who wants to act (the proposer)
func (node *Node) ProposeAction(a *Action) error {
	// action should already be signed by the player
	pid, err := proposalID(a)
	if err != nil {
		return err
	}
	proposal := ProposalMsg{Action: a, Signature: a.Signature}
	// cache locally
	node.mtx.Lock()
	node.proposals[pid] = proposal
	node.mtx.Unlock()

	b, _ := json.Marshal(proposal)
	if node.Broadcast != nil {
		_ = node.Broadcast("proposal", b) // best-effort broadcast
	}
	// also locally process to send our own vote
	go node.onReceiveProposal(proposal)
	return nil
}

// network layer calls this when a proposal arrives
func (node *Node) onReceiveProposal(p ProposalMsg) {
	// verify action signature
	if p.Action == nil {
		return
	}
	pub, ok := node.PlayersPK[p.Action.PlayerID]
	if !ok {
		// unknown player
		node.broadcastVoteForProposal(p, VoteReject, "unknown-player")
		return
	}
	okv, _ := p.Action.VerifySignature(pub)
	if !okv {
		node.broadcastVoteForProposal(p, VoteReject, "bad-signature")
		return
	}
	// validate action against local session rules
	if err := node.validateActionAgainstSession(p.Action); err != nil {
		node.broadcastVoteForProposal(p, VoteReject, err.Error())
		return
	}
	// valid
	node.broadcastVoteForProposal(p, VoteAccept, "")
}

// helper to broadcast vote
func (node *Node) broadcastVoteForProposal(p ProposalMsg, v VoteValue, reason string) {
	pid, _ := proposalID(p.Action)
	vote := VoteMsg{
		ProposalID: pid,
		VoterID:    node.ID,
		Value:      v,
		Reason:     reason,
	}
	// sign minimal vote fields
	toSign, _ := json.Marshal(struct {
		ProposalID string    `json:"proposal_id"`
		VoterID    string    `json:"voter_id"`
		Value      VoteValue `json:"value"`
	}{pid, node.ID, v})
	sig := ed25519.Sign(node.Priv, toSign)
	vote.Sig = sig

	// cache proposal if missing
	node.mtx.Lock()
	if _, ex := node.proposals[pid]; !ex {
		node.proposals[pid] = p
	}
	if _, ex := node.votes[pid]; !ex {
		node.votes[pid] = make(map[string]VoteMsg)
	}
	node.votes[pid][node.ID] = vote
	node.mtx.Unlock()

	b, _ := json.Marshal(vote)
	if node.Broadcast != nil {
		_ = node.Broadcast("vote", b)
	}

	// locally process the vote (simulate immediate arrival)
	go node.onReceiveVote(vote)
}

func (node *Node) onReceiveVote(v VoteMsg) {
	node.mtx.Lock()
	defer node.mtx.Unlock()

	// verify vote signature
	pub, ok := node.PlayersPK[v.VoterID]
	if !ok {
		return
	}
	toSign, _ := json.Marshal(struct {
		ProposalID string    `json:"proposal_id"`
		VoterID    string    `json:"voter_id"`
		Value      VoteValue `json:"value"`
	}{v.ProposalID, v.VoterID, v.Value})
	if !ed25519.Verify(pub, toSign, v.Sig) {
		return
	}
	if _, ex := node.votes[v.ProposalID]; !ex {
		node.votes[v.ProposalID] = make(map[string]VoteMsg)
	}
	node.votes[v.ProposalID][v.VoterID] = v

	// compute counts
	accepts := 0
	rejects := 0
	for _, vv := range node.votes[v.ProposalID] {
		if vv.Value == VoteAccept {
			accepts++
		} else {
			rejects++
		}
	}

	// quick helper to fetch proposal
	prop, hasProp := node.proposals[v.ProposalID]

	// commit if accept quorum
	if accepts >= node.quorum && hasProp {
		// collect votes
		votes := collectVotes(node.votes[v.ProposalID], VoteAccept)
		cert := CommitCertificate{Proposal: &prop, Votes: votes, Committed: true}
		// broadcast commit certificate
		cb, _ := json.Marshal(cert)
		if node.Broadcast != nil {
			_ = node.Broadcast("commit", cb)
		}
		// apply locally
		_ = node.applyCommit(cert)
		return
	}
	// ban if reject quorum
	if rejects >= node.quorum && hasProp {
		// For brevity: here we simply print and mark ban. In production you build BanCertificate
		fmt.Printf("BAN quorum reached for proposal %s (proposer=%s)\n", v.ProposalID, prop.Action.PlayerID)
		// perform local removal
		node.removePlayerByID(prop.Action.PlayerID)
		return
	}
}

func collectVotes(m map[string]VoteMsg, filter VoteValue) []VoteMsg {
	out := []VoteMsg{}
	for _, v := range m {
		if v.Value == filter {
			out = append(out, v)
		}
	}
	return out
}

// applyCommit verifies certificate and applies the action deterministically
func (node *Node) applyCommit(cert CommitCertificate) error {
	if cert.Proposal == nil || cert.Proposal.Action == nil {
		return errors.New("bad cert")
	}
	// verify that we have enough votes (counted earlier but double-check)
	if len(cert.Votes) < node.quorum {
		return errors.New("not enough votes in certificate")
	}
	// verify action signature
	pub, ok := node.PlayersPK[cert.Proposal.Action.PlayerID]
	if !ok {
		return errors.New("unknown player in cert")
	}
	okv, _ := cert.Proposal.Action.VerifySignature(pub)
	if !okv {
		return errors.New("bad action signature in cert")
	}
	// apply to session deterministically
	playerIdx := node.findPlayerIndex(cert.Proposal.Action.PlayerID)
	if playerIdx == -1 {
		return errors.New("player not in session")
	}
	if err := node.applyActionToSession(cert.Proposal.Action, playerIdx); err != nil {
		return err
	}
	// update LastIndex
	node.Session.LastIndex++
	return nil
}

// findPlayerIndex helper
func (node *Node) findPlayerIndex(playerID string) int {
	for i, p := range node.Session.Players {
		if p.Name == playerID {
			return i
		}
	}
	return -1
}

// removePlayerByID removes a player from the session (deterministic) and adjusts turn
func (node *Node) removePlayerByID(playerID string) {
	node.mtx.Lock()
	defer node.mtx.Unlock()
	idx := node.findPlayerIndex(playerID)
	if idx == -1 {
		return
	}
	// remove player slice entry
	node.Session.Players = append(node.Session.Players[:idx], node.Session.Players[idx+1:]...)
	// adjust CurrentTurn if necessary
	if int(node.Session.CurrentTurn) >= len(node.Session.Players) {
		node.Session.CurrentTurn = 0
	}
	// recompute quorum
	node.N = len(node.Session.Players)
	node.quorum = ceil2n3(node.N)
}

// applyActionToSession applies validated actions to the Session FSM
func (node *Node) applyActionToSession(a *Action, idx int) error {
	switch a.Type {
	case ActionFold:
		node.Session.Players[idx].HasFolded = true
		node.advanceTurnLocked()
	case ActionBet:
		node.Session.Players[idx].Bet += a.Amount
		if node.Session.Players[idx].Bet > node.Session.HighestBet {
			node.Session.HighestBet = node.Session.Players[idx].Bet
		}
		node.Session.Pot += a.Amount
		node.advanceTurnLocked()
	case ActionRaise:
		node.Session.Players[idx].Bet += a.Amount
		if node.Session.Players[idx].Bet > node.Session.HighestBet {
			node.Session.HighestBet = node.Session.Players[idx].Bet
		}
		node.Session.Pot += a.Amount
		node.advanceTurnLocked()
	case ActionCall:
		diff := node.Session.HighestBet - node.Session.Players[idx].Bet
		if diff > 0 {
			node.Session.Players[idx].Bet += diff
			node.Session.Pot += diff
		}
		node.advanceTurnLocked()
	case ActionCheck:
		if node.Session.Players[idx].Bet != node.Session.HighestBet {
			return fmt.Errorf("invalid check")
		}
		node.advanceTurnLocked()
	default:
		return fmt.Errorf("unknown action")
	}
	return nil
}

func (node *Node) advanceTurnLocked() {
	n := len(node.Session.Players)
	if n == 0 {
		return
	}
	for i := 1; i <= n; i++ {
		next := (int(node.Session.CurrentTurn) + i) % n
		if !node.Session.Players[next].HasFolded {
			node.Session.CurrentTurn = uint(next)
			return
		}
	}
}

// validateActionAgainstSession checks local rules (turn, amounts, round) and returns error if invalid
func (node *Node) validateActionAgainstSession(a *Action) error {
	// ensure round matches
	if a.RoundID != node.Session.RoundID {
		return fmt.Errorf("wrong round")
	}
	// check player exists
	idx := node.findPlayerIndex(a.PlayerID)
	if idx == -1 {
		return fmt.Errorf("player not in session")
	}
	// check it is player's turn
	if uint(idx) != node.Session.CurrentTurn {
		return fmt.Errorf("out-of-turn")
	}
	// amount checks for bet/raise
	if a.Type == ActionBet || a.Type == ActionRaise {
		if a.Amount == 0 {
			return fmt.Errorf("bad amount")
		}
	}
	return nil
}

// proposalID computes a stable id for a proposal
func proposalID(a *Action) (string, error) {
	b, err := a.Bytes()
	if err != nil {
		return "", err
	}
	return Sha256Hex(b), nil
}
