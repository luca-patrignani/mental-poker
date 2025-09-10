package poker

import (
	"sort"

	"github.com/luca-patrignani/mental-poker/deck"
	"github.com/paulhankin/poker"
)

// Deck is the rappresentation of a game session.
type Session struct {
	Board   [5]Card
	Players []Player
	Deck    deck.Deck
	Pot uint
	HighestBet  uint
	Dealer uint
	CurrentTurn uint // index into Players for who must act
	RoundID string // identifier for the current betting round/hand
	LastIndex uint64 // last committed transaction/block index
}

// Evaluate the final hand and return the peer rank of the winner
func (s *Session) WinnerEval() ([]Player, error) {
	players := make([]struct {
		p     Player
		score int16
	}, len(s.Players))
	for i, player := range s.Players {
		var finalHand [7]poker.Card
		var err error
		for i := 0; i < 5; i++ {
			finalHand[i], err = poker.MakeCard(poker.Suit(s.Board[i].suit), poker.Rank(s.Board[i].rank))
			if err != nil {
				return nil, err
			}
		}
		finalHand[5], err = poker.MakeCard(poker.Suit(player.Hand[0].suit), poker.Rank(player.Hand[0].rank))
		if err != nil {
			return nil, err
		}
		finalHand[6], err = poker.MakeCard(poker.Suit(player.Hand[1].suit), poker.Rank(player.Hand[1].rank))
		if err != nil {
			return nil, err
		}
		players[i].p = player
		players[i].score = poker.Eval7(&finalHand)
	}

	sort.Slice(players, func(i, j int) bool {
		return players[i].score > players[j].score
	})
	// Check for ties
	winners := []Player{players[0].p}
	for i := 1; i < len(players); i++ {
		if players[i].score != players[0].score {
			break
		}
		winners = append(winners, players[i].p)
	}
	sort.Slice(winners, func(i, j int) bool {
		return winners[i].Rank < winners[j].Rank
	})
	return winners, nil
}
