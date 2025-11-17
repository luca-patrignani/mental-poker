// Package poker implements the domain logic for Texas Hold'em poker games,
// including hand evaluation, pot management, player actions, and game flow.
//
// # Core Types
//
// Session: Represents the complete state of a poker game including players,
// pots, community cards, and current betting round.
//
// Player: Represents a single player with their chips, hand, and betting state.
//
// Card: Represents a playing card with suit and rank.
//
// PokerAction: Represents a player's action (bet, raise, call, fold, etc.).
//
// # Game Flow
//
// A poker game progresses through rounds: PreFlop → Flop → Turn → River → Showdown.
// Players take turns performing actions, and the game manages pot calculations,
// side pots for all-in scenarios, and winner determination.
//
// # Hand Evaluation
//
// The package uses 7-card poker hand evaluation to determine winners at showdown.
// It handles ties by splitting pots equally among winners.
package poker