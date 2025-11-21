package poker

type Round string

const (
	PreFlop  Round = "preflop"
	Flop     Round = "flop"
	Turn     Round = "Turn"
	River    Round = "River"
	Showdown Round = "Showdown"
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
	session.Round = nextRound(session.Round)
	if session.Round == PreFlop {
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
func nextRound(current Round) Round {
	c := current
	rounds := []Round{PreFlop, Flop, Turn, River, Showdown}

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

// Helper: gets the next active (non-folded) player index after the given index
func (session *Session) getNextActivePlayer(currentIdx uint) int {
	n := len(session.Players)

	for i := 1; i <= n; i++ {
		next := (int(currentIdx) + i) % n
		if !session.Players[next].HasFolded && session.Players[next].Pot > 0 {
			return next
		}
	}

	// No active players found (shouldn't happen in valid game state)
	return -1
}

func (s *Session) setNextMatchDealer() {
	l := uint(len(s.Players))
	s.Dealer = (s.Dealer + 1) % l
	s.CurrentTurn = (s.Dealer + 1) % l
}
