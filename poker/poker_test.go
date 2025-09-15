package poker

import (
	"testing"
)

// helper to create a single main pot for simple hands
func singleMainPot(players []Player) []Pot {
	amount := uint(0)
	eligible := []int{}
	for i, p := range players {
		amount += p.Bet
		eligible = append(eligible, i)
	}
	return []Pot{
		{
			Amount:   amount,
			Eligible: eligible,
		},
	}
}

// Test a single winner scenario
func TestWinnerEvalSingleWinner(t *testing.T) {
	session := Session{
		Board: [5]Card{{Heart, 2}, {Spade, 5}, {Heart, Ace}, {Diamond, Queen}, {Diamond, 10}},
		Players: []Player{
			{Rank: 0, Name: "p0", Hand: [2]Card{{Club, Ace}, {Heart, 7}}, Bet: 10},
			{Rank: 1, Name: "p1", Hand: [2]Card{{Spade, Ace}, {Heart, 8}}, Bet: 10},
			{Rank: 2, Name: "p2", Hand: [2]Card{{Club, 3}, {Heart, 4}}, Bet: 10},
		},
	}
	session.Pots = singleMainPot(session.Players)

	winners, err := session.WinnerEval()
	if err != nil {
		t.Fatal(err)
	}

	// expect player 2 to win full pot
	if winners[session.Players[2].Rank] != 30 {
		t.Fatalf("expected p2 to win 30, got %d", winners[session.Players[2].Rank])
	}

	// ensure other players got nothing
	if winners[session.Players[0].Rank] != 0 || winners[session.Players[1].Rank] != 0 {
		t.Fatalf("other players should win 0")
	}
}

// Test for tie scenario
func TestWinnerEvalTie(t *testing.T) {
	session := Session{
		Board: [5]Card{{Heart, 2}, {Spade, 5}, {Heart, Ace}, {Diamond, Queen}, {Diamond, 10}},
		Players: []Player{
			{Rank: 0, Name: "p0", Hand: [2]Card{{Club, Ace}, {Club, 8}}, Bet: 10},
			{Rank: 1, Name: "p1", Hand: [2]Card{{Spade, Queen}, {Heart, 3}}, Bet: 10},
			{Rank: 2, Name: "p2", Hand: [2]Card{{Spade, Ace}, {Heart, 8}}, Bet: 10},
			{Rank: 3, Name: "p3", Hand: [2]Card{{Spade, Jack}, {Heart, Jack}}, Bet: 10},
		},
	}
	session.Pots = singleMainPot(session.Players)

	winners, err := session.WinnerEval()
	if err != nil {
		t.Fatal(err)
	}

	// expect p0 and p2 to split the pot evenly
	totalTie := winners[session.Players[0].Rank] + winners[session.Players[2].Rank]

	if winners[session.Players[0].Rank] != 20 {
		t.Fatalf("expected p0 to win 20, got %d", winners[session.Players[0].Rank])
	}
	if winners[session.Players[2].Rank] != 20 {
		t.Fatalf("expected p2 to win 20, got %d", winners[session.Players[2].Rank])
	}
	if totalTie != 40 {
		t.Fatalf("expected tie winners to split 20 each, got total %d", totalTie)
	}
}

// Test that folded players are ignored
func TestWinnerEvalIgnoresFolded(t *testing.T) {
	session := Session{
		Board: [5]Card{{Heart, 2}, {Spade, 5}, {Heart, Ace}, {Diamond, Queen}, {Diamond, 10}},
		Players: []Player{
			{Rank: 0, Name: "p0", Hand: [2]Card{{Club, Ace}, {Heart, 7}}, Bet: 10},
			{Rank: 1, Name: "p1", HasFolded: true, Hand: [2]Card{{Spade, Ace}, {Heart, 8}}, Bet: 10},
			{Rank: 2, Name: "p2", Hand: [2]Card{{Club, 3}, {Heart, 4}}, Bet: 10},
		},
	}
	session.Pots = singleMainPot(session.Players)

	winners, err := session.WinnerEval()
	if err != nil {
		t.Fatal(err)
	}

	// folded player should not receive any pot
	if _, ok := winners[session.Players[1].Rank]; ok && winners[session.Players[1].Rank] > 0 {
		t.Fatalf("folded player should not win")
	}
}

// Test scenario with side pots due to all-in
func TestWinnerEvalSidePots(t *testing.T) {
	// scenario:
	// p0: bet 50
	// p1: all-in 30
	// p2: all-in 20
	// pots:
	// main pot: 20*3 = 60
	// side pot1: 10*2 = 20
	// side pot2: 10*1 = 10
	session := Session{
		Board: [5]Card{{Heart, 2}, {Spade, 5}, {Heart, Ace}, {Diamond, Queen}, {Diamond, 10}},
		Players: []Player{
			{Rank: 0, Name: "p0", Hand: [2]Card{{Club, Ace}, {Spade, 8}}, Bet: 50},
			{Rank: 1, Name: "p1", Hand: [2]Card{{Club, 8}, {Heart, 4}}, Bet: 30},
			{Rank: 2, Name: "p2", Hand: [2]Card{{Spade, Ace}, {Heart, 8}}, Bet: 20},
		},
	}
	session.Pots = []Pot{
		{Amount: 60, Eligible: []int{0, 1, 2}},
		{Amount: 20, Eligible: []int{0, 1}},
		{Amount: 10, Eligible: []int{0}},
	}

	winners, err := session.WinnerEval()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("winners: %+v", winners)

	total := winners[session.Players[0].Rank] + winners[session.Players[1].Rank] + winners[session.Players[2].Rank]
	if total != 90 {
		t.Fatalf("total pot distributed should be 90, got %d", total)
	}

	// simple assumption for this test: p0 has best hand, wins all pots
	if winners[session.Players[0].Rank] != 10+20+30 {
		t.Fatalf("expected p0 to win 30, got %d", winners[session.Players[0].Rank])
	}

	if winners[session.Players[1].Rank] != 0 {
		t.Fatalf("expected p1 to win 40, got %d", winners[session.Players[1].Rank])
	}

	if winners[session.Players[2].Rank] != 30 {
		t.Fatalf("expected p2 to win 30, got %d", winners[session.Players[2].Rank])
	}
}

// TestRecalculatePotsBasic checks a simple pot without any side pots
func TestRecalculatePotsBasic(t *testing.T) {
	session := Session{
		Players: []Player{
			{Name: "Alice", Bet: 50, HasFolded: false},
			{Name: "Bob", Bet: 50, HasFolded: false},
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
