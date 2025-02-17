package deck

import (
	"testing"
	"time"

	"github.com/luca-patrignani/mental-poker/common"
	"go.dedis.ch/kyber/v4"
)

func TestShuffle(t *testing.T) {
	n := 10
	listeners, addresses := common.CreateListeners(n)
	errChan := make(chan error)
	decks := make(chan []kyber.Point, n)
	for i := 0; i < n; i++ {
		go func() {
			deck := Deck{
				DeckSize: 52,
				Peer:     common.NewPeer(i, addresses, listeners[i], time.Second),
			}
			defer deck.Peer.Close()
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
			decks <- deck.EncryptedDeck
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
