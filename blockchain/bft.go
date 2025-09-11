package blockchain

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

func ceil2n3(n int) int { return (2*n + 2) / 3 }

func Sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// ProposeAction is called by the player who wants to act (the proposer)
func (node *Node) ProposeAction(a *Action) error {
	
	idx := node.findPlayerIndex(a.PlayerID)
	if idx == -1 { return fmt.Errorf("player not in session") }
	// check it's player's turn
	if uint(idx) != node.Session.CurrentTurn {
	    return fmt.Errorf("cannot propose out-of-turn")
	}
	// action should already be signed by the player
	pid, err := proposalID(a)
    if err != nil {
        return err
    }
    proposal := makeProposalMsg(a, a.Signature)

    // cache locally
    node.mtx.Lock()
    node.proposals[pid] = proposal
    node.mtx.Unlock()

    b, _ := json.Marshal(proposal)
    // proposer uses its own rank as root
    if _, err := node.peer.Broadcast(b, node.peer.Rank); err != nil {
        return err
    }
        // locally process to send our own vote
    node.onReceiveProposal(proposal)

  
    return nil

}

// network layer calls this when a proposal arrives
func (node *Node) onReceiveProposal(p ProposalMsg) {
	print("Arrivata proposta\n")
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
	node.broadcastVoteForProposal(p, VoteAccept, "valid")
}

// helper to broadcast vote
func (node *Node) broadcastVoteForProposal(p ProposalMsg, v VoteValue, reason string) error {
	fmt.Printf("Node %s voting %s for proposal from %s: %s\n", node.ID, v, p.Action.PlayerID, reason)
	pid, _ := proposalID(p.Action)
	vote := makeVoteMsg(pid,node.ID,v,reason)
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
	
	fmt.Printf("Node %s received %d votes from AllToAll\n", node.ID, len(votes))
	node.onReceiveVotes(votes)
	return nil
}

func ensureSameProposal(votes []VoteMsg) (error, string) {
    if len(votes) == 0 {
        return fmt.Errorf("Votes array is empty"), "" // no votes means invalid
    }

    firstProposal := votes[0].ProposalID
    for _, v := range votes[1:] {
        if v.ProposalID != firstProposal {
            return fmt.Errorf("Votes don't refer to the same proposal"), ""
        }
    }
    return nil, firstProposal
}

// onReceiveVotes processes multiple votes at once
func (node *Node) onReceiveVotes(votes []VoteMsg) {
	err,id := ensureSameProposal(votes)
	if err != nil {
		fmt.Printf("Node %s received invalid votes: %v\n", node.ID, err)
		return
	}

	fmt.Printf("Node %s processing %d votes\n", node.ID, len(votes))
    node.mtx.Lock()
    defer node.mtx.Unlock()

    for _, v := range votes {
        // verify signature
        pub, ok := node.PlayersPK[v.VoterID]
        if !ok {
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
            node.votes[v.ProposalID] = make(map[string]VoteMsg)
        }
        node.votes[v.ProposalID][v.VoterID] = v
    }

    // now check quorum for all proposals included in this batch
    node.checkAndCommit(id)
    
}

// checkAndCommit triggers commit if quorum is reached
func (node *Node) checkAndCommit(proposalID string) {
    prop, hasProp := node.proposals[proposalID]
    if !hasProp {
        return
    }

    accepts := 0
    rejects := 0
    for _, vv := range node.votes[proposalID] {
        if vv.Value == VoteAccept {
            accepts++
        } else {
            rejects++
        }
    }

    if accepts >= node.quorum {
		fmt.Printf("Node %s committing proposal %s\n", node.ID, proposalID)
        cert := makeCommitCertificate(&prop, collectVotes(node.votes[proposalID], VoteAccept), true)
        _ = node.applyCommit(cert)
    } else if rejects >= node.quorum {
		fmt.Printf("Node %s banning player from proposal %s\n", node.ID, proposalID)
        bc := makeBanCertificate(proposalID, prop.Action.PlayerID, collectVotes(node.votes[proposalID], VoteReject))
        node.handleBanCertificate(bc)
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
	fmt.Printf("Node %s applying commit certificate for proposal %s\n", node.ID, cert.Proposal.Type)
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
		pID,err := strconv.Atoi(playerID)
		if err != nil {
			return -1
		}
		if p.Rank == pID {
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
		if node.Session.Players[idx].Pot < a.Amount {
        return fmt.Errorf("insufficient funds")
    	}
    	node.Session.Players[idx].Pot -= a.Amount    
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
	b, err := a.signingBytes()
	if err != nil {
		return "", err
	}
	return Sha256Hex(b), nil
}
