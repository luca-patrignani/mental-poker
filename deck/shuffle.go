package deck

import (
	"math/rand"

	"go.dedis.ch/kyber/v3"
)

func (d *Deck) Shuffle() error {
	d.EncryptedDeck = make([]kyber.Point, d.DeckSize+1)
	for i, card := range d.CardCollection {
		d.EncryptedDeck[i] = card.Clone()
	}
	for j := 0; j < len(d.peer.Addresses); j++ {
		if j == d.peer.Rank {
			x := suite.Scalar().Pick(suite.RandomStream())
			perm := permutation(d.DeckSize)
			for i := 0; i <= d.DeckSize; i++ {
				d.EncryptedDeck[i] = suite.Point().Mul(x, d.EncryptedDeck[perm[i]]) //TODO: add temp variable for permutation
			}
		}
		var err error
		d.EncryptedDeck, err = d.broadcastMultiple(d.EncryptedDeck, j, d.DeckSize+1)
		if err != nil {
			return err
		}
		//TODO: prove that shuffle is good with protocol 4 (ZKA, so it's optional)
	}

	return nil
}

func permutation(permSize int) []int {
	perm := rand.Perm(permSize)
	for i := 0; i < permSize; i++ {
		perm[i]++
	}

	return append([]int{0}, perm...)
}
