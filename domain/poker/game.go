package poker

import (
	"fmt"
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
	Round    Round      `json:"round_id"`
	PlayerID int        `json:"player_id"`
	Type     ActionType `json:"type"`
	Amount   uint       `json:"amount"`
}

type ActionType string

const (
	ActionBet      ActionType = "bet"
	ActionCall     ActionType = "call"
	ActionRaise    ActionType = "raise"
	ActionAllIn    ActionType = "allin"
	ActionFold     ActionType = "fold"
	ActionCheck    ActionType = "check"
	ActionReveal   ActionType = "reveal"
	ActionBan      ActionType = "ban"
	ActionShowdown ActionType = "showdown"
)

// Deck is the rappresentation of a game session.
type Session struct {
	Board       [5]Card
	Players     []Player
	Pots        []Pot
	HighestBet  uint
	LastToRaise uint  // index of the Player who last raised
	Dealer      uint  // index of the Player that is the dealer
	CurrentTurn uint  // index of the Player who must act
	Round       Round // identifier for the current betting round/hand
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

	if everybodyFolded(s.Players) {
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
func onePlayerRemained(players []Player) bool {
	pRemained := 0
	for _, p := range players {
		if !p.HasFolded && p.Pot > 0 {
			pRemained++
		}
	}
	return pRemained <= 1
}

func everybodyFolded(players []Player) bool {
	pRemained := 0
	for _, p := range players {
		if !p.HasFolded {
			pRemained++
		}
	}
	return pRemained <= 1
}

func (s *Session) OnePlayerRemained() bool {
	return onePlayerRemained(s.Players)
}

func (s *Session) EverybodyFolded() bool {
	return everybodyFolded(s.Players)
}

// ApplyAction applies a poker action to the session state and advances the turn to the next
// eligible player. Supports fold, bet, raise, call, all-in, check, and ban actions.
// Returns an error if the action type is unknown.
func applyAction(a ActionType, amount uint, session *Session, idx int) error {
	switch a {
	case ActionFold:
		session.Players[idx].HasFolded = true
		session.recalculatePots()
		if onePlayerRemained(session.Players) {
			session.Round = Showdown
			session.advanceTurn()
		} else {
			if session.isRoundFinished() {
				session.advanceRound()
			} else {
				session.advanceTurn()
			}
		}
	case ActionBet:
		session.Players[idx].Bet += amount
		session.Players[idx].Pot -= amount
		if session.Players[idx].Bet > session.HighestBet {
			session.HighestBet = session.Players[idx].Bet
			session.LastToRaise = uint(idx)
		}
		session.recalculatePots()
		if session.isRoundFinished() {
			session.advanceRound()
		} else {
			session.advanceTurn()
		}
	case ActionRaise:
		session.Players[idx].Bet += amount
		session.Players[idx].Pot -= amount
		session.HighestBet = session.Players[idx].Bet
		session.LastToRaise = uint(idx)
		session.recalculatePots()
		if onePlayerRemained(session.Players) {
			session.Round = Showdown
			session.advanceTurn()
		} else {
			if session.isRoundFinished() {
				session.advanceRound()
			} else {
				session.advanceTurn()
			}
		}
	case ActionCall:
		diff := session.HighestBet - session.Players[idx].Bet
		session.Players[idx].Bet += diff
		session.Players[idx].Pot -= diff
		session.recalculatePots()
		if onePlayerRemained(session.Players) {
			session.Round = Showdown
			session.advanceTurn()
		} else {
			if session.isRoundFinished() {
				session.advanceRound()
			} else {
				session.advanceTurn()
			}
		}
	case ActionAllIn:
		session.Players[idx].Bet += session.Players[idx].Pot
		session.Players[idx].Pot = 0
		if session.Players[idx].Bet > session.HighestBet {
			session.HighestBet = session.Players[idx].Bet
			session.LastToRaise = uint(idx)
		} else {
			if onePlayerRemained(session.Players) {
				session.Round = Showdown
				session.advanceTurn()
			}
		}
		session.recalculatePots()

		if session.isRoundFinished() {
			session.advanceRound()
		} else {
			session.advanceTurn()
		}

	case ActionCheck:
		if session.isRoundFinished() {
			session.advanceRound()
		} else {
			session.advanceTurn()
		}
	case ActionBan:
		session.Players = append(session.Players[:idx], session.Players[idx+1:]...)
		n := len(session.Players)
		session.Dealer %= uint(n)
		session.CurrentTurn = uint(session.getNextActivePlayer(session.Dealer))
		if session.isRoundFinished() {
			session.advanceRound()
		}
	case ActionShowdown:
		winners, err := session.winnerEval()
		if err != nil {
			return err
		}
		// distribute pots to winners
		for winnerId, amount := range winners {
			winnerIdx := session.FindPlayerIndex(winnerId)
			if winnerIdx == -1 {
				continue // should not happen
			}
			session.Players[winnerIdx].Pot += amount
		}
		session.advanceRound()

	default:
		return fmt.Errorf("unknown action")
	}
	return nil
}

// FindPlayerIndex returns the session index of the player with the given ID, or -1 if not found.
func (s *Session) FindPlayerIndex(playerID int) int {
	for i, p := range s.Players {
		if p.Id == playerID {
			return i
		}
	}
	return -1
}
