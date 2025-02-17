package poker

import "testing"

func TestConvertCard(t *testing.T) {
	expectedCard := Card{suit: Heart, rank: 2}
	testCard, err := convertCard(28)
	if err != nil {
		t.Fatal(err)
	}
	if testCard != expectedCard {
		t.Fatalf("expected %v, get %v", expectedCard, testCard)
	}

}
func TestAllCardConvert(t *testing.T) {
	for i:= 1; i < 53; i++ {
		_, err := convertCard(i)
		if err != nil {
			t.Fatal(err)
		}
	}
}