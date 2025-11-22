package deck

import (
	"testing"
	"time"

	"github.com/luca-patrignani/mental-poker/network"
	"go.dedis.ch/kyber/v4"
)

func TestShuffle(t *testing.T) {
	n := 10
	listeners, addresses := network.CreateListeners(n)
	errChan := make(chan error)
	decks := make(chan []kyber.Point, n)
	for i := 0; i < n; i++ {
		go func() {
			p := network.NewPeer(i, addresses, listeners[i], 1000*time.Second)
			deck := Deck{
				DeckSize: 52,
				Peer:     network.NewP2P(&p),
			}
			defer func() {
				if err := deck.Peer.Close(); err != nil {
					errChan <- err
				}
			}()
			err := deck.PrepareDeck()
			if err != nil {
				errChan <- err
				return
			}
			err = deck.Shuffle()
			if err != nil {
				errChan <- err
				return
			}
			decks <- deck.encryptedDeck
			errChan <- nil
		}()
	}
	for i := 0; i < n; i++ {
		err := <-errChan
		if err != nil {
			t.Fatal(err)
		}
	}
	close(decks)
	deck := <-decks
	for d := range decks {
		if len(d) != 53 {
			t.Fatal(len(d))
		}
		for i := range d {
			if !deck[i].Equal(d[i]) {
				t.Fatal(deck, d)
			}
		}
	}
}
