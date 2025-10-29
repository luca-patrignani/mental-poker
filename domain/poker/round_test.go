package poker

import (
	"testing"
)

// TestNextRound verifies round progression logic
func TestNextRound(t *testing.T) {
	tests := []struct {
		name     string
		current  Round
		expected Round
	}{
		{
			name:     "PreFlop to Flop",
			current:  PreFlop,
			expected: Flop,
		},
		{
			name:     "Flop to Turn",
			current:  Flop,
			expected: Turn,
		},
		{
			name:     "Turn to River",
			current:  Turn,
			expected: River,
		},
		{
			name:     "River to Showdown",
			current:  River,
			expected: Showdown,
		},
		{
			name:     "Showdown reset to PreFlop",
			current:  Showdown,
			expected: PreFlop,
		},
		{
			name:     "Unknown round defaults to PreFlop",
			current:  "unknown",
			expected: PreFlop,
		},
		{
			name:     "Empty string defaults to PreFlop",
			current:  "",
			expected: PreFlop,
		},
		{
			name:     "Round (PreFlop)",
			current:  PreFlop,
			expected: Flop,
		},
		{
			name:     "Round (Turn)",
			current:  Turn,
			expected: River,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nextRound(tt.current)
			if result != tt.expected {
				t.Errorf("nextRound(%q) = %q, want %q", tt.current, result, tt.expected)
			}
		})
	}
}

// TestIsRoundFinished verifies betting round completion logic
func TestIsRoundFinished(t *testing.T) {
	tests := []struct {
		name        string
		session     Session
		expected    bool
		description string
	}{
		{
			name: "Round not finished - not back to last raiser",
			session: Session{
				Players: []Player{
					{Id: 0, Bet: 100, HasFolded: false, Pot: 1000},
					{Id: 1, Bet: 100, HasFolded: false, Pot: 1000},
					{Id: 2, Bet: 100, HasFolded: false, Pot: 1000},
				},
				HighestBet:  100,
				LastToRaise: 2,
				CurrentTurn: 0,
			},
			expected:    false,
			description: "CurrentTurn (0) + 1 != LastToRaise (2)",
		},
		{
			name: "Round finished - all bets equal and back to raiser",
			session: Session{
				Players: []Player{
					{Id: 0, Bet: 100, HasFolded: false, Pot: 1000},
					{Id: 1, Bet: 100, HasFolded: false, Pot: 1000},
					{Id: 2, Bet: 100, HasFolded: false, Pot: 1000},
				},
				HighestBet:  100,
				LastToRaise: 1,
				CurrentTurn: 0,
			},
			expected:    true,
			description: "CurrentTurn + 1 == LastToRaise and all bets equal",
		},
		{
			name: "Round not finished - player hasn't matched bet",
			session: Session{
				Players: []Player{
					{Id: 0, Bet: 50, HasFolded: false, Pot: 1000},
					{Id: 1, Bet: 100, HasFolded: false, Pot: 1000},
					{Id: 2, Bet: 100, HasFolded: false, Pot: 1000},
				},
				HighestBet:  100,
				LastToRaise: 1,
				CurrentTurn: 0,
			},
			expected:    false,
			description: "Player 0 has not matched highest bet",
		},
		{
			name: "Round finished - folded players ignored",
			session: Session{
				Players: []Player{
					{Id: 0, Bet: 100, HasFolded: false, Pot: 1000},
					{Id: 1, Bet: 50, HasFolded: true, Pot: 1000},
					{Id: 2, Bet: 100, HasFolded: false, Pot: 1000},
				},
				HighestBet:  100,
				LastToRaise: 2,
				CurrentTurn: 0,
			},
			expected:    true,
			description: "Folded player's bet doesn't matter",
		},
		{
			name: "Round finished - all-in player with less than highest bet",
			session: Session{
				Players: []Player{
					{Id: 0, Bet: 50, HasFolded: false, Pot: 50}, // all-in
					{Id: 1, Bet: 100, HasFolded: false, Pot: 1000},
					{Id: 2, Bet: 100, HasFolded: false, Pot: 1000},
				},
				HighestBet:  100,
				LastToRaise: 1,
				CurrentTurn: 0,
			},
			expected:    true,
			description: "Player 0 is all-in (Pot <= Bet), so round can finish",
		},
		{
			name: "All players check (no bets)",
			session: Session{
				Players: []Player{
					{Id: 0, Bet: 0, HasFolded: false, Pot: 1000},
					{Id: 1, Bet: 0, HasFolded: false, Pot: 1000},
					{Id: 2, Bet: 0, HasFolded: false, Pot: 1000},
				},
				HighestBet:  0,
				LastToRaise: 1,
				CurrentTurn: 0,
			},
			expected:    true,
			description: "All players checked, bets are equal (0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.isRoundFinished()
			if result != tt.expected {
				t.Errorf("%s: isRoundFinished() = %v, want %v", tt.description, result, tt.expected)
			}
		})
	}
}

