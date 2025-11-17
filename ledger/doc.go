// Package ledger implements an immutable blockchain ledger for recording
// consensus decisions in the distributed poker game.
//
// # Core Components
//
// Blockchain: An append-only log of consensus decisions with cryptographic
// hash chaining for tamper detection.
//
// Block: A single consensus decision containing the game state, action,
// votes, and cryptographic links to previous blocks.
//
// # Security Properties
//
// The blockchain provides:
//   - Immutability: Once recorded, blocks cannot be modified
//   - Verifiability: Anyone can verify the integrity of the entire chain
//   - Auditability: Complete history of all game actions and votes
//   - Tamper detection: Any modification breaks the hash chain
//
// # Usage
//
// Create a blockchain with an initial game state, then append blocks as
// consensus decisions are reached. The Verify method can be called at any
// time to ensure the chain remains intact.
package ledger