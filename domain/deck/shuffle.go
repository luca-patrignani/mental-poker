package deck

import (
	"math/rand"

	"go.dedis.ch/kyber/v4"
)

func (d *Deck) Shuffle() error {
	d.encryptedDeck = make([]kyber.Point, d.DeckSize+1)
	for i, card := range d.cardCollection {
		d.encryptedDeck[i] = card.Clone()
	}
	orderRank := d.Peer.GetOrderedRanks()
	for _,j := range orderRank {
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
