package poker

import "testing"

func TestConvertCard(t *testing.T) {
	expectedCard := Card{suit: Heart, rank: 2}
	testCard, err := ConvertCard(28)
	if err != nil {
		t.Fatal(err)
	}
	if testCard != expectedCard {
		t.Fatalf("expected %v, get %v", expectedCard, testCard)
	}

}
func TestAllCardConvert(t *testing.T) {
	for i := 1; i < 53; i++ {
		_, err := ConvertCard(i)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCardStringFaces(t *testing.T) {
	c := Card{suit: Heart, rank: 1}
	if c.String() != "A♥" {
		t.Fatalf("expected A♥, got %s", c.String())
	}
	c = Card{suit: Club, rank: 11}
	if c.String() != "J♣" {
		t.Fatalf("expected J♣, got %s", c.String())
	}
}
