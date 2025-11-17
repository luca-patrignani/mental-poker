// Package network provides peer-to-peer communication primitives for
// distributed consensus. It implements reliable broadcast and all-to-all
// communication patterns with synchronization barriers.
//
// # Core Components
//
// Peer: Low-level network node that handles HTTP-based communication
// between nodes in the distributed system.
//
// P2P: High-level adapter that implements the consensus.NetworkLayer interface.
//
// # Communication Patterns
//
// Broadcast: One node sends data to all other nodes (one-to-all).
// All nodes receive the same data.
//
// AllToAll: Each node sends data to all other nodes (all-to-all).
// Each node receives data from every other node.
//
// # Synchronization
//
// All communication methods include implicit barrier synchronization,
// ensuring that no peer can proceed until all peers have participated
// in the communication round. This is essential for consensus protocols.
//
// # Timeout Support
//
// Methods with "withTimeout" suffix support configurable timeouts with
// automatic retry logic, making the system more resilient to transient
// network issues.
package network