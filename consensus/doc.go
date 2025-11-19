// Package consensus implements a Byzantine Fault Tolerant (BFT) consensus protocol
// for distributed poker games. It ensures all nodes agree on the sequence of game
// actions even in the presence of malicious or faulty players.
//
// # Architecture
//
// The consensus layer sits between the poker game logic and the network layer,
// validating and committing player actions through a voting mechanism. Each action
// must receive a quorum of votes before being applied to the game state.
//
// # Core Components
//
// ConsensusNode manages the consensus protocol for a single node, handling proposal
// creation, vote collection, and state transitions. It coordinates with the state
// machine (poker game), ledger (blockchain), and network layer.
//
// StateManager interface defines how the consensus layer interacts with game logic,
// providing validation and state transition methods.
//
// Ledger interface maintains an immutable, cryptographically-linked record of all
// consensus decisions for auditability and verification.
//
// NetworkLayer interface abstracts peer-to-peer communication, providing reliable
// broadcast and all-to-all communication patterns with synchronization.
//
// # Consensus Protocol Flow
//
// 1. Proposal Phase: The current player creates and signs an action, then broadcasts
//    it to all peers with a 30-second timeout.
//
// 2. Validation Phase: Each node independently validates the action against poker
//    rules and cryptographic signatures.
//
// 3. Voting Phase: Nodes broadcast signed votes (ACCEPT or REJECT) to all peers
//    using all-to-all communication.
//
// 4. Commitment Phase: Once a quorum is reached, the action is committed to the
//    ledger and applied to the game state. Invalid actions result in proposer banning.
//
// # Byzantine Fault Tolerance
//
// The protocol tolerates up to f Byzantine (malicious or faulty) nodes where:
//   f < n/3
//
// Quorum is calculated as: ceiling((2n+2)/3)
//
// This ensures that any two quorums intersect in at least one honest node,
// preventing conflicting decisions even with Byzantine failures.
//
// # Security Properties
//
// - Safety: Honest nodes never commit conflicting actions
// - Liveness: The protocol makes progress if > 2/3 nodes are honest and responsive
// - Accountability: All votes are cryptographically signed and recorded
// - Cheater detection: Invalid actions are detected and proposers are banned
//
// # Example Usage
//
//	// Initialize consensus node
//	node := consensus.NewConsensusNode(pub, priv, peerKeys, stateManager, ledger, network)
//	
//	// Exchange public keys with peers
//	if err := node.UpdatePeers(); err != nil {
//	    return err
//	}
//	
//	// Propose an action (when it's your turn)
//	action, _ := consensus.MakeAction(playerID, pokerAction)
//	action.Sign(privateKey)
//	if err := node.ProposeAction(&action); err != nil {
//	    return err
//	}
//	
//	// Wait for others' actions (when it's not your turn)
//	if err := node.WaitForProposal(); err != nil {
//	    return err
//	}
package consensus
