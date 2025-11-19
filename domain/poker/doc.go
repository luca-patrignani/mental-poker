// Package poker implements the domain logic for Texas Hold'em poker games,
// including hand evaluation, pot management, player actions, and game flow.
// This package is game-rule-specific and decoupled from consensus concerns.
//
// # Game Rules
//
// This implementation follows standard Texas Hold'em rules:
//   - Each player receives 2 hole cards
//   - 5 community cards (flop: 3, turn: 1, river: 1)
//   - Betting rounds: PreFlop, Flop, Turn, River, Showdown
//   - Hand rankings: High card through Royal Flush
//   - Side pots for all-in scenarios
//
// # Core Types
//
// Session represents the complete state of a poker game including players,
// pots, community cards, current betting round, and turn tracking.
//
// Player represents a single player with their chips, hand, betting state,
// and fold status.
//
// Card represents a playing card with suit (clubs/diamonds/hearts/spades)
// and rank (ace through king).
//
// PokerAction represents a player's action (bet, raise, call, fold, check, etc.)
// with associated metadata like amount and round.
//
// # Game Flow
//
// 1. Initialization: Create session with players and initial chip stacks
// 2. Hand Start: Deal hole cards, post blinds, set dealer button
// 3. Betting Rounds: Players act in turn (PreFlop → Flop → Turn → River)
// 4. Showdown: Compare hands, determine winners, distribute pots
// 5. Hand End: Reset for next hand, advance dealer button
//
// # Pot Management
//
// The package handles complex pot scenarios:
//   - Main pot: Includes bets from all players up to minimum all-in
//   - Side pots: Created when players have different all-in amounts
//   - Eligibility: Only players who haven't folded can win each pot
//   - Split pots: Ties divide pot equally among winners
//
// # Hand Evaluation
//
// Uses 7-card poker hand evaluation (2 hole + 5 community) to determine winners.
// Handles standard rankings: High Card, Pair, Two Pair, Three of a Kind, Straight,
// Flush, Full House, Four of a Kind, Straight Flush, Royal Flush.
//
// # Example Usage
//
//	// Create players
//	players := []poker.Player{
//	    {Name: "Alice", Id: 0, Pot: 1000},
//	    {Name: "Bob", Id: 1, Pot: 1000},
//	}
//	
//	// Initialize session
//	session := poker.Session{
//	    Players:     players,
//	    Dealer:      0,
//	    CurrentTurn: 1,
//	    Round:       poker.PreFlop,
//	}
//	
//	// Create poker manager
//	manager := poker.PokerManager{Session: &session, Player: 0}
//	
//	// Validate and apply action
//	action := manager.ActionRaise(50)
//	if err := manager.Validate(action); err != nil {
//	    return err
//	}
//	if err := manager.Apply(action); err != nil {
//	    return err
//	}
package poker
