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

	session.recalculatePots()

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

	session.recalculatePots()

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

	session.recalculatePots()

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

	session.recalculatePots()

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

	session.recalculatePots()

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

func TestApplyAction_Fold(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Bet: 50, HasFolded: false},
			{Name: "Bob", Bet: 50, HasFolded: false},
			{Name: "John", Bet: 50, HasFolded: false},
		},
		CurrentTurn: 0,
	}

	err := applyAction(ActionFold, 0, session, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !session.Players[0].HasFolded {
		t.Fatal("player should be folded")
	}

	if session.CurrentTurn != 1 {
		t.Fatalf("turn should advance to 1, got %d", session.CurrentTurn)
	}
}

func TestApplyAction_OnePlayerRemained(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Bet: 50, HasFolded: false},
			{Name: "Bob", Bet: 50, HasFolded: true},
			{Name: "John", Bet: 50, HasFolded: false},
		},
		CurrentTurn: 0,
	}

	err := applyAction(ActionFold, 0, session, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !session.Players[0].HasFolded {
		t.Fatal("player should be folded")
	}

	if session.CurrentTurn != 2 {
		t.Fatalf("turn should advance to 1, got %d", session.CurrentTurn)
	}

	if session.Round != Showdown {
		t.Fatalf("round should be Showdown, got %s", session.Round)
	}

}

func TestApplyAction_Bet_UpdatesHighestBet(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 100, Bet: 0},
			{Name: "Bob", Pot: 100, Bet: 0},
		},
		CurrentTurn: 0,
		HighestBet:  0,
		Round:       "preflop",
	}

	err := applyAction(ActionBet, 50, session, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.Players[0].Bet != 50 {
		t.Fatalf("expected bet 50, got %d", session.Players[0].Bet)
	}

	if session.Players[0].Pot != 50 {
		t.Fatalf("expected pot 50, got %d", session.Players[0].Pot)
	}

	if session.HighestBet != 50 {
		t.Fatalf("expected highest bet 50, got %d", session.HighestBet)
	}
}

func TestApplyAction_Raise_UpdatesHighestBet(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 200, Bet: 50},
			{Name: "Bob", Pot: 100, Bet: 50},
		},
		CurrentTurn: 0,
		HighestBet:  50,
		Round:       PreFlop,
	}

	err := applyAction(ActionRaise, 50, session, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.Players[0].Bet != 100 {
		t.Fatalf("expected bet 100, got %d", session.Players[0].Bet)
	}

	if session.HighestBet != 100 {
		t.Fatalf("expected highest bet 100, got %d", session.HighestBet)
	}
}

func TestApplyAction_Call_MatchesHighestBet(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 100, Bet: 50},
			{Name: "Bob", Pot: 100, Bet: 100},
		},
		CurrentTurn: 0,
		HighestBet:  100,
		Round:       PreFlop,
	}

	err := applyAction(ActionCall, 0, session, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.Players[0].Bet != 100 {
		t.Fatalf("expected bet 100, got %d", session.Players[0].Bet)
	}

	if session.Players[0].Pot != 50 {
		t.Fatalf("expected pot 50, got %d", session.Players[0].Pot)
	}
}

func TestApplyAction_AllIn_EmptiesPot(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 75, Bet: 50},
			{Name: "Bob", Pot: 100, Bet: 100},
		},
		CurrentTurn: 0,
		HighestBet:  100,
		Round:       PreFlop,
	}

	err := applyAction(ActionAllIn, 0, session, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.Players[0].Pot != 0 {
		t.Fatalf("expected pot 0, got %d", session.Players[0].Pot)
	}

	if session.Players[0].Bet != 125 {
		t.Fatalf("expected bet 125, got %d", session.Players[0].Bet)
	}
}

func TestApplyAction_Check_NoStateChange(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 100, Bet: 50},
			{Name: "Bob", Pot: 100, Bet: 50},
		},
		CurrentTurn: 0,
		HighestBet:  50,
	}

	err := applyAction(ActionCheck, 0, session, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.Players[0].Bet != 50 {
		t.Fatal("bet should not change on check")
	}

	if session.CurrentTurn != 1 {
		t.Fatalf("turn should advance to 1, got %d", session.CurrentTurn)
	}
}

func TestApplyAction_Ban_RemovesPlayer(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Id: 1},
			{Name: "Bob", Id: 2},
			{Name: "Carol", Id: 3},
		},
		Dealer:      0,
		CurrentTurn: 1,
	}

	err := applyAction(ActionBan, 0, session, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(session.Players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(session.Players))
	}

	if session.Players[1].Name != "Carol" {
		t.Fatal("wrong player was removed")
	}
}

func TestAdvanceTurn_SkipsFoldedPlayers(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", HasFolded: false},
			{Name: "Bob", HasFolded: true},
			{Name: "Carol", HasFolded: false},
		},
		CurrentTurn: 0,
	}

	session.advanceTurn()

	if session.CurrentTurn != 2 {
		t.Fatalf("expected turn 2, got %d", session.CurrentTurn)
	}
}

func TestAdvanceTurn_WrapsAround(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", HasFolded: false},
			{Name: "Bob", HasFolded: false},
		},
		CurrentTurn: 1,
	}

	session.advanceTurn()

	if session.CurrentTurn != 0 {
		t.Fatalf("expected turn to wrap to 0, got %d", session.CurrentTurn)
	}
}
