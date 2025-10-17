package poker

import (
	"errors"
	"fmt"
)

const (
	Club    = 0
	Diamond = 1
	Heart   = 2
	Spade   = 3
)

const (
	Jack  = 11
	Queen = 12
	King  = 13
	Ace   = 1
)

type Card struct {
	suit uint8
	rank uint8
}

// NewCard creates a new Card with the specified suit and rank. Returns an error if the suit
// is greater than 3 or if the rank is not between 1 and 13.
func NewCard(suit uint8, rank uint8) (Card, error) {
	if suit > 3 || rank == 0 || rank > 13 {
		return Card{}, fmt.Errorf("invalid card %d, %d", suit, rank)
	}

	return Card{
		suit: suit,
		rank: rank,
	}, nil
}

// ConvertCard converts a raw card number (1-52) to a Card. Card numbers map to suits in order
// (clubs, diamonds, hearts, spades) with ranks 1-13 within each suit. Returns an error
// if the card number is outside the valid range.
func ConvertCard(rawCard int) (Card, error) {
	if rawCard > 52 || rawCard < 1 {
		return Card{}, errors.New("the card to convert have an invalid value")
	}

	suit := uint8(((rawCard - 1) / 13))
	rank := uint8(((rawCard - 1) % 13) + 1)
	card, err := NewCard(suit, rank)
	if err != nil {
		return Card{}, err
	}
	return card, nil
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
		suit = "♣"
	case 1:
		suit = "♦"
	case 2:
		suit = "♥"
	case 3:
		suit = "♠"
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
	return rankStr + suit
}
