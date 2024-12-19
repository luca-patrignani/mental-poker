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
				peer: common.Peer{
					Rank:      i,
					Addresses: addresses,
				},
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
				peer: common.Peer{
					Rank:      i,
					Addresses: addresses,
				},
			}
			recvs, err := d.broadcastMultiple(points[i], root, m)
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
