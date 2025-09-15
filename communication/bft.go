package communication

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/luca-patrignani/mental-poker/poker"
)

func Sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// ProposeAction is called by the player who wants to act (the proposer)
func (node *Node) ProposeAction(a *Action) error {

	idx := node.findPlayerIndex(a.PlayerID)
	if idx == -1 {
		return fmt.Errorf("player not in session")
	}
	if uint(idx) != node.Session.CurrentTurn {
		return fmt.Errorf("cannot propose out-of-turn")
	}
	pid, err := proposalID(a)
	if err != nil {
		return err
	}
	proposal := makeProposalMsg(a, a.Signature)

	// cache proposal
	node.proposals[pid] = proposal

	b, _ := json.Marshal(proposal)
	if _, err := node.peer.Broadcast(b, node.peer.Rank); err != nil {
		return err
	}
	err = node.onReceiveProposal(proposal)
	if err != nil {
		return err
	}
	return nil
}

// Calls this when a proposal arrives
func (node *Node) onReceiveProposal(p ProposalMsg) error {
	fmt.Printf("Node %s received proposal from player %s\n", node.ID, p.Action.PlayerID)

	if p.Action == nil {
		return errors.New("nil action in proposal")
	}
	pub, find := node.PlayersPK[p.Action.PlayerID]
	if !find {
		err := node.broadcastVoteForProposal(p, VoteReject, "unknown-player")
		if err != nil {
			return err
		}
		return nil
	}
	verified, _ := p.Action.VerifySignature(pub)
	if !verified {
		err := node.broadcastVoteForProposal(p, VoteReject, "bad-signature")
		if err != nil {
			return err
		}
		return nil
	}

	invalid := node.validateActionAgainstSession(p.Action)
	if invalid != nil {
		err := node.broadcastVoteForProposal(p, VoteReject, invalid.Error())
		if err != nil {
			return err
		}
		return nil
	}

	err := node.broadcastVoteForProposal(p, VoteAccept, "valid")
	if err != nil {
		return err
	}
	return nil
}

func (node *Node) broadcastVoteForProposal(p ProposalMsg, v VoteValue, reason string) error {
	fmt.Printf("Node %s voting %s for proposal from %s: %s\n", node.ID, v, p.Action.PlayerID, reason)

	pid, _ := proposalID(p.Action)
	vote := makeVoteMsg(pid, node.ID, v, reason)

	toSign, _ := json.Marshal(struct {
		ProposalID string    `json:"proposal_id"`
		VoterID    string    `json:"voter_id"`
		Value      VoteValue `json:"value"`
	}{pid, node.ID, v})

	sig := ed25519.Sign(node.Priv, toSign)
	vote.Sig = sig

	// cache proposal if missing
	if _, ex := node.proposals[pid]; !ex {
		node.proposals[pid] = p
	}
	if _, ex := node.votes[pid]; !ex {
		node.votes[pid] = make(map[string]VoteMsg)
	}
	node.votes[pid][node.ID] = vote

	fmt.Printf("Node %s broadcasting vote %s for proposal %s\n", node.ID, v, pid)
	b, _ := json.Marshal(vote)
	votesBytes, err := node.peer.AllToAll(b)
	if err != nil {
		return err
	}

	votes := make([]VoteMsg, 0, len(votesBytes))
	for _, vb := range votesBytes {
		var v VoteMsg
		if err := json.Unmarshal(vb, &v); err != nil {
			fmt.Printf("failed to unmarshal vote: %v\n", err)
			continue // skip malformed messages
		}
		votes = append(votes, v)
	}

	err = node.onReceiveVotes(votes)
	if err != nil {
		return err
	}
	return nil
}

func ensureSameProposal(votes []VoteMsg) (error, string) {
	if len(votes) == 0 {
		return fmt.Errorf("Votes array is empty"), ""
	}

	firstProposal := votes[0].ProposalID
	for _, v := range votes[1:] {
		if v.ProposalID != firstProposal {
			return fmt.Errorf("Votes don't refer to the same proposal"), ""
		}
	}
	return nil, firstProposal
}