// TestAdvanceRound verifies turn advancement and round progression
func TestAdvanceRound(t *testing.T) {
	tests := []struct {
		name          string
		session       Session
		expectedTurn  uint
		expectedRound Round
		description   string
	}{
		{
			name: "Advance to next active player",
			session: Session{
				Players: []Player{
					{Id: 0, HasFolded: false},
					{Id: 1, HasFolded: false},
					{Id: 2, HasFolded: false},
				},
				Dealer:      0,
				CurrentTurn: 0,
				Round:     PreFlop,
			},
			expectedTurn:  1,
			expectedRound: Flop, // Round doesn't change in advanceRound, just turn
			description:   "Should advance to player 1",
		},
		{
			name: "Skip folded players",
			session: Session{
				Players: []Player{
					{Id: 0, HasFolded: false},
					{Id: 1, HasFolded: true},
					{Id: 2, HasFolded: false},
				},
				Dealer:      0,
				CurrentTurn: 0,
				Round:     Flop,
			},
			expectedTurn:  2,
			expectedRound: Turn,
			description:   "Should skip folded player 1 and go to player 2",
		},
		{
			name: "Wrap around from last to first player",
			session: Session{
				Players: []Player{
					{Id: 0, HasFolded: false},
					{Id: 1, HasFolded: false},
					{Id: 2, HasFolded: false},
				},
				Dealer:      2,
				CurrentTurn: 2,
				Round:     Turn,
			},
			expectedTurn:  0,
			expectedRound: River,
			description:   "Should wrap around to player 0",
		},
		{
			name: "Only one player left (all others folded)",
			session: Session{
				Players: []Player{
					{Id: 0, HasFolded: true},
					{Id: 1, HasFolded: false},
					{Id: 2, HasFolded: true},
				},
				Dealer:      0,
				CurrentTurn: 0,
				Round:     River,
			},
			expectedTurn:  1,
			expectedRound: Showdown,
			description:   "Should find the only active player",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.session.advanceRound()

			if tt.session.CurrentTurn != tt.expectedTurn {
				t.Errorf("%s: CurrentTurn = %d, want %d", tt.description, tt.session.CurrentTurn, tt.expectedTurn)
			}

			// Verify round name part (ignore timestamp)
			actualRound := tt.session.Round
			if actualRound != tt.expectedRound {
				t.Errorf("%s: Round = %s, want %s", tt.description, actualRound, tt.expectedRound)
			}
		})
	}
}

// TestRoundProgression verifies full round cycle
func TestRoundProgression(t *testing.T) {
	rounds := []Round{PreFlop, Flop, Turn, River, Showdown}

	currentRoundID := PreFlop

	for i := 0; i < len(rounds)-1; i++ {
		extracted := currentRoundID
		if extracted != rounds[i] {
			t.Errorf("Step %d: expected round %s, got %s", i, rounds[i], extracted)
		}

		nextRoundName := nextRound(currentRoundID)
		currentRoundID = nextRoundName
	}

	// Verify we're at Showdown
	finalRound := currentRoundID
	if finalRound != Showdown {
		t.Errorf("Final round should be Showdown, got %s", finalRound)
	}

	// Verify Showdown stays at Showdown
	nextAfterShowdown := nextRound(currentRoundID)
	if nextAfterShowdown != PreFlop {
		t.Errorf("After Showdown, should reset to Preflop, got %s", nextAfterShowdown)
	}
}

// TestAllPlayersCheckScenario verifies round progression when all players check
func TestAllPlayersCheckScenario(t *testing.T) {
	session := Session{
		Players: []Player{
			{Id: 0, Bet: 10, HasFolded: false, Pot: 1000},
			{Id: 1, Bet: 10, HasFolded: false, Pot: 1000},
			{Id: 2, Bet: 10, HasFolded: false, Pot: 1000},
		},
		HighestBet:  10,
		LastToRaise: 1,
		CurrentTurn: 0,
		Dealer:      2,
		Round:     PreFlop,
	}

	// Verify round can finish when all check
	if !session.isRoundFinished() {
		t.Error("Round should be finished when all players check (bet = 0)")
	}

	// Advance to next round
	oldRound := session.Round
	session.advanceRound()

	// Verify turn advanced
	if session.CurrentTurn != 0 {
		t.Error("CurrentTurn should have been the first player after dealer")
	}

	// Verify round progressed
	newRound := session.Round
	expectedNext := nextRound(oldRound)
	if newRound != expectedNext {
		t.Errorf("Round should progress from %s to %s, got %s", oldRound, expectedNext, newRound)
	}
}
