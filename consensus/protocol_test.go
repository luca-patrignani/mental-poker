package consensus

import (
	"crypto/ed25519"
	"testing"
)

func makeSignedVote(t *testing.T, actionID string, voterID int, value VoteValue, reason string, priv ed25519.PrivateKey) Vote {
	t.Helper()
	v := Vote{ActionId: actionID,
		VoterID: voterID,
		Value:   value,
		Reason:  reason}
	_ = v.Sign(priv)
	return v
}

func TestEnsureSameProposal(t *testing.T) {
	v1 := Vote{ActionId: "p1"}
	v2 := Vote{ActionId: "p1"}
	v3 := Vote{ActionId: "p2"}

	err := ensureSameProposal([]Vote{})
	if err == nil {
		t.Fatalf("expected error for empty slice")
	}

	err = ensureSameProposal([]Vote{v1, v2})
	if err != nil {
		t.Fatalf("expected success for matching proposals, got %v", err)
	}

	err = ensureSameProposal([]Vote{v1, v3})
	if err == nil {
		t.Fatalf("expected error for mismatched proposals")
	}
}

func TestCollectVotes(t *testing.T) {
	m := map[int]Vote{
		1:  {VoterID: 1, Value: VoteAccept},
		12: {VoterID: 12, Value: VoteReject},
		13: {VoterID: 13, Value: VoteAccept},
	}
	accepts := collectVotes(m, VoteAccept)
	if len(accepts) != 2 {
		t.Fatalf("expected 2 accepts, got %d", len(accepts))
	}
	rejects := collectVotes(m, VoteReject)
	if len(rejects) != 1 {
		t.Fatalf("expected 1 reject, got %d", len(rejects))
	}
}

