package poker

import (
	"fmt"
	"sort"

	"github.com/luca-patrignani/mental-poker/deck"
	"github.com/paulhankin/poker"
)

// Deck is the rappresentation of a game session.
type Session struct {
	Board       [5]Card
	Players     []Player
	Deck        deck.Deck
	Pots        []Pot
	HighestBet  uint
	Dealer      uint
	CurrentTurn uint   // index into Players for who must act
	RoundID     string // identifier for the current betting round/hand
	LastIndex   uint64 // last committed transaction/block index
}

type Pot struct {
	Amount   uint
	Eligible []int // PlayerIDs che possono vincere questo piatto
}

// Evaluate the final hand and return the peer rank of the winner
func (s *Session) WinnerEval() (map[int]uint, error) {
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
			results[s.Players[w].Rank] += share
		}
	}

	return results, nil
}

func (s *Session) RecalculatePots() {
	s.Pots = nil

	// copy bets
	bets := make([]uint, len(s.Players))
	for i, p := range s.Players {
		bets[i] = p.Bet
	}

	for {
		// players with remaining bet
		contributors := []int{}
		for i, b := range bets {
			if b > 0 {
				contributors = append(contributors, i)
			}
		}
		if len(contributors) == 0 {
			break
		}

		// min bet among contributors
		minBet := bets[contributors[0]]
		for _, idx := range contributors {
			if bets[idx] < minBet {
				minBet = bets[idx]
			}
		}

		// compute pot amount
		potAmount := uint(0)
		for _, idx := range contributors {
			potAmount += min(bets[idx], minBet)
			bets[idx] -= minBet
		}

		// determine eligible players for this pot (must not be folded)
		eligible := []int{}
		for _, idx := range contributors {
			if !s.Players[idx].HasFolded {
				eligible = append(eligible, idx)
			}
		}

		s.Pots = append(s.Pots, Pot{
			Amount:   potAmount,
			Eligible: eligible,
		})
	}

	if onePlayerRemained(s.Pots) {
		totalPot := 0
		for _, p := range s.Pots {
			totalPot += int(p.Amount)
		}
		s.Pots = []Pot{{
			Amount:   uint(totalPot),
			Eligible: []int{s.Pots[0].Eligible[0]},
		}}
	}
}

func onePlayerRemained(lists []Pot) bool {
	for _, pot := range lists {
		if len(pot.Eligible) != 1 {
			return false
		}
	}
	return true
}
