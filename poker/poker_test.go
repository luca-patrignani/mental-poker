package poker

import (
	"testing"
)

// TODO: make more elaborate test
func TestConvertCard(t *testing.T) {
	expectedCard := Card{suit: Heart, rank: 2}
	testCard, err := convertCard(28)
	if err != nil {
		t.Fatal(err)
	}
	if testCard != expectedCard {
		errString := "expected " + expectedCard.String() + ", get " + testCard.String()
		t.Fatal(errString)
	}

}
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