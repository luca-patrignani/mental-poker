package deck

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/luca-patrignani/mental-poker/network"
	"go.dedis.ch/kyber/v4"
)

func TestAllToAllSingle(t *testing.T) {
	n := 10
	listeners, addresses := network.CreateListeners(n)
	points := []kyber.Point{}
	for i := 0; i < n; i++ {
		points = append(points, suite.Point().Mul(suite.Scalar().Pick(suite.RandomStream()), nil))
	}
	errChan := make(chan error)
	for i := 0; i < n; i++ {
		go func() {
			p := network.NewPeer(i, addresses, listeners[i], 30*time.Second)
			d := Deck{
				Peer: network.NewP2P(&p),
			}
			defer d.Peer.Close()
			_, err := d.allToAllSingle(points[i])
			if err != nil {
				errChan <- errors.Join(fmt.Errorf("error from %d", i), err)
				return
			}
			_, err = d.allToAllSingle(points[i])
			if err != nil {
				errChan <- errors.Join(fmt.Errorf("error from %d", i), err)
				return
			}
			_, err = d.allToAllSingle(points[i])
			if err != nil {
				errChan <- errors.Join(fmt.Errorf("error from %d", i), err)
				return
			}
			recvs, err := d.allToAllSingle(points[i])
			if err != nil {
				errChan <- errors.Join(fmt.Errorf("error from %d", i), err)
				return
			}
			for j := 0; j < n; j++ {
				if !recvs[j].Equal(points[j]) {
					errChan <- fmt.Errorf("expected %s, actual %s", points[j].String(), recvs[j].String())
				}
			}
			errChan <- nil
		}()
	}
	for i := 0; i < n; i++ {
		err := <-errChan
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestBroadcastMultiple(t *testing.T) {
	n := 10
	m := 52
	listeners, addresses := network.CreateListeners(n)
	points := make([][]kyber.Point, m)
	for i := 0; i < n; i++ {
		points[i] = []kyber.Point{}
		for j := 0; j < m; j++ {
			points[i] = append(points[i], suite.Point().Mul(suite.Scalar().Pick(suite.RandomStream()), nil))
		}
	}
	errChan := make(chan error)
	root := 3
	for i := 0; i < n; i++ {
		go func() {
			p := network.NewPeer(i, addresses, listeners[i], 30*time.Second)
			d := Deck{
				Peer: network.NewP2P(&p),
			}
			defer d.Peer.Close()
			var recvs []kyber.Point
			var err error
			if d.Peer.GetRank() == root {
				recvs, err = d.broadcastMultiple(points[i], root, m)

			} else {
				recvs, err = d.broadcastMultiple(nil, root, m)
			}
			if err != nil {
				errChan <- errors.Join(fmt.Errorf("error from %d", i), err)
				return
			}
			for j := 0; j < m; j++ {
				if !recvs[j].Equal(points[root][j]) {
					errChan <- fmt.Errorf("expected %s, actual %s", points[root][j].String(), recvs[j].String())
				}
			}
			errChan <- nil
		}()
	}
	for i := 0; i < n; i++ {
		err := <-errChan
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestBroadcastSingle(t *testing.T) {
	n := 10
	root := 3
	listeners, addresses := network.CreateListeners(n)
	point := suite.Point().Mul(suite.Scalar().Pick(suite.RandomStream()), nil)
	errChan := make(chan error)
	for i := 0; i < n; i++ {
		go func() {
			p := network.NewPeer(i, addresses, listeners[i], 30*time.Second)
			d := Deck{
				Peer: network.NewP2P(&p),
			}
			defer d.Peer.Close()
			var recv kyber.Point
			var err error
			if i == root {
				recv, err = d.broadcastSingle(point, root)
			} else {
				recv, err = d.broadcastSingle(nil, root)
			}
			if err != nil {
				errChan <- errors.Join(fmt.Errorf("error from %d", i), err)
				return
			}
			if !recv.Equal(point) {
				errChan <- fmt.Errorf("expected %s, actual %s", point.String(), recv.String())
			}
			errChan <- nil
		}()
	}
	for i := 0; i < n; i++ {
		err := <-errChan
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestGenerateRandomElement(t *testing.T) {
	n := 10
	listeners, addresses := network.CreateListeners(n)
	errChan := make(chan error)
	points := make(chan kyber.Point, n)
	for i := 0; i < n; i++ {
		go func() {
			peer := network.NewPeer(i, addresses, listeners[i], 30*time.Second)
			deck := Deck{
				DeckSize: 52,
				Peer:     network.NewP2P(&peer),
			}
			defer deck.Peer.Close()
			_, err := deck.generateRandomElement()
			if err != nil {
				errChan <- err
				return
			}
			_, err = deck.generateRandomElement()
			if err != nil {
				errChan <- err
				return
			}
			_, err = deck.generateRandomElement()
			if err != nil {
				errChan <- err
				return
			}
			p, err := deck.generateRandomElement()
			if err != nil {
				errChan <- err
			}
			points <- p
			errChan <- nil
		}()
	}
	for i := 0; i < n; i++ {
		err := <-errChan
		if err != nil {
			t.Fatal(err)
		}
	}
	close(points)
	p := <-points
	for pp := range points {
		if !pp.Equal(p) {
			t.Fatal(pp, p)
		}
	}
}

func TestDrawCardOpenCard(t *testing.T) {
	n := 10
	drawer := 0
	listeners, addresses := network.CreateListeners(n)
	errChan := make(chan error)
	cardChan := make(chan int, 10)
	for i := 0; i < n; i++ {
		go func() {
			p := network.NewPeer(i, addresses, listeners[i], 30*time.Second)
			deck := Deck{
				DeckSize: 52,
				Peer:     network.NewP2P(&p),
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
			card, err := deck.DrawCard(drawer)
			//t.Logf("player %d draw card %d", i, card)
			if err != nil && i == drawer {
				errChan <- err
				return
			}
			cardB, err := deck.OpenCard(drawer, card)
			//t.Logf("Player %d sees that the %dth player's card is a %d", i, drawer, cardB)
			if err != nil {
				errChan <- err
				return
			}
			cardChan <- cardB
			errChan <- nil
		}()
	}
	for i := 0; i < n; i++ {
		err := <-errChan
		if err != nil {
			t.Fatal(err)
		}
	}
	card := <-cardChan
	for i := 1; i < n; i++ {
		c := <-cardChan
		if card != c {
			t.Fatal(card, c)
		}
	}
}
