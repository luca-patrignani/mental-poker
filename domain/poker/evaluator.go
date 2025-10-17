package poker

import (
	"fmt"
	"sort"

	"github.com/paulhankin/poker"
)

// WinnerEval evaluates the final hand for all players and distributes the pot among winners.
// It uses 7-card hand evaluation for each eligible player in each pot, handles ties by splitting
// the pot equally, and excludes folded players. Returns a map of player ID to winnings.
func (s *Session) winnerEval() (map[int]uint, error) {
	results := make(map[int]uint)

	for _, pot := range s.Pots {
		type scored struct {
			idx   int
			score int16
		}
		var scoredPlayers []scored

		for _, idx := range pot.Eligible {
			player := s.Players[idx]
			if player.HasFolded {
				continue
			}

			// sanity check
			if player.Hand[0].rank == 0 || player.Hand[1].rank == 0 {
				continue
			}

			var finalHand [7]poker.Card
			for i := 0; i < 5; i++ {
				c := s.Board[i]
				card, err := poker.MakeCard(poker.Suit(c.suit), poker.Rank(c.rank))
				if err != nil {
					return nil, fmt.Errorf("invalid board card at idx %d: %w", i, err)
				}
				finalHand[i] = card
			}

			c0, err := poker.MakeCard(poker.Suit(player.Hand[0].suit), poker.Rank(player.Hand[0].rank))
			if err != nil {
				return nil, fmt.Errorf("invalid player card: %w", err)
			}
			c1, err := poker.MakeCard(poker.Suit(player.Hand[1].suit), poker.Rank(player.Hand[1].rank))
			if err != nil {
				return nil, fmt.Errorf("invalid player card: %w", err)
			}
			finalHand[5] = c0
			finalHand[6] = c1

			score := poker.Eval7(&finalHand)
			scoredPlayers = append(scoredPlayers, scored{idx: idx, score: score})
		}

		if len(scoredPlayers) == 0 {
			continue // no eligible players
		}

		// sort by score descending
		sort.Slice(scoredPlayers, func(i, j int) bool {
			return scoredPlayers[i].score > scoredPlayers[j].score
		})

		bestScore := scoredPlayers[0].score
		winners := []int{scoredPlayers[0].idx}
		for i := 1; i < len(scoredPlayers); i++ {
			if scoredPlayers[i].score == bestScore {
				winners = append(winners, scoredPlayers[i].idx)
			} else {
				break
			}
		}

		share := pot.Amount / uint(len(winners))
		for _, w := range winners {
			results[s.Players[w].Id] += share
		}
	}

	return results, nil
}
