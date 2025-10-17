package deck

import (
	"math/rand"

	"go.dedis.ch/kyber/v4"
)

func (d *Deck) Shuffle() error {
	d.EncryptedDeck = make([]kyber.Point, d.DeckSize+1)
	for i, card := range d.CardCollection {
		d.EncryptedDeck[i] = card.Clone()
	}
	for j := 0; j < len(d.Peer.GetAddresses()); j++ {
		if j == d.Peer.GetRank() {
			x := suite.Scalar().Pick(suite.RandomStream())
			d.SecretKey = x
			perm := permutation(d.DeckSize)
			tmp := make([]kyber.Point, d.DeckSize+1)
			for i, card := range d.EncryptedDeck {
				tmp[i] = card.Clone()
			}
			for i := 0; i <= d.DeckSize; i++ {
				d.EncryptedDeck[i].Mul(x, tmp[perm[i]])
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
