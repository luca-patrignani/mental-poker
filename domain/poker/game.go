package poker

import (
	"fmt"
)

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

func ApplyAction(a ActionType, amount uint, session *Session, idx int) error {
	switch a {
	case ActionFold:
		session.Players[idx].HasFolded = true
		session.RecalculatePots()
		session.advanceTurn()
	case ActionBet:
		session.Players[idx].Bet += amount
		session.Players[idx].Pot -= amount
		if session.Players[idx].Bet > session.HighestBet {
			session.HighestBet = session.Players[idx].Bet
		}
		session.RecalculatePots()
		session.advanceTurn()
	case ActionRaise:
		session.Players[idx].Bet += amount
		session.Players[idx].Pot -= amount
		session.HighestBet = session.Players[idx].Bet
		session.RecalculatePots()
		session.advanceTurn()
	case ActionCall:
		diff := session.HighestBet - session.Players[idx].Bet
		session.Players[idx].Bet += diff
		session.Players[idx].Pot -= diff
		session.RecalculatePots()
		session.advanceTurn()
	case ActionAllIn:
		session.Players[idx].Bet += session.Players[idx].Pot
		session.Players[idx].Pot = 0
		if session.Players[idx].Bet >= session.HighestBet {
			session.HighestBet = session.Players[idx].Bet
		}
		session.RecalculatePots()
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
