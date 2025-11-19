package poker

import (
	"fmt"

	"github.com/pterm/pterm"
)

// Card suit constants (0-3)
const (
	Club    = 0  // ♣ (black)
	Diamond = 1  // ♦ (red)
	Heart   = 2  // ♥ (red)
	Spade   = 3  // ♠ (black)
)

// Card rank constants for face cards and ace
const (
	Jack  = 11  // J
	Queen = 12  // Q
	King  = 13  // K
	Ace   = 1   // A (low in straights, high in value)
)

// FaceDown is the display character for hidden cards
const (
	FaceDown = "▓"
)

// Card represents a playing card with suit and rank.
// Rank 0 indicates a face-down or uninitialized card.
type Card struct {
	suit uint8  // 0-3: clubs, diamonds, hearts, spades
	rank uint8  // 1-13: ace through king (0 = face down)
}

// NewCard creates a new Card with validation.
//
// Parameters:
//   - suit: 0-3 (Club, Diamond, Heart, Spade)
//   - rank: 1-13 (Ace=1, 2-10=face value, Jack=11, Queen=12, King=13)
//
// Returns the Card or an error if suit or rank is invalid.
func NewCard(suit uint8, rank uint8) (Card, error) {
	if suit > 3 || rank == 0 || rank > 13 {
		return Card{}, fmt.Errorf("invalid card %d, %d", suit, rank)
	}

	return Card{
		suit: suit,
		rank: rank,
	}, nil
}

// Suit returns the suit value of the Card (0-3: clubs, diamonds, hearts, spades).
func (c Card) Suit() uint8 {
	return c.suit
}

// Rank returns the rank value of the Card (1-13: ace through king).
func (c Card) Rank() uint8 {
	return c.rank
}

// String returns a human-readable representation of the Card using suit symbols
// (♣, ♦, ♥, ♠) and rank abbreviations (A, J, Q, K, or number).
func (c Card) String() string {
	var suit string
	switch c.suit {
	case 0:
		suit = pterm.Black("♣")
	case 1:
		suit = pterm.LightRed("♦")
	case 2:
		suit = pterm.LightRed("♥")
	case 3:
		suit = pterm.Black("♠")
	default:
		suit = "?"
	}

	var rankStr string
	switch c.rank {
	case Ace:
		rankStr = "A"
	case Jack:
		rankStr = "J"
	case Queen:
		rankStr = "Q"
	case King:
		rankStr = "K"
	default:
		rankStr = fmt.Sprintf("%d", c.rank)
	}
	if c.rank == 0 {
		return FaceDown
	}
	return rankStr + suit
}
