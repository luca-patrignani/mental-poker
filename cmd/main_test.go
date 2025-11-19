package main

import (
	"fmt"
	"net"
	"slices"
	"testing"

	"github.com/luca-patrignani/mental-poker/network"
)

var mapListeners map[int]net.Listener
var addresses []string
var n int = 10

func setup() {
	if mapListeners == nil {
		ml, mapAddresses := network.CreateListeners(n)
		addresses = []string{}
		for i := 0; i < n; i++ {
			addresses = append(addresses, mapAddresses[i])
		}
		mapListeners = ml
	}
}

func TestTestConnections(t *testing.T) {
	setup()
	errChan := make(chan error)
	for i := range n {
		go func() {
			p2p := createP2P(i, addresses, mapListeners[i], []network.PeerOption{})
			names, err := testConnections(p2p, "name"+fmt.Sprint(i))
			if err != nil {
				errChan <- err
				return
			}
			if len(names) != n {
				errChan <- fmt.Errorf("expected %d names, got %d", n, len(names))
				return
			}
			slices.Sort(names)
			for j := range n {
				expectedName := "name" + fmt.Sprint(j)
				if names[j] != expectedName {
					errChan <- fmt.Errorf("expected name %s, got %s", expectedName, names[j])
					return
				}
			}
			errChan <- nil
		}()
	}
	for range n {
		err := <-errChan
		if err != nil {
			t.Fatal(err)
		}
	}
}