func (node *Node) onReceiveVotes(votes []VoteMsg) error {
	err, id := ensureSameProposal(votes)
	if err != nil {
		fmt.Printf("Node %s received invalid votes: %v\n", node.ID, err)
		return err
	}

	fmt.Printf("Node %s processing %d votes\n", node.ID, len(votes))

	// cache valid votes
	for _, v := range votes {
		pub, present := node.PlayersPK[v.VoterID]
		if !present {
			fmt.Printf("unknown voter %s\n", v.VoterID)
			continue
		}
		toSign, _ := json.Marshal(struct {
			ProposalID string    `json:"proposal_id"`
			VoterID    string    `json:"voter_id"`
			Value      VoteValue `json:"value"`
		}{v.ProposalID, v.VoterID, v.Value})

		if !ed25519.Verify(pub, toSign, v.Sig) {
			fmt.Printf("bad signature from %s\n", v.VoterID)
			continue
		}

		if _, ex := node.votes[v.ProposalID]; !ex {
			fmt.Printf("Vote doesn't match any known proposal\n")
			continue
		}

		node.votes[v.ProposalID][v.VoterID] = v
	}

	// now check quorum
	err = node.checkAndCommit(id)
	if err != nil {
		return err
	}
	return nil

}

// checkAndCommit triggers commit if quorum is reached
func (node *Node) checkAndCommit(proposalID string) error {
	prop, hasProp := node.proposals[proposalID]
	if !hasProp {
		return fmt.Errorf("missing proposal for id %s", proposalID)
	}

	accepts := len(collectVotes(node.votes[proposalID], VoteAccept))
	rejectVotes := collectVotes(node.votes[proposalID], VoteReject)
	rejects := len(rejectVotes)

	reason := ""
	for _, vv := range rejectVotes {
		if reason != vv.Reason {
			reason += vv.Reason + "; "
		} else {
			reason = vv.Reason
		}
	}

	if accepts >= node.quorum {
		fmt.Printf("Node %s committing proposal %s\n", node.ID, proposalID)
		cert := makeCommitCertificate(&prop, collectVotes(node.votes[proposalID], VoteAccept), true)
		err := node.applyCommit(cert)
		if err != nil {
			return err
		}
	} else if rejects >= node.quorum {
		fmt.Printf("Node %s banning player due to s\n", node.ID)
		bc := makeBanCertificate(proposalID, prop.Action.PlayerID, reason, collectVotes(node.votes[proposalID], VoteReject))
		err := node.handleBanCertificate(bc)
		if err != nil {
			return err
		}
	}
	return nil
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
	fmt.Printf("Node %s applying commit certificate for proposal %s\n", node.ID, cert.Proposal.Action.Type)
	if cert.Proposal == nil || cert.Proposal.Action == nil {
		return errors.New("bad certificate format")
	}
	// verify action signature
	pub, present := node.PlayersPK[cert.Proposal.Action.PlayerID]
	if !present {
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

// handleBanCertificate is invoked when this node receives a BanCertificate.
// If it's valid, removes the accused player.
func (node *Node) handleBanCertificate(cert BanCertificate) error {
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

// removePlayerByID removes a player from the session and adjusts turn
func (node *Node) removePlayerByID(playerID string, reason string) error {
	idx := node.findPlayerIndex(playerID)
	if idx == -1 {
		return fmt.Errorf("player %s to remove not found", playerID)
	}
	if node.ID == playerID {
		node.peer.Close()
		fmt.Printf("You have been banned for %s, shutting down Now\n", reason)
		return nil
	}
	// remove player slice entry
	//TODO: Check for problems when list index shift after removal
	node.Session.Players = append(node.Session.Players[:idx], node.Session.Players[idx+1:]...)

	// adjust CurrentTurn if necessary
	if int(node.Session.CurrentTurn) >= len(node.Session.Players) {
		node.Session.CurrentTurn = 0
	}
	// recompute quorum
	node.N = len(node.Session.Players)
	node.quorum = ceil2n3(node.N)
	fmt.Printf("Node %s removed player %s for %s, new N=%d quorum=%d\n", node.ID, playerID, reason, node.N, node.quorum)
	return nil
}

// applyActionToSession applies validated actions to the Session
func (node *Node) applyActionToSession(a *Action, idx int) error {
	err := checkPokerLogic(a, &node.Session, idx)
	if err != nil {
		return err
	}
	switch a.Type {
	case ActionFold:
		node.Session.Players[idx].HasFolded = true
		node.Session.RecalculatePots()
		node.advanceTurn()
	case ActionBet:
		node.Session.Players[idx].Bet += a.Amount
		node.Session.Players[idx].Pot -= a.Amount
		if node.Session.Players[idx].Bet > node.Session.HighestBet {
			node.Session.HighestBet = node.Session.Players[idx].Bet
		}
		node.Session.RecalculatePots()
		node.advanceTurn()
	case ActionRaise:
		node.Session.Players[idx].Bet += a.Amount
		node.Session.Players[idx].Pot -= a.Amount
		node.Session.HighestBet = node.Session.Players[idx].Bet
		node.Session.RecalculatePots()
		node.advanceTurn()
	case ActionCall:
		diff := node.Session.HighestBet - node.Session.Players[idx].Bet
		node.Session.Players[idx].Bet += diff
		node.Session.Players[idx].Pot -= diff
		node.Session.RecalculatePots()
		node.advanceTurn()
	case ActionAllIn:
		node.Session.Players[idx].Bet += node.Session.Players[idx].Pot
		node.Session.Players[idx].Pot = 0
		if node.Session.Players[idx].Bet >= node.Session.HighestBet {
			node.Session.HighestBet = node.Session.Players[idx].Bet
		}
		node.Session.RecalculatePots()
		node.advanceTurn()

	case ActionCheck:
		node.advanceTurn()
	default:
		return fmt.Errorf("unknown action")
	}
	return nil
}

func (node *Node) advanceTurn() {
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

func (node *Node) validateActionAgainstSession(a *Action) error {
	if a.RoundID != node.Session.RoundID {
		return fmt.Errorf("wrong round")
	}
	idx := node.findPlayerIndex(a.PlayerID)
	if idx == -1 {
		return fmt.Errorf("player not in session")
	}
	if uint(idx) != node.Session.CurrentTurn {
		return fmt.Errorf("out-of-turn")
	}

	err := checkPokerLogic(a, &node.Session, idx)
	if err != nil {
		return err
	}

	return nil
}

func checkPokerLogic(a *Action, session *poker.Session, idx int) error {
	switch a.Type {
	case ActionFold:
		return nil
	case ActionBet:
		if session.Players[idx].Pot < a.Amount {
			return fmt.Errorf("insufficient funds")
		}
	case ActionRaise:
		if session.Players[idx].Bet < session.HighestBet {
			return fmt.Errorf("raise must at least match highest bet")
		}
	case ActionCall:
		diff := session.HighestBet - session.Players[idx].Bet
		if diff > session.Players[idx].Pot {
			return fmt.Errorf("insufficient funds to call")
		}
	case ActionAllIn:
		remaining := session.Players[idx].Pot + session.Players[idx].Bet
		if remaining != a.Amount {
			return fmt.Errorf("allin amount must match player's remaining pot")
		}
	case ActionCheck:
		if session.Players[idx].Bet != session.HighestBet {
			return fmt.Errorf("cannot check, must call, raise or fold")
		}
	default:
		return fmt.Errorf("unknown action")
	}
	return nil
}
