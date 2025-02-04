package poker

import (
	"github.com/paulhankin/poker"
)

//TODO: Add a struct for holding hand and river card 

//TODO: Add interface for card, matching card struct of the package 

// The suit order is the following: hearts -> diamonds -> clubs -> spades
func convertCard() error {
	card, err := poker.MakeCard(poker.Spade, 12) 
	if poker.Rank(card) > 10 {
		return nil
	}
	return err
}

