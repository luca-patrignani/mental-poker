package communication

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/luca-patrignani/mental-poker/common"
)

// Test WaitForProposalAndProcess behavior when the proposer sends invalid bytes.
func TestWaitForProposalAndProcess_InvalidJSON(t *testing.T) {
	// create two listeners and addresses
	listeners, addresses := common.CreateListeners(2)
	defer func() {
		for _, l := range listeners {
			_ = l.Close()
		}
	}()

	// create peers (use short timeout)
	timeout := 2 * time.Second
	p0 := common.NewPeer(0, addresses, listeners[0], timeout)
	p1 := common.NewPeer(1, addresses, listeners[1], timeout)
	defer p0.Close()
	defer p1.Close()

	// create simple keypairs and playersPK (both nodes know both pubkeys)
	pub0, priv0 := mustKeypair(t)
	pub1, priv1 := mustKeypair(t)
	playersPK := map[string]ed25519.PublicKey{"0": pub0, "1": pub1}

	// create nodes with the peers
	node0 := NewNode(&p0, pub0, priv0, playersPK)
	node1 := NewNode(&p1, pub1, priv1, playersPK)

	// set node1.Session.CurrentTurn = 0 so it expects proposer rank 0
	// use reflection-free approach: fill Session.Players minimally via helper used earlier in other tests
	setSessionPlayers(t, node1, 2)
	setSessionPlayers(t, node0, 2)

	// ensure CurrentTurn == 0 for node1
	// (setSessionPlayers sets CurrentTurn=0)

	// run waiter in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- node1.WaitForProposalAndProcess()
	}()

	// let goroutine start and wait a little
	time.Sleep(100 * time.Millisecond)

	// proposer broadcasts invalid JSON bytes (this will be received by node1)
	invalid := []byte("this-is-not-json")
	if _, err := node0.peer.Broadcast(invalid, 0); err != nil {
		t.Fatalf("proposer Broadcast failed: %v", err)
	}

	// get waiter result
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatalf("expected error from WaitForProposalAndProcess due to invalid JSON")
		}
		// error message contains 'invalid proposal bytes' per implementation wrapping
		// we check substring
		if !contains(err.Error(), "invalid proposal bytes") {
			t.Fatalf("unexpected error from WaitForProposalAndProcess: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for WaitForProposalAndProcess result")
	}
}

// helper contains check (small helper to avoid importing strings only for this)
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (len(sub) == 0 || (len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()))
}
