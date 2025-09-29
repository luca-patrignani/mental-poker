package poker

import (
	"testing"
)

// TestRecalculatePotsBasic checks a simple pot without any side pots
func TestRecalculatePotsBasic(t *testing.T) {
	session := Session{
		Players: []Player{
			{Name: "Alice", Id: 50, Bet: 50, HasFolded: false},
			{Name: "Bob", Id: 1, Bet: 50, HasFolded: false},
		},
	}

	session.RecalculatePots()

	if len(session.Pots) != 1 {
		t.Fatalf("expected 1 pot, got %d", len(session.Pots))
	}
	if session.Pots[0].Amount != 100 {
		t.Errorf("expected pot amount 100, got %d", session.Pots[0].Amount)
	}
	if len(session.Pots[0].Eligible) != 2 {
		t.Errorf("expected 2 eligible players, got %d", len(session.Pots[0].Eligible))
	}
}

// TestRecalculatePotsWithFold ensures folded players are excluded from future side pots
func TestRecalculatePotsWithFold(t *testing.T) {
	session := Session{
		Players: []Player{
			{Name: "Alice", Bet: 50, HasFolded: false},
			{Name: "Bob", Bet: 100, HasFolded: true},
			{Name: "Carol", Bet: 200, HasFolded: false},
		},
	}

	session.RecalculatePots()

	// Expected pots: 50*3=150, 50*2=100, 100*1=100
	expectedAmounts := []uint{150, 100, 100}
	expectedEligible := [][]int{
		{0, 2}, // first pot: Alice + Carol (Bob folded, but bet counts)
		{2},    // second pot: only Carol
		{2},    // third pot: only Carol
	}

	if len(session.Pots) != len(expectedAmounts) {
		t.Fatalf("expected %d pots, got %d", len(expectedAmounts), len(session.Pots))
	}

	for i, pot := range session.Pots {
		if pot.Amount != expectedAmounts[i] {
			t.Errorf("pot %d: expected amount %d, got %d", i, expectedAmounts[i], pot.Amount)
		}
		if len(pot.Eligible) != len(expectedEligible[i]) {
			t.Errorf("pot %d: expected %d eligible, got %d", i, len(expectedEligible[i]), len(pot.Eligible))
			continue
		}
		for j, idx := range pot.Eligible {
			if idx != expectedEligible[i][j] {
				t.Errorf("pot %d: expected eligible %d at pos %d, got %d", i, expectedEligible[i][j], j, idx)
			}
		}
	}
}

// TestRecalculatePotsAllIn verifies correct distribution when one or more players go all-in
func TestRecalculatePotsAllIn(t *testing.T) {
	session := Session{
		Players: []Player{
			{Name: "Alice", Bet: 50, Pot: 0, HasFolded: false},
			{Name: "Bob", Bet: 200, Pot: 0, HasFolded: false},
			{Name: "Carol", Bet: 100, Pot: 0, HasFolded: false},
		},
	}

	session.RecalculatePots()

	// Expected pots: 50*3=150, 50*2=100, 50*1=50
	expectedAmounts := []uint{150, 100, 100}
	expectedEligible := [][]int{
		{0, 1, 2},
		{1, 2},
		{1},
	}

	if len(session.Pots) != len(expectedAmounts) {
		t.Fatalf("expected %d pots, got %d", len(expectedAmounts), len(session.Pots))
	}
	for i, pot := range session.Pots {
		if pot.Amount != expectedAmounts[i] {
			t.Errorf("pot %d: expected amount %d, got %d", i, expectedAmounts[i], pot.Amount)
		}
		if len(pot.Eligible) != len(expectedEligible[i]) {
			t.Errorf("pot %d: expected %d eligible, got %d", i, len(expectedEligible[i]), len(pot.Eligible))
			continue
		}
		for j, idx := range pot.Eligible {
			if idx != expectedEligible[i][j] {
				t.Errorf("pot %d: expected eligible %d at pos %d, got %d", i, expectedEligible[i][j], j, idx)
			}
		}
	}
}

// TestRecalculatePotsAllFoldExceptOne checks that if everyone else folds, the remaining player gets the main pot
func TestRecalculatePotsAllFoldExceptOne(t *testing.T) {
	session := Session{
		Players: []Player{
			{Name: "Alice", Bet: 50, HasFolded: true},
			{Name: "Bob", Bet: 100, HasFolded: true},
			{Name: "Carol", Bet: 200, HasFolded: false},
		},
	}

	session.RecalculatePots()

	if len(session.Pots) != 1 {
		t.Fatalf("expected 1 pot, got %d", len(session.Pots))
	}
	if session.Pots[0].Amount != 350 { // all bets are in main pot
		t.Errorf("expected pot amount 350, got %d", session.Pots[0].Amount)
	}
	if len(session.Pots[0].Eligible) != 1 || session.Pots[0].Eligible[0] != 2 {
		t.Errorf("expected only Carol eligible, got %+v", session.Pots[0].Eligible)
	}
}

// TestRecalculatePotsEqualBetsNoSide checks that when bets are equal, only one pot is created
func TestRecalculatePotsEqualBetsNoSide(t *testing.T) {
	session := Session{
		Players: []Player{
			{Name: "Alice", Bet: 100, HasFolded: false},
			{Name: "Bob", Bet: 100, HasFolded: false},
			{Name: "Carol", Bet: 100, HasFolded: false},
		},
	}

	session.RecalculatePots()

	if len(session.Pots) != 1 {
		t.Fatalf("expected 1 pot, got %d", len(session.Pots))
	}
	if session.Pots[0].Amount != 300 {
		t.Errorf("expected pot amount 300, got %d", session.Pots[0].Amount)
	}
	if len(session.Pots[0].Eligible) != 3 {
		t.Errorf("expected 3 eligible players, got %d", len(session.Pots[0].Eligible))
	}
}
