package deck

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/luca-patrignani/mental-poker/common"
	"go.dedis.ch/kyber/v3"
)

func createAddresses(n int) []net.TCPAddr {
	addresses := []net.TCPAddr{}
	for i := 0; i < n; i++ {
		addresses = append(addresses, net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 50000 + i,
		})
		if addresses[i].IP == nil {
			panic(addresses[i].IP)
		}
	}
	return addresses
}

func TestAllToAllSingle(t *testing.T) {
	n := 10
	addresses := createAddresses(n)
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