/*// Full integration test: proposer sends proposal, followers receive, validate, vote, commit
func TestProposeReceive(t *testing.T) {
	// create listeners and peers
	listeners, addresses := common.CreateListeners(3)
	peers := make([]*common.Peer, 3)
	for i := 0; i < 3; i++ {
		p := common.NewPeer(i, addresses, listeners[i], 5*time.Second)
		peers[i] = &p
	}
	defer func() {
		for i := 1; i < 3; i++ {
			err := peers[i].Close()
			if err != nil {
				t.Logf("error closing peer %d: %v", i, err)
			}
		}
	}()

	fatal := make(chan error, 3)
	nodes_chan := make(chan *Node, 3)

	for i := 0; i < 3; i++ {
		go func() {
			playersPK := make(map[string]ed25519.PublicKey)
			pub, priv := mustKeypair(t)
			node := NewNode(peers[i], pub, priv, playersPK)

			b, err := json.Marshal(pub)
			if err != nil {
				fatal <- err
				return
			}
			pkBytes, err := node.peer.AllToAll(b)
			if err != nil {
				fatal <- err
				return
			}
			pk := make(map[string]ed25519.PublicKey, len(pkBytes))
			for i, pki := range pkBytes {
				var p ed25519.PublicKey
				if err := json.Unmarshal(pki, &p); err != nil {
					fatal <- fmt.Errorf("failed to unmarshal vote: %v\n", err)
					continue // skip malformed messages
				}
				pk[strconv.Itoa(i)] = p
			}
			node.PlayersPK = pk

			deck := deck.Deck{
				DeckSize: 52,
				Peer:     *node.peer,
			}

			err = deck.PrepareDeck()
			if err != nil {
				fatal <- err
				return
			}
			players := make([]poker.Player, len(pk))
			i := 0
			for k := range pk {
				players[i] = poker.Player{
					Name:      k,
					Rank:      i,
					Hand:      [2]poker.Card{},
					HasFolded: false,
					Bet:       0,
					Pot:       100,
				}
				i++
			}

			s := poker.Session{
				Board:       [5]poker.Card{}, // empty board
				Players:     players,
				Deck:        deck,
				Pots:        []poker.Pot{{Amount: 0, Eligible: []int{0, 1, 2}}},
				HighestBet:  0,
				Dealer:      0,
				CurrentTurn: 0,
				RoundID:     "round-0",
			}
			node.Session = &s
			node.N = len(node.Session.Players)
			node.quorum = ceil2n3(node.N)
			nodes_chan <- node
			fatal <- nil
		}()
	}
	<-fatal
	<-fatal
	<-fatal
	close(fatal)
	close(nodes_chan)

	var nodes []*Node
	for n := range nodes_chan {
		nodes = append(nodes, n)
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
		RoundID:    nodes[0].Session.RoundID,
		PlayerID:   strconv.Itoa(nodes[0].peer.Rank),
		CommitType: poker.ActionBet,
		Amount:     10,
	}
	_ = a.Sign(nodes[0].Priv)

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
		if nodes[i].Session.Pots[0].Amount != expectedPot {
			t.Fatalf("expected pot=%d, got %d on node %d",
				expectedPot, nodes[i].Session.Pots[0].Amount, i)
		}
	}
}

// Full integration test: proposer sends malformed proposal. Followers should reject it and not update state.
func TestProposeReceiveAndBan(t *testing.T) {
	// create listeners and peers
	listeners, addresses := common.CreateListeners(3)
	peers := make([]*common.Peer, 3)
	for i := 0; i < 3; i++ {
		p := common.NewPeer(i, addresses, listeners[i], 5*time.Second)
		peers[i] = &p
	}
	defer func() {
		for i := 1; i < 3; i++ {
			err := peers[i].Close()
			if err != nil {
				t.Logf("error closing peer %d: %v", i, err)
			}
		}
	}()

	fatal := make(chan error, 3)
	nodes_chan := make(chan *Node, 3)

	for i := 0; i < 3; i++ {
		go func() {
			playersPK := make(map[string]ed25519.PublicKey)
			pub, priv := mustKeypair(t)
			node := NewNode(peers[i], pub, priv, playersPK)

			b, err := json.Marshal(pub)
			if err != nil {
				fatal <- err
				return
			}
			pkBytes, err := node.peer.AllToAll(b)
			if err != nil {
				fatal <- err
				return
			}
			pk := make(map[string]ed25519.PublicKey, len(pkBytes))
			for i, pki := range pkBytes {
				var p ed25519.PublicKey
				if err := json.Unmarshal(pki, &p); err != nil {
					fatal <- fmt.Errorf("failed to unmarshal public key: %v\n", err)
					continue // skip malformed messages
				}
				pk[strconv.Itoa(i)] = p
			}
			node.PlayersPK = pk

			deck := deck.Deck{
				DeckSize: 52,
				Peer:     *node.peer,
			}

			err = deck.PrepareDeck()
			if err != nil {
				fatal <- err
				return
			}
			players := make([]poker.Player, len(pk))
			i := 0
			for k := range pk {
				players[i] = poker.Player{
					Name:      k,
					Rank:      i,
					Hand:      [2]poker.Card{},
					HasFolded: false,
					Bet:       0,
					Pot:       100,
				}
				i++
			}

			s := poker.Session{
				Board:       [5]poker.Card{}, // empty board
				Players:     players,
				Deck:        deck,
				Pots:        []poker.Pot{{Amount: 0, Eligible: []int{0, 1, 2}}},
				HighestBet:  0,
				Dealer:      0,
				CurrentTurn: 0,
				RoundID:     "round-0",
			}
			node.Session = &s
			node.N = len(node.Session.Players)
			node.quorum = ceil2n3(node.N)
			nodes_chan <- node
			fatal <- nil
		}()
	}
	<-fatal
	<-fatal
	<-fatal
	close(fatal)
	close(nodes_chan)

	var nodes []*Node
	for n := range nodes_chan {
		nodes = append(nodes, n)
	}
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
		RoundID:    nodes[0].Session.RoundID,
		PlayerID:   strconv.Itoa(nodes[0].peer.Rank),
		CommitType: poker.ActionBet,
		Amount:     110, // too high, should trigger validation error
	}
	_ = a.Sign(nodes[0].Priv)

	err := nodes[0].ProposeAction(a)
	if err != nil {
		t.Fatalf("propose failed: %v", err)
	}

	// wait for receivers
	<-done
	<-done

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

	// 3. Check that commit did NOT affect session state (e.g. pot unchanged)
	expectedPot := uint(0)
	for i := 0; i < 3; i++ {
		if nodes[i].Session.Pots[0].Amount != expectedPot {
			t.Fatalf("expected pot=%d, got %d on node %d",
				expectedPot, nodes[i].Session.Pots[0].Amount, i)
		}
	}

	// 4. Check that proposer was removed from session (banned)
	for i := 1; i < 3; i++ {
		if idx := nodes[i].findPlayerIndex(nodes[0].ID); idx != -1 {
			t.Fatalf("expected proposer to be banned, still found at index %d", idx)
		}
	}
}*/
