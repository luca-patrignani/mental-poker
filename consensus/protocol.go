package consensus

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/luca-patrignani/mental-poker/domain/poker"
)

type Action struct {
	Id        string            `json:"id"`
	PlayerID  int               `json:"actor_id"`
	Payload   poker.PokerAction `json:"payload"` //domain action
	Timestamp int64             `json:"ts"`
	Signature []byte            `json:"sig,omitempty"`
}

// ToString returns the JSON string representation of the Action.
func (a *Action) ToString() string {
	b, _ := json.Marshal(a)
	return string(b)
}

// MakeAction creates a new Action with a unique ID derived from the actor ID and action
// payload, along with random entropy. The ID is hex-encoded from the first 8 bytes of
// marshaled data. Timestamp is not set until Sign is called.
func MakeAction(actorId int, payload poker.PokerAction) (Action, error) {
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

// ProposeAction broadcasts a poker action to all peers for consensus. The proposer must be
// the current player. The action is cached locally, broadcast to peers, and then processed
// through onReceiveProposal. Returns an error if the proposer is not the current player or
// if the broadcast fails.
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
	if _, err := node.network.BroadcastwithTimeout(b, node.network.GetRank(), 30*time.Second); err != nil {
		return err
	}
	err := node.onReceiveProposal(node.proposal)
	if err != nil {
		return err
	}
	return nil
}

// WaitForProposal blocks until a proposal is received from the current player and processes it.
// It receives the proposal via Broadcast from the proposer's rank and validates it through
// onReceiveProposal. Returns an error if the broadcast fails or the proposal cannot be unmarshaled.
func (node *ConsensusNode) WaitForProposal() error {
	proposer := node.pokerSM.GetCurrentPlayer()

	data, err := node.network.BroadcastwithTimeout(nil, proposer, 30*time.Second)
	if err != nil {
		return err
	}
	var p Action
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("failed to unmarshal action proposal: %v\n", err)
	}

	return node.onReceiveProposal(&p)
}

// onReceiveProposal validates a received action proposal by checking the proposer's signature,
// verifying player existence, and validating poker rules. It then broadcasts a vote
// (ACCEPT or REJECT) based on the validation result. Caches the proposal if missing.
func (node *ConsensusNode) onReceiveProposal(p *Action) error {
	//fmt.Printf("Node %s received proposal from player %s\n", node.ID, p.Action.PlayerID)

	pub, find := node.playersPK[p.PlayerID]
	for key, value := range node.playersPK {
		if pub.Equal(value)  {
			fmt.Printf("Key: %d, Value: %s\n", key, value)
		}
	}
	if !find {
		err := node.broadcastVoteForProposal(p, VoteReject, "unknown-player")
		if err != nil {
			return err
		}
		return nil
	}
	verified, err := p.VerifySignature(pub)
	if err != nil {
		return err
	}
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

	err = node.broadcastVoteForProposal(p, VoteAccept, "valid")
	if err != nil {
		return err
	}
	return nil
}

// broadcastVoteForProposal creates and broadcasts a signed vote for the proposal to all peers.
// It caches the vote locally, collects all votes from peers via AllToAll, and processes them
// through onReceiveVotes. Supports voting either ACCEPT or REJECT with a reason string.
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
	votesBytes, err := node.network.AllToAllwithTimeout(b, 30*time.Second)
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

// ensureSameProposal verifies that all votes in the slice reference the same action ID.
// Returns an error if the votes array is empty or if votes contain differing action IDs.
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

// onReceiveVotes processes a collection of votes by validating signatures, checking voter
// eligibility, caching valid votes, and triggering checkAndCommit. Skips votes with invalid
// signatures or unknown voters, logging the issues.
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

		if idx := node.pokerSM.FindPlayerIndex(v.VoterID); idx == -1 {
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

// checkAndCommit evaluates whether quorum has been reached for either accepting or rejecting
// the current proposal. If accepts >= quorum, commits the action. If rejects >= quorum,
// bans the proposer. Returns an error if neither quorum is reached or if commit fails.
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
		return nil
	} else if rejects >= node.quorum {
		//fmt.Printf("Node %s banning player due to s\n", node.ID)
		payload, err := node.pokerSM.NotifyBan(cert.Proposal.PlayerID)

		if err != nil {
			return err
		}
		cert.Proposal.Payload = payload
		err = node.applyCommit(cert, cert.Proposal)
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
		return nil
	}

	return fmt.Errorf("Not enough elegible votes to reach quorum yet, state not changed. (%d accepts, %d rejects, need %d)", accepts, rejects, node.quorum)
}

// collectVotes filters votes from the vote map by value. If filter is "both", returns all votes;
// otherwise returns only votes matching the specified VoteValue (ACCEPT or REJECT).
func collectVotes(m map[int]Vote, filter VoteValue) []Vote {
	out := []Vote{}
	for _, v := range m {
		if v.Value == filter || filter == "both" {
			out = append(out, v)
		}
	}
	return out
}

// getBanReason extracts unique rejection reasons from reject votes and concatenates them
// into a semicolon-separated string for logging or record keeping.
func getBanReason(rejectVotes []Vote) string {
	reason := ""
	for _, vv := range rejectVotes {
		if reason != vv.Reason+"; " {
			reason += vv.Reason + "; "
		}
	}
	return reason
}

// applyCommit applies a validated certificate by executing the state machine action,
// appending to the ledger, and removing the banned proposer from the peer map (if applicable).
// The optional ban parameter is used when the proposal represents a player banning.
func (node *ConsensusNode) applyCommit(cert Certificate, ban ...*Action) error {
	//fmt.Printf("Node %s applying commit certificate for proposal %s\n", node.ID, cert.Proposal.Action.Type)
	if cert.Proposal == nil {
		return errors.New("bad certificate format")
	}
	err := node.pokerSM.Apply(cert.Proposal.Payload)
	if err != nil {
		return err
	}

	ses := node.pokerSM.GetSession()

	if len(ban) > 0 {
		data := map[string]string{"rejectedAction": ban[0].ToString()}

		err = node.ledger.Append(*ses, cert.Proposal.Payload, cert.Votes, cert.Proposal.PlayerID, node.quorum, data)
		if err != nil {
			return err
		}
	} else {
		err := node.ledger.Append(*ses, cert.Proposal.Payload, cert.Votes, cert.Proposal.PlayerID, node.quorum)
		if err != nil {
			return err
		}
	}
	return nil
}
