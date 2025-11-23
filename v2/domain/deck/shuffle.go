package deck

import (
	"math/rand"

	"go.dedis.ch/kyber/v4"
)

// Protocol 3: Shuffle Deck
// Each peer shuffles and re-encrypts the deck
func (d *Deck) Shuffle() error {
	d.lastDrawnCard = 0
	d.encryptedDeck = make([]kyber.Point, d.DeckSize+1)
	for i, card := range d.cardCollection {
		d.encryptedDeck[i] = card.Clone()
	}
	for j := 0; j < len(d.Peer.GetAddresses()); j++ {
		if j == d.Peer.GetRank() {
			x := suite.Scalar().Pick(suite.RandomStream())
			d.secretKey = x
			perm := permutation(d.DeckSize)
			tmp := make([]kyber.Point, d.DeckSize+1)
			for i, card := range d.encryptedDeck {
				tmp[i] = card.Clone()
			}
			for i := 0; i <= d.DeckSize; i++ {
				d.encryptedDeck[i].Mul(x, tmp[perm[i]])
			}
		}
		var err error
		d.encryptedDeck, err = d.broadcastMultiple(d.encryptedDeck, j, d.DeckSize+1)
		if err != nil {
			return err
		}
	}

	return nil
}

// Helper function to generate a random permutation of size permSize
func permutation(permSize int) []int {
	perm := rand.Perm(permSize)
	for i := 0; i < permSize; i++ {
		perm[i]++
	}

	return append([]int{0}, perm...)
}
