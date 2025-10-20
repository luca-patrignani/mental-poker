package poker

import (
	"fmt"
	"strings"
	"time"
)

const (
	PreFlop  string = "preflop"
	Flop     string = "flop"
	Turn     string = "Turn"
	River    string = "River"
	Showdown string = "Showdown"
)

// advanceTurn moves the current turn to the next non-folded player in the session.
// It wraps around from the last player to the first and handles the case where all
// other players have folded (no advancement occurs).
func (session *Session) advanceTurn() {
	n := len(session.Players)
	if n == 0 {
		return
	}
	next := session.getNextActivePlayer(session.CurrentTurn)
	if next == -1 {
		// All players folded, no advancement
		return
	}
	session.CurrentTurn = uint(next)
}

// Verifies if the current betting round is finished.
func (session *Session) isRoundFinished() bool {
	if uint(session.getNextActivePlayer(session.CurrentTurn)) == session.LastToRaise {
		for _, player := range session.Players {
			if !player.HasFolded && player.Bet != session.HighestBet && player.Pot > player.Bet {
				return false
			}
		}
		return true
	}
	return false
}

func (session *Session) advanceRound() {
	idx := session.getNextActivePlayer(session.Dealer)
	session.CurrentTurn = uint(idx)
	session.LastToRaise = uint(idx)
	session.RoundID = makeRoundID(nextRound(session.RoundID))
	if extractRoundName(session.RoundID) == PreFlop {
		session.Dealer = uint(session.Dealer + 1%uint(len(session.Players)))
		session.CurrentTurn = uint(session.Dealer + 1%uint(len(session.Players)))
		session.LastToRaise = session.CurrentTurn
		session.HighestBet = 0
		for i := range session.Players {
			session.Players[i].Bet = 0
		}
		session.Pots = []Pot{}
	}
}

// Returns the next round name. If at last round, stays at last.
func nextRound(current string) string {
	c := extractRoundName(current)
	rounds := []string{PreFlop, Flop, Turn, River, Showdown}

	for i, r := range rounds {
		if r == c {
			if i < len(rounds)-1 {
				return rounds[i+1]
			}
			return rounds[0]
		}
	}
	// Not found, default to first
	return PreFlop
}

// Combine round name and Unix timestamp
func makeRoundID(round string) string {
	return fmt.Sprintf("%s-%d", round, time.Now().Unix())
}

// Extract round name from combined ID
func extractRoundName(roundID string) string {
	parts := strings.SplitN(roundID, "-", 2)
	return parts[0]
}

// Helper: gets the next active (non-folded) player index after the given index
func (session *Session) getNextActivePlayer(currentIdx uint) int {
	n := len(session.Players)

	for i := 1; i <= n; i++ {
		next := (int(currentIdx) + i) % n
		if !session.Players[next].HasFolded {
			return next
		}
	}

	// No active players found (shouldn't happen in valid game state)
	return -1
}
