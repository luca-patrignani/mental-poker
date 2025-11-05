package poker

import (
	"errors"

	"github.com/luca-patrignani/mental-poker/domain/deck"
)

type PokerDeck struct {
	*deck.Deck
}

func NewPokerDeck(peer deck.NetworkLayer) PokerDeck {
	return PokerDeck{
		Deck: &deck.Deck{
			DeckSize: 52,
			Peer:     peer,
		},
	}
}

// IntToCard converts a raw card number (1-52) to a Card. Card numbers map to suits in order
// (clubs, diamonds, hearts, spades) with ranks 1-13 within each suit. Returns an error
// if the card number is outside the valid range.
func IntToCard(rawCard int) (Card, error) {
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

func CardToInt(card Card) int {
	return int(card.Suit())*13 + int(card.Rank())
}

func (d PokerDeck) DrawCard(drawer int) (*Card, error) {
	c, err := d.Deck.DrawCard(drawer)
	if c == 0 {
		card, _ := NewCard(0, 0)
		return &card, nil
	}
	if err != nil {
		return nil, err
	}
	card, err := IntToCard(c)
	if err != nil {
		return nil, err
	}
	return &card, nil
}

func (d PokerDeck) OpenCard(player int, card *Card) (Card, error) {
	rawCard := 0
	if card != nil {
		rawCard = CardToInt(*card)
	}
	rawCard, err := d.Deck.OpenCard(player, rawCard)
	if err != nil {
		return Card{}, err
	}
	return IntToCard(rawCard)
}
