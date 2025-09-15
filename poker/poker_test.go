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
