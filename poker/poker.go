package poker

import (
	"fmt"
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
	type scored struct {
		p     Player
		score int16
	}

	var scoredPlayers []scored

	for _, player := range s.Players {
		// skip folded players
		if player.HasFolded {
			continue
		}
		// require both hole cards present (basic sanity check)
		if player.Hand[0].rank == 0 || player.Hand[1].rank == 0 {
			// skip players with invalid/missing hand
			continue
		}

		var finalHand [7]poker.Card
		// board 5 cards
		for i := 0; i < 5; i++ {
			c := s.Board[i]
			card, err := poker.MakeCard(poker.Suit(c.suit), poker.Rank(c.rank))
			if err != nil {
				return nil, fmt.Errorf("invalid board card at idx %d: %w", i, err)
			}
			finalHand[i] = card
		}
		// player's two hole cards
		c0, err := poker.MakeCard(poker.Suit(player.Hand[0].suit), poker.Rank(player.Hand[0].rank))
		if err != nil {
			return nil, fmt.Errorf("invalid player hole card: %w", err)
		}
		c1, err := poker.MakeCard(poker.Suit(player.Hand[1].suit), poker.Rank(player.Hand[1].rank))
		if err != nil {
			return nil, fmt.Errorf("invalid player hole card: %w", err)
		}
		finalHand[5] = c0
		finalHand[6] = c1

		score := poker.Eval7(&finalHand)
		scoredPlayers = append(scoredPlayers, scored{p: player, score: score})
	}

	if len(scoredPlayers) == 0 {
		return nil, fmt.Errorf("no active players to evaluate")
	}

	// sort by descending score
	sort.Slice(scoredPlayers, func(i, j int) bool {
		return scoredPlayers[i].score > scoredPlayers[j].score
	})

	// collect winners (tie-handling)
	winners := []Player{scoredPlayers[0].p}
	for i := 1; i < len(scoredPlayers); i++ {
		if scoredPlayers[i].score != scoredPlayers[0].score {
			break
		}
		winners = append(winners, scoredPlayers[i].p)
	}

	// stable ordering of winners (optional): by Rank (seat index)
	sort.Slice(winners, func(i, j int) bool {
		return winners[i].Rank < winners[j].Rank
	})

	return winners, nil
}
