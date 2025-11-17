// Package consensus implements a Byzantine Fault Tolerant (BFT) consensus protocol
// for distributed poker games. It provides mechanisms for proposing, voting on, and
// committing poker actions across multiple nodes with cryptographic verification.
//
// The consensus layer ensures that all nodes agree on the sequence of game actions
// and can detect and ban malicious players who attempt invalid moves.
//
// # Core Components
//
// ConsensusNode: Manages the consensus protocol for a single node, including
// proposal creation, vote collection, and commitment logic.
//
// StateManager: Interface for game state validation and transitions.
//
// Ledger: Interface for maintaining an immutable log of consensus decisions.
//
// NetworkLayer: Interface for peer-to-peer communication primitives.
//
// # Consensus Protocol
//
// The protocol follows these steps:
//   1. Proposer broadcasts an action to all nodes
//   2. Each node validates the action independently
//   3. Nodes broadcast their votes (ACCEPT or REJECT)
//   4. Once quorum is reached, the action is committed
//   5. Invalid actions result in proposer being banned
//
// # Byzantine Fault Tolerance
//
// The system can tolerate up to (n-1)/3 Byzantine (malicious or faulty) nodes
// out of n total nodes. The quorum requirement ensures that honest nodes always
// form a majority for any decision.
package consensus