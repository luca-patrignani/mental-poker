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
	Pots    []Pot
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


// GamePhase represents the current state of the poker game
type GamePhase string

const (
	StateWaitingForPlayers GamePhase = "waiting_for_players"
	StatePreFlop           GamePhase = "pre_flop"
	StateFlop              GamePhase = "flop"
	StateTurn              GamePhase = "turn"
	StateRiver             GamePhase = "river"
	StateShowdown          GamePhase = "showdown"
	StateHandComplete      GamePhase = "hand_complete"
)

// BettingState tracks the betting within a round
type BettingState string

const (
	BettingNotStarted BettingState = "not_started"
	BettingInProgress BettingState = "in_progress"
	BettingComplete   BettingState = "complete"
)
