package deck

import (
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/suites"
)

type Deck struct {
	DeckSize       int
	CardCollection []kyber.Point
}

var suite suites.Suite = suites.MustFind("Ed25519")

// Protocol 1: Deck Preparation
// Generate the deck as a set of encrypted values in a cyclic group
func (d *Deck) PrepareDeck() []kyber.Point {
	// Initialize deck
	deck := make([]kyber.Point, d.DeckSize)
	// Generate encrypted values for each card
	for i := 0; i < d.DeckSize; i++ {
		card := d.generateRandomElement() // Encrypt card as a^(i)
		deck[i] = card
	}

	return deck
}

// Protocol 2: Generate Random Element
// Generation of a random element in a distributed way to ensure secretness
func (d *Deck) generateRandomElement() kyber.Point {
	// initialize random generator of cyclic group G
	gj := suite.Point().Mul(suite.Scalar().Pick(suite.RandomStream()), nil)
	hj := suite.Point().Mul(suite.Scalar().Pick(suite.RandomStream()), nil)

	for gj.Equal(hj) {
		hj = suite.Point().Mul(suite.Scalar().Pick(suite.RandomStream()), nil)
	}

	lambda := suite.Scalar().Pick(suite.RandomStream()) // random lambda 0 < lambda < n

	gPrime := suite.Point().Mul(lambda, gj)
	_ = gPrime

	//TODO: broadcast gPrime All to ALL
	//gArray := BroadcastAlltoALl(g,gPrime,h)

	hPrime := suite.Point().Mul(lambda, hj)
	_ = hPrime

	//TODO: broadcast hPrime All to ALL
	var hArray []kyber.Point
	//hArray := BroadcastAlltoALl(hPrime)

	//TODO: ZKA (optional)

	hResult := hArray[0]
	for i := 1; i < len(hArray); i++ {
		hResult.Add(hResult, hArray[i])
	}

	return hResult
}
