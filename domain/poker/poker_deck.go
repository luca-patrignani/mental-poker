package poker

import (
	"errors"

	"github.com/luca-patrignani/mental-poker/domain/deck"
)

// PokerDeck wraps a generic mental poker deck and provides poker-specific
// card handling. It converts between poker Card representations and the
// underlying cryptographic deck implementation.
//
// The deck uses mental poker protocols to ensure:
//   - Fair shuffling without a trusted dealer
//   - Secret card distribution (only drawer sees card)
//   - Provable card reveals (all players verify)
type PokerDeck struct {
	*deck.Deck
}

// NewPokerDeck creates a new poker deck with 52 cards using the provided network layer.
// The deck uses mental poker protocols to ensure fairness in distributed card games.
//
// Parameters:
//   - peer: Network layer for P2P communication during shuffling and drawing
//
// Returns a PokerDeck ready for preparation and shuffling.
func NewPokerDeck(peer deck.NetworkLayer) PokerDeck {
	return PokerDeck{
		Deck: &deck.Deck{
			DeckSize: 52,
			Peer:     peer,
		},
	}
}

// IntToCard converts a raw card number (1-52) to a Card. Card numbers map to suits in order
// (clubs, diamonds, hearts, spades) with ranks 1-13 within each suit.
//
// Card numbering:
//   - 1-13: Clubs (Ace through King)
//   - 14-26: Diamonds (Ace through King)
//   - 27-39: Hearts (Ace through King)
//   - 40-52: Spades (Ace through King)
//
// Parameters:
//   - rawCard: Integer from 1 to 52 representing a card
//
// Returns the corresponding Card or an error if the number is outside valid range.
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

// CardToInt converts a Card to its integer representation (1-52).
// This is the inverse operation of IntToCard.
//
// Parameters:
//   - card: The poker card to convert
//
// Returns an integer from 1 to 52 representing the card.
func CardToInt(card Card) int {
	return int(card.Suit())*13 + int(card.Rank())
}

// DrawCard draws a card from the deck for the specified player using the mental poker protocol.
// The card is encrypted until all players cooperate to reveal it. Only the drawer receives
// the actual card value; other players receive a face-down placeholder.
//
// Parameters:
//   - drawer: Player ID who should receive the card
//
// Returns a pointer to the drawn Card, or an error if drawing fails.
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

// OpenCard reveals a previously drawn card to all players using the mental poker protocol.
// All players cooperate to decrypt the card, ensuring no single player can cheat.
//
// Parameters:
//   - player: Player ID whose card should be revealed
//   - card: Pointer to the encrypted card to reveal (can be nil)
//
// Returns the revealed Card or an error if the reveal protocol fails.
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
