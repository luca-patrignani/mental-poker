package poker

import (
	//"github.com/luca-patrignani/mental-poker/deck"
	"github.com/paulhankin/poker"
)

// TODO: Add a struct for holding hand and river card
// Deck is the rappresentation of a game session.
type PokerHand struct {
	Board [5]poker.Card
	Hand  [2]poker.Card
}

//TODO: Add interface for card matching card struct of the package

// Convert the raw input card with the following suit order: ♣ -> ♦ -> ♥ -> ♠
func convertCard(rawCard int) (poker.Card, error) {
	suit := poker.Suit(uint8((rawCard / 13) + 1))
	rank := poker.Rank((rawCard - 1) % 13)
	card, err := poker.MakeCard(suit, rank)
	if err != nil {
		return 0, err
	}
	return card, nil
}

//Evaluate the final hand and return the peer rank of the winner
func (hb *PokerHand) FinalHandEvaluation()([]int,error) {
	finalHand := append(hb.Hand,hb.Board...)
	score := poker.Eval7(finalHand)
	//TODO: broadcast all to all to determine best
}
