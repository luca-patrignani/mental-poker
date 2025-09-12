package poker

import (
	"testing"
)

func TestWinnerEvalSingleWinner(t *testing.T) {
	session := Session{
		Board: [5]Card{{Heart, 2}, {Spade, 5}, {Heart, Ace}, {Diamond, Queen}, {Diamond, 10}},
		Players: []Player{
			{Rank: 0, Hand: [2]Card{{Club, Ace}, {Heart, 7}}},
			{Rank: 1, Hand: [2]Card{{Spade, Ace}, {Heart, 8}}},
			{Rank: 2, Hand: [2]Card{{Club, 3}, {Heart, 4}}},
		},
	}
	winners, err := session.WinnerEval()
	if err != nil {
		t.Fatal(err)
	}
	if len(winners) != 1 {
		t.Fatalf("expected length 1, actual %d", len(winners))
	}
	if winners[0].Rank != 2 {
		t.Fatalf("expected player 2, actual %d", winners[0].Rank)
	}
}

func TestWinnerEvalTie(t *testing.T) {
	session := Session{
		Board: [5]Card{{Heart, 2}, {Spade, 5}, {Heart, Ace}, {Diamond, Queen}, {Diamond, 10}},
		Players: []Player{
			{Rank: 0, Hand: [2]Card{{Club, Ace}, {Club, 8}}},
			{Rank: 1, Hand: [2]Card{{Spade, Queen}, {Heart, 3}}},
			{Rank: 2, Hand: [2]Card{{Spade, Ace}, {Heart, 8}}},
			{Rank: 3, Hand: [2]Card{{Spade, Jack}, {Heart, Jack}}},
		},
	}
	winners, err := session.WinnerEval()
	if err != nil {
		t.Fatal(err)
	}
	if len(winners) != 2 {
		t.Fatalf("expected length 2, actual %d", len(winners))
	}
	if winners[0].Rank != 0 {
		t.Fatalf("expected player 0, actual %d", winners[0].Rank)
	}
	if winners[1].Rank != 2 {
		t.Fatalf("expected player 2, actual %d", winners[1].Rank)
	}
}

func TestWinnerEvalIgnoresFolded(t *testing.T) {
	session := Session{
		Board: [5]Card{{Heart, 2}, {Spade, 5}, {Heart, Ace}, {Diamond, Queen}, {Diamond, 10}},
		Players: []Player{
			{Rank: 0, Hand: [2]Card{{Club, Ace}, {Heart, 7}}},
			{Rank: 1, HasFolded: true, Hand: [2]Card{{Spade, Ace}, {Heart, 8}}},
			{Rank: 2, Hand: [2]Card{{Club, 3}, {Heart, 4}}},
		},
	}
	winners, err := session.WinnerEval()
	if err != nil {
		t.Fatal(err)
	}
	// player 1 folded and must be ignored
	for _, w := range winners {
		if w.Rank == 1 {
			t.Fatalf("folded player should not be winner")
		}
	}
}
