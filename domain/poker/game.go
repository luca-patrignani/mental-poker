package poker

import (
	"fmt"

	"github.com/luca-patrignani/mental-poker/domain/deck"
)

type Player struct {
	Name      string
	Id        int //rank
	Hand      [2]Card
	HasFolded bool
	Bet       uint // The amount of money bet in the current betting round
	Pot       uint
}

// PokerAction è l'azione specifica del dominio poker
type PokerAction struct {
	RoundID  string     `json:"round_id"`
	PlayerID int        `json:"player_id"`
	Type     ActionType `json:"type"`
	Amount   uint       `json:"amount"`
}

type ActionType string

const (
	ActionBet    ActionType = "bet"
	ActionCall   ActionType = "call"
	ActionRaise  ActionType = "raise"
	ActionAllIn  ActionType = "allin"
	ActionFold   ActionType = "fold"
	ActionCheck  ActionType = "check"
	ActionReveal ActionType = "reveal"
	ActionBan    ActionType = "ban"
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
}

type Pot struct {
	Amount   uint
	Eligible []int // PlayerIDs che possono vincere questo piatto
}

// RecalculatePots recomputes the pot structure based on current player bets. It creates main
// pots for minimum bets and side pots for additional bets, determining eligibility based on
// non-folded players. Handles all-in scenarios where different players contribute different amounts.
func (s *Session) recalculatePots() {
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

// onePlayerRemained checks if all pots have exactly one eligible player, which consolidates
// multiple pots into a single pot for that player.
func onePlayerRemained(lists []Pot) bool {
	for _, pot := range lists {
		if len(pot.Eligible) != 1 {
			return false
		}
	}
	return true
}

// ApplyAction applies a poker action to the session state and advances the turn to the next
// eligible player. Supports fold, bet, raise, call, all-in, check, and ban actions.
// Returns an error if the action type is unknown.
func applyAction(a ActionType, amount uint, session *Session, idx int) error {
	switch a {
	case ActionFold:
		session.Players[idx].HasFolded = true
		session.recalculatePots()
		session.advanceTurn()
	case ActionBet:
		session.Players[idx].Bet += amount
		session.Players[idx].Pot -= amount
		if session.Players[idx].Bet > session.HighestBet {
			session.HighestBet = session.Players[idx].Bet
		}
		session.recalculatePots()
		session.advanceTurn()
	case ActionRaise:
		session.Players[idx].Bet += amount
		session.Players[idx].Pot -= amount
		session.HighestBet = session.Players[idx].Bet
		session.recalculatePots()
		session.advanceTurn()
	case ActionCall:
		diff := session.HighestBet - session.Players[idx].Bet
		session.Players[idx].Bet += diff
		session.Players[idx].Pot -= diff
		session.recalculatePots()
		session.advanceTurn()
	case ActionAllIn:
		session.Players[idx].Bet += session.Players[idx].Pot
		session.Players[idx].Pot = 0
		if session.Players[idx].Bet >= session.HighestBet {
			session.HighestBet = session.Players[idx].Bet
		}
		session.recalculatePots()
		session.advanceTurn()

	case ActionCheck:
		session.advanceTurn()
	case ActionBan:
		session.Players = append(session.Players[:idx], session.Players[idx+1:]...)
		n := len(session.Players)
		session.Dealer %= uint(n)
		session.CurrentTurn = (session.Dealer + 1) % uint(n)

	default:
		return fmt.Errorf("unknown action")
	}
	return nil
}

// advanceTurn moves the current turn to the next non-folded player in the session.
// It wraps around from the last player to the first and handles the case where all
// other players have folded (no advancement occurs).
func (session *Session) advanceTurn() {
	n := len(session.Players)
	if n == 0 {
		return
	}
	for i := 1; i <= n; i++ {
		next := (int(session.CurrentTurn) + i) % n
		if !session.Players[next].HasFolded {
			session.CurrentTurn = uint(next)
			return
		}
	}
}
