package blockchain

import (
	"crypto/ed25519"
	"fmt"
	"testing"
	"time"

	"github.com/luca-patrignani/mental-poker/common"
	"github.com/luca-patrignani/mental-poker/deck"
	"github.com/luca-patrignani/mental-poker/poker"
)

// SampleSessionForTest crea una sessione di test semplice e deterministica.
// - ids: lista di player IDs (es. ["p0","p1","p2"])
// - assegna Rank = indice, Pot iniziale = 1000 (configurabile qui), CurrentTurn = 0, Dealer = 0
// - Board e Hands sono lasciati a zero-value (da popolare dal test se serve)
func SampleSessionForTest(ids []string) poker.Session {
	players := make([]poker.Player, len(ids))
	for i, id := range ids {
		players[i] = poker.Player{
			Name:      id,
			Rank:      i,
			Hand:      [2]poker.Card{}, // mano vuota — i test che hanno bisogno di carte possono impostarla
			HasFolded: false,
			Bet:       0,
			Pot:       1000, // valore di default, cambia se vuoi
		}
	}

	return poker.Session{
		Board:       [5]poker.Card{}, // empty board
		Players:     players,
		Deck:        deck.Deck{},
		Pot:         0,
		HighestBet:  0,
		Dealer:      0,
		CurrentTurn: 0,
		RoundID:     "round-0",
		LastIndex:   0,
	}
}

func TestProposeReceive(t *testing.T) {
    // create listeners and peers
    listeners, addresses := common.CreateListeners(3)
    peers := make([]*common.Peer, 3)
    for i := 0; i < 3; i++ {
        p := common.NewPeer(i, addresses, listeners[i], 5*time.Second)
        peers[i] = &p
    }
    defer func() {
        for i := 0; i < 3; i++ { _ = peers[i].Close() }
    }()

    // generate keys and players map
    playersPK := make(map[string]ed25519.PublicKey)
    privs := make([]ed25519.PrivateKey, 3)
    ids := make([]string, 3)
    for i := 0; i < 3; i++ {
        pub, priv, _ := NewEd25519Keypair()
        ids[i] = "p" + fmt.Sprintf("0%d", i)
        playersPK[ids[i]] = pub
        privs[i] = priv
    }

    // create nodes
    nodes := make([]*Node, 3)
    for i := 0; i < 3; i++ {
        nodes[i] = NewNode(ids[i], peers[i], playersPK[ids[i]], privs[i], playersPK)
    }

    // set identical session state on all nodes (simple)
    for _, n := range nodes {
        n.Session = SampleSessionForTest(ids)
    }

    // start receiver goroutines for non-proposers
    done := make(chan struct{})
    for i := 1; i < 3; i++ {
        go func(idx int) {
            if err := nodes[idx].WaitForProposalAndProcess(); err != nil {
                t.Logf("node %d receive error: %v", idx, err)
            }
            done <- struct{}{}
        }(i)
    }

    // proposer builds action and proposes
    a := &Action{
        RoundID:  nodes[0].Session.RoundID,
        PlayerID: nodes[0].Session.Players[0].Name,
        Type:     ActionBet,
        Amount:   10,
    }
    _ = a.Sign(privs[0])

    if err := nodes[0].ProposeAction(a); err != nil {
        t.Fatalf("propose failed: %v", err)
    }

    // wait for receivers
    <-done
    <-done

    time.Sleep(500 * time.Millisecond) // wait a bit for votes to be processed

    // 1. Check that proposal was stored on each node
    for i := 0; i < 3; i++ {
        //t.Logf("Node %d proposals: %+v\n", i, nodes[i].proposals)
        if nodes[i].proposals == nil || len(nodes[i].proposals) != 1 {
            t.Fatalf("node %d did not store proposal", i)
        }
    }

    // 2. Check that votes were produced by followers
    for i := 1; i < 3; i++ {
        if len(nodes[i].votes) == 0 {
            t.Fatalf("node %d did not cast any vote", i)
        }
    }

    // 3. Check that commit affected session state (e.g. pot updated)
    expectedPot := uint(10)
    for i := 0; i < 3; i++ {
        if nodes[i].Session.Pot != expectedPot {
            t.Fatalf("expected pot=%d, got %d on node %d",
                expectedPot, nodes[i].Session.Pot, i)
        }
    }
}
