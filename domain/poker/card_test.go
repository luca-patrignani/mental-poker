package poker

import (
	"fmt"
	"testing"

	"github.com/pterm/pterm"
)

func TestIntToCard(t *testing.T) {
	expectedCard := Card{suit: Heart, rank: 2}
	testCard, err := IntToCard(28)
	if err != nil {
		t.Fatal(err)
	}
	if testCard != expectedCard {
		t.Fatalf("expected %v, get %v", expectedCard, testCard)
	}
}

func TestAllIntToCard(t *testing.T) {
	for i := 1; i < 53; i++ {
		_, err := IntToCard(i)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCardToInt(t *testing.T) {
	for i := 1; i < 53; i++ {
		card, err := IntToCard(i)
		if err != nil {
			t.Fatal(err)
		}
		rawCard := CardToInt(card)
		if rawCard != i {
			t.Fatalf("expected %d, got %d", i, rawCard)
		}
	}
}

func TestNewCard_ValidCard(t *testing.T) {
	card, err := NewCard(Heart, Ace)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if card.Suit() != Heart || card.Rank() != Ace {
		t.Fatal("card has wrong suit or rank")
	}
}

func TestNewCard_InvalidSuit(t *testing.T) {
	_, err := NewCard(4, Ace)
	if err == nil {
		t.Fatal("expected error for invalid suit")
	}
}

func TestNewCard_InvalidRank_Zero(t *testing.T) {
	_, err := NewCard(Heart, 0)
	if err == nil {
		t.Fatal("expected error for rank 0")
	}
}

func TestNewCard_InvalidRank_TooHigh(t *testing.T) {
	_, err := NewCard(Heart, 14)
	if err == nil {
		t.Fatal("expected error for rank > 13")
	}
}

func TestConvertCard_BoundaryValues(t *testing.T) {
	// Test card 1 (Ace of Clubs)
	card, err := IntToCard(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card.Suit() != Club || card.Rank() != Ace {
		t.Fatalf("card 1 should be Ace of Clubs, got %s", card.String())
	}

	// Test card 52 (King of Spades)
	card, err = IntToCard(52)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card.Suit() != Spade || card.Rank() != King {
		t.Fatalf("card 52 should be King of Spades, got %s", card.String())
	}
}

func TestConvertCard_InvalidValues(t *testing.T) {
	tests := []int{0, 53, -1, 100}

	for _, val := range tests {
		_, err := IntToCard(val)
		if err == nil {
			t.Fatalf("expected error for card value %d", val)
		}
	}
}

func TestCardString_AllSuits(t *testing.T) {
	suits := []struct {
		suit     uint8
		expected string
	}{
		{Club, pterm.Black("♣")},
		{Diamond, pterm.LightRed("♦")},
		{Heart, pterm.LightRed("♥")},
		{Spade, pterm.Black("♠")},
	}

	for _, tc := range suits {
		card := Card{suit: tc.suit, rank: Ace}
		if card.String() != "A"+tc.expected {
			t.Fatalf("expected %s, got %s", "A"+tc.expected, card.String())
		}
	}
}

func TestCardString_NumberCards(t *testing.T) {
	for rank := uint8(2); rank <= 10; rank++ {
		card := Card{suit: Heart, rank: rank}
		expected := fmt.Sprintf("%d", rank)
		expected = expected + pterm.LightRed("♥")
		if card.String() != expected {
			t.Fatalf("expected %s, got %s", expected, card.String())
		}
	}
}
