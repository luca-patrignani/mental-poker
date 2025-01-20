package deck

import (
	"errors"
	"fmt"
	"testing"

	"github.com/luca-patrignani/mental-poker/common"
	"go.dedis.ch/kyber/v3"
)

func TestAllToAllSingle(t *testing.T) {
	n := 10
	addresses := common.CreateAddresses(n)
	points := []kyber.Point{}
	for i := 0; i < n; i++ {
		points = append(points, suite.Point().Mul(suite.Scalar().Pick(suite.RandomStream()), nil))
	}
	errChan := make(chan error)
	for i := 0; i < n; i++ {
		go func() {
			d := Deck{
				Peer: common.NewPeer(i, addresses),
			}
			defer d.Peer.Close()
			d.allToAllSingle(points[i])
			d.allToAllSingle(points[i])
			d.allToAllSingle(points[i])
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
	addresses := common.CreateAddresses(n)
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
			d := Deck{
				Peer: common.NewPeer(i, addresses),
			}
			defer d.Peer.Close()
			var recvs []kyber.Point
			var err error
			if d.Peer.Rank == root {
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
	addresses := common.CreateAddresses(n)
	point := suite.Point().Mul(suite.Scalar().Pick(suite.RandomStream()), nil)
	errChan := make(chan error)
	for i := 0; i < n; i++ {
		go func() {
			d := Deck{
				Peer: common.NewPeer(i, addresses),
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
	addresses := common.CreateAddresses(n)
	errChan := make(chan error)
	points := make(chan kyber.Point, n)
	for i := 0; i < n; i++ {
		go func() {
			deck := Deck{
				DeckSize: 52,
				Peer: common.NewPeer(i, addresses),
			}
			defer deck.Peer.Close()
			deck.generateRandomElement()
			deck.generateRandomElement()
			deck.generateRandomElement()
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
