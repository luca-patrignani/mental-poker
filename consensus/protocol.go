package consensus

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ProposeAction is called by the player who wants to act (the proposer)
func (node *ConsensusNode) ProposeAction(a *Action) error {

	idx := node.pokerSM.FindPlayerIndex(a.PlayerID)

	if idx < 0 {
		return fmt.Errorf("player not in session")
	}

	if idx != node.pokerSM.GetCurrentPlayer() {
		return fmt.Errorf("cannot propose out-of-turn")
	}

	// cache proposal
	node.proposal = a

	b, _ := json.Marshal(*node.proposal)
	if _, err := node.BroadcastwithTimeout(b, node.network.GetRank(), 30*time.Second); err != nil {
		return err
	}
	err := node.onReceiveProposal(node.proposal)
	if err != nil {
		return err
	}
	return nil
}

// WaitForProposal attende una proposta dal proposer corrente
func (cn *ConsensusNode) WaitForProposal() error {
	proposer := cn.pokerSM.GetCurrentPlayer()

	data, err := cn.BroadcastwithTimeout(nil, proposer, 30*time.Second)
	if err != nil {
		return err
	}
	var p Action
	if err := json.Unmarshal(data, &p); err != nil {
		fmt.Printf("failed to unmarshal action proposal: %v\n", err)
	}

	return cn.onReceiveProposal(&p)
}

// Calls this when a proposal arrives
func (node *ConsensusNode) onReceiveProposal(p *Action) error {
	//fmt.Printf("Node %s received proposal from player %s\n", node.ID, p.Action.PlayerID)

	if p.Payload == nil {
		return errors.New("empty poker action in proposal")
	}
	pub, find := node.playersPK[p.PlayerID]
	if !find {
		err := node.broadcastVoteForProposal(p, VoteReject, "unknown-player")
		if err != nil {
			return err
		}
		return nil
	}
	verified, _ := p.VerifySignature(pub)
	if !verified {
		err := node.broadcastVoteForProposal(p, VoteReject, "bad-signature")
		if err != nil {
			return err
		}
		return nil
	}

	invalid := node.pokerSM.Validate(p.Payload)
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

func (node *ConsensusNode) broadcastVoteForProposal(p *Action, v VoteValue, reason string) error {
	//fmt.Printf("Node %s voting %s for proposal from %s: %s\n", node.ID, v, p.Action.PlayerID, reason)

	vote := Vote{ActionId: p.Id,
		VoterID: node.network.GetRank(),
		Value:   v,
		Reason:  reason}

	err := vote.Sign(node.priv)
	if err != nil {
		return err
	}

	// cache proposal if missing
	if node.proposal == nil {
		node.proposal = p
	}

	node.votes[node.network.GetRank()] = vote

	//fmt.Printf("Node %s broadcasting vote %s for proposal %s\n", node.ID, v, pid)
	b, _ := json.Marshal(vote)
	votesBytes, err := node.AllToAllwithTimeout(b, 30*time.Second)
	if err != nil {
		return err
	}

	votes := make([]Vote, 0, len(votesBytes))
	for _, vb := range votesBytes {
		var v Vote
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

func ensureSameProposal(votes []Vote) error {
	if len(votes) == 0 {
		return fmt.Errorf("Votes array is empty")
	}

	firstProposal := votes[0].ActionId
	for _, v := range votes[1:] {
		if v.ActionId != firstProposal {
			return fmt.Errorf("Votes don't refer to the same proposal")
		}
	}
	return nil
}

func (node *ConsensusNode) onReceiveVotes(votes []Vote) error {
	err := ensureSameProposal(votes)
	if err != nil {
		fmt.Printf("Node %d received invalid votes: %v\n", node.network.GetRank(), err)
		return err
	}

	//fmt.Printf("Node %s processing %d votes\n", node.ID, len(votes))

	// cache valid votes
	for _, v := range votes {
		pub, present := node.playersPK[v.VoterID]
		if !present {
			fmt.Printf("unknown voter: %d\n", v.VoterID)
			continue
		}

		ok, err := v.VerifySignature(pub)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Printf("bad signature from %d\n", v.VoterID)
			continue
		}

		if _, ex := node.votes[v.VoterID]; !ex {
			fmt.Printf("Vote doesn't match any known player\n")
			continue
		}

		node.votes[v.VoterID] = v
	}

	// now check quorum
	err = node.checkAndCommit()
	if err != nil {
		return err
	}
	return nil

}

// checkAndCommit triggers commit if quorum is reached
func (node *ConsensusNode) checkAndCommit() error {

	if node.proposal == nil {
		return fmt.Errorf("missing proposal to commit\n")
	}

	accepts := len(collectVotes(node.votes, VoteAccept))
	rejectVotes := collectVotes(node.votes, VoteReject)
	rejects := len(rejectVotes)
	reason := getBanReason(rejectVotes)
	cert := Certificate{
		Proposal: node.proposal,
		Votes:    collectVotes(node.votes, "both"),
		Reason:   reason,
	}
	if accepts >= node.quorum {
		//fmt.Printf("Node %s committing proposal %s\n", node.ID, proposalID)

		err := node.applyCommit(cert)
		if err != nil {
			return err
		}
	} else if rejects >= node.quorum {
		//fmt.Printf("Node %s banning player due to s\n", node.ID)
		payload, err := node.pokerSM.NotifyBan(cert.Proposal.PlayerID)
		if err != nil {
			return err
		}
		cert.Proposal.Payload = payload
		err = node.applyCommit(cert)
		if err != nil {
			return err
		}
		if node.network.GetRank() == cert.Proposal.PlayerID {
			err := node.network.Close()
			if err != nil {
				return err
			}
			fmt.Printf("You have been banned for %s Shutting down Now\n", reason)
			return nil
		}
		delete(node.playersPK, cert.Proposal.PlayerID)
		node.quorum = computeQuorum(node.network.GetPeerCount())
	}
	return nil
}

func collectVotes(m map[int]Vote, filter VoteValue) []Vote {
	out := []Vote{}
	for _, v := range m {
		if v.Value == filter || filter == "both" {
			out = append(out, v)
		}
	}
	return out
}

func getBanReason(rejectVotes []Vote) string {
	reason := ""
	for _, vv := range rejectVotes {
		if reason != vv.Reason+"; " {
			reason += vv.Reason + "; "
		}
	}
	return reason
}

// applyCommit verifies certificate and applies the action deterministically
func (node *ConsensusNode) applyCommit(cert Certificate) error {
	//fmt.Printf("Node %s applying commit certificate for proposal %s\n", node.ID, cert.Proposal.Action.Type)
	if cert.Proposal == nil {
		return errors.New("bad certificate format")
	}
	node.pokerSM.Apply(cert.Proposal.Payload)

	votesBytes := make([][]byte, len(cert.Votes))
	for _, v := range cert.Votes {
		vb, err := json.Marshal(v)
		if err != nil {
			continue
		}
		votesBytes = append(votesBytes, vb)
	}

	err := node.ledger.Append(cert.Proposal.Payload, votesBytes, cert.Proposal.PlayerID, node.quorum)
	if err != nil {
		return err
	}
	return nil
}
