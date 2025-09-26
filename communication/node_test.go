package communication

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/luca-patrignani/mental-poker/common"
	"github.com/luca-patrignani/mental-poker/deck"
	"github.com/luca-patrignani/mental-poker/poker"
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

	s := poker.Session{
		Board: [5]poker.Card{},
		Players: []poker.Player{
			{Rank: 0, Hand: [2]poker.Card{}, HasFolded: false, Pot: 100, Bet: 0},
			{Rank: 1, Hand: [2]poker.Card{}, HasFolded: false, Pot: 100, Bet: 0},
		},
		Deck:        deck.Deck{},
		Pots:        []poker.Pot{{Amount: 0, Eligible: []int{0, 1}}},
		HighestBet:  0,
		Dealer:      0,
		CurrentTurn: 0,
		RoundID:     "round1",
	}
	node0.Session = &s
	node1.Session = &s

	errChan := make(chan error, 2)

	go func() {
		if err := node1.WaitForProposalAndProcess(); err != nil {
			errChan <- err
		}
		errChan <- nil
	}()

	// proposer broadcasts invalid JSON bytes (this will be received by node1)
	go func() {
		invalid := []byte("this-is-not-json")
		if _, err := node0.peer.Broadcast(invalid, 0); err != nil {
			errChan <- err
		}
		errChan <- nil
	}()

	err0 := <-errChan
	err1 := <-errChan
	close(errChan)
	if err0 == nil && err1 == nil {
		t.Fatalf("expected error from WaitForProposalAndProcess for Invalid JSON")
	}
}
