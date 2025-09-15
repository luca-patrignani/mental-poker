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

func NewCard(suit uint8, rank uint8) (Card, error) {
	if suit > 3 || rank == 0 || rank > 13 {
		return Card{}, fmt.Errorf("invalid card %d, %d", suit, rank)
	}

	return Card{
		suit: suit,
		rank: rank,
	}, nil
}

// Convert the raw input card with the following suit order: ♣clubs -> ♦diamonds -> ♥hearts -> ♠spades
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

func (c Card) Suit() uint8 {
	return c.suit
}

func (c Card) Rank() uint8 {
	return c.rank
}

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
