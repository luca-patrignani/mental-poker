package consensus

import (
	"crypto/ed25519"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/luca-patrignani/mental-poker/domain/deck"
	"github.com/luca-patrignani/mental-poker/domain/poker"
	"github.com/luca-patrignani/mental-poker/network"
)

type mockBlock struct {
	Session poker.Session     `json:"session"`
	Action  poker.PokerAction `json:"poker_action"` // Generic action data
	Votes   []Vote            `json:"votes"`
}
type mockBlockChain struct {
	blocks []mockBlock
}

func NewBlockchain() *mockBlockChain {
	bc := &mockBlockChain{
		blocks: make([]mockBlock, 0),
	}

	// Crea genesis block
	genesis := mockBlock{
		Session: poker.Session{},
		Action:  poker.PokerAction{Type: "genesis"},
		Votes:   []Vote{},
	}
	bc.blocks = append(bc.blocks, genesis)

	return bc
}

func (m *mockBlockChain) Append(session poker.Session, pa poker.PokerAction, votes []Vote, proposerID int, quorum int, extra ...map[string]string) error {

	newBlock := mockBlock{
		Session: session,
		Action:  pa,
		Votes:   votes,
	}
	m.blocks = append(m.blocks, newBlock)
	return nil

}

// Verify verifica l'integrità della chain
func (m *mockBlockChain) Verify() error {
	return nil
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

func TestWaitForProposalAndProcess_InvalidJSON(t *testing.T) {

	// create two listeners and addresses
	listeners, addresses := network.CreateListeners(2)
	defer func() {
		for _, l := range listeners {
			_ = l.Close()
		}
	}()

	// create peers (use short timeout)
	timeout := 30 * time.Second
	peer0 := network.NewPeer(0, addresses, listeners[0], timeout)
	peer1 := network.NewPeer(1, addresses, listeners[1], timeout)
	p0 := network.NewP2P(&peer0)
	p1 := network.NewP2P(&peer1)
	defer p0.Close()
	defer p1.Close()

	// create simple keypairs and playersPK (both nodes know both pubkeys)
	pub0, priv0, _ := ed25519.GenerateKey(nil)
	pub1, priv1, _ := ed25519.GenerateKey(nil)
	playersPK := map[int]ed25519.PublicKey{0: pub0, 1: pub1}

	// create nodes with the peers
	s := poker.Session{
		Board: [5]poker.Card{},
		Players: []poker.Player{
			{Name: "Alice", Id: 0, Hand: [2]poker.Card{}, HasFolded: false, Pot: 100, Bet: 0},
			{Name: "Bob", Id: 1, Hand: [2]poker.Card{}, HasFolded: false, Pot: 100, Bet: 0},
		},
		Deck:        deck.Deck{},
		Pots:        []poker.Pot{{Amount: 0, Eligible: []int{0, 1}}},
		HighestBet:  0,
		Dealer:      0,
		CurrentTurn: 0,
		RoundID:     "round1",
	}
	psm := poker.NewPokerManager(&s)
	ldg := NewBlockchain()

	node0 := NewConsensusNode(pub0, priv0, playersPK, psm, ldg, p0)
	node1 := NewConsensusNode(pub1, priv1, playersPK, psm, ldg, p1)
	errChan := make(chan error, 2)

	go func() {
		if err := node1.WaitForProposal(); err != nil {
			errChan <- err
		}
		errChan <- nil
	}()

	// proposer broadcasts invalid JSON bytes (this will be received by node1)
	go func() {
		invalid := []byte("this-is-not-json")
		a, err := node0.network.Broadcast(invalid, 0)
		if err != nil || a == nil {
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

// Full integration test: proposer sends proposal, followers receive, validate, vote, commit
func TestProposeReceive(t *testing.T) {
	// create listeners and peers
	n := 3
	listeners, addresses := network.CreateListeners(n)
	peers := make([]*network.Peer, n)
	for i := 0; i < n; i++ {
		p := network.NewPeer(i, addresses, listeners[i], 30*time.Second)
		peers[i] = &p
	}
	defer func() {
		for i := 0; i < n; i++ {
			err := peers[i].Close()
			if err != nil {
				t.Logf("error closing peer %d: %v", i, err)
			}
		}
	}()

	fatal := make(chan error, n)
	nodes_chan := make(chan *ConsensusNode, n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			playersPK := make(map[int]ed25519.PublicKey)
			pub, priv, err := ed25519.GenerateKey(nil)
			if err != nil {
				fatal <- err
				return
			}
			p2p := network.NewP2P(peers[idx])
			psm := poker.PokerManager{}
			ldg := NewBlockchain()
			node := NewConsensusNode(pub, priv, playersPK, &psm, ldg, p2p)

			err = node.UpdatePeers()
			if err != nil {
				fatal <- err
				return
			}

			players := make([]poker.Player, 0, n)
			for k := range node.playersPK {
				player := poker.Player{
					Name:      "Player" + fmt.Sprint(k),
					Id:        k,
					Hand:      [2]poker.Card{},
					HasFolded: false,
					Bet:       0,
					Pot:       100,
				}
				players = append(players, player)
			}
			sort.Slice(players, func(i, j int) bool {
				return players[i].Id < players[j].Id
			})

			d := deck.Deck{DeckSize: 52, Peer: p2p}
			err = d.PrepareDeck()
			if err != nil {
				fatal <- err
				return
			}

			s := poker.Session{
				Board:       [5]poker.Card{},
				Players:     players,
				Deck:        d,
				Pots:        []poker.Pot{{Amount: 0, Eligible: []int{0, 1, 2}}},
				HighestBet:  0,
				Dealer:      0,
				CurrentTurn: 0,
				RoundID:     "preflop-1",
			}

			psm = *poker.NewPokerManager(&s)
			node.pokerSM = &psm

			nodes_chan <- node
			fatal <- nil
		}(i)
	}

	// Wait for all nodes to initialize
	for i := 0; i < n; i++ {
		if err := <-fatal; err != nil {
			t.Fatalf("node initialization failed: %v", err)
		}
	}
	close(fatal)
	close(nodes_chan)

	var nodes []*ConsensusNode
	for node := range nodes_chan {
		nodes = append(nodes, node)
	}

	// Initialize votes map for all nodes
	for i := 0; i < n; i++ {
		nodes[i].votes = make(map[int]Vote)
	}

	// start receiver goroutines for non-proposers
	done := make(chan struct{}, n-1)
	errChan := make(chan error, n-1)

	// Sync barrier: ensure all receivers are ready
	ready := make(chan struct{}, n-1)

	for i := 0; i < n; i++ {
		idx := nodes[i].pokerSM.FindPlayerIndex(nodes[i].network.GetRank())
		if idx != 0 {
			go func(nodeIdx int) {
				ready <- struct{}{} // Signal ready
				if err := nodes[nodeIdx].WaitForProposal(); err != nil {
					t.Logf("node %d receive error: %v", nodeIdx, err)
					errChan <- err
				} else {
					errChan <- nil
				}
				done <- struct{}{}
			}(i)
		}
	}

	// Wait for all receivers to be ready
	for i := 0; i < n-1; i++ {
		<-ready
	}

	// Small delay to ensure handlers are listening
	time.Sleep(100 * time.Millisecond)

	// Now proposer sends
	for i := 0; i < n; i++ {
		idx := nodes[i].pokerSM.FindPlayerIndex(nodes[i].network.GetRank())
		if idx == 0 {
			// proposer builds action and proposes
			pa := poker.PokerAction{
				RoundID:  "preflop-1",
				PlayerID: nodes[i].network.GetRank(),
				Type:     poker.ActionRaise,
				Amount:   30,
			}

			a, err := MakeAction(nodes[i].network.GetRank(), pa)
			if err != nil {
				t.Fatalf("%s", err.Error())
			}
			err = a.Sign(nodes[i].priv)
			if err != nil {
				t.Fatalf("%s", err.Error())
			}

			if err := nodes[i].ProposeAction(&a); err != nil {
				t.Fatalf("propose failed: %v", err)
			}
			break
		}
	}

	// wait for receivers
	for i := 0; i < n-1; i++ {
		<-done
		if err := <-errChan; err != nil {
			t.Fatalf("receiver error: %v", err)
		}
	}

	// 1. Check that proposal was stored on each node
	for i := 0; i < 3; i++ {
		if nodes[i].proposal == nil {
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
	expectedPot := uint(30)
	for i := 0; i < 3; i++ {
		s := nodes[i].pokerSM.GetSession()

		if s.HighestBet != 30 {
			t.Fatalf("expected HighestBet=%d, got %d on node %d",
				expectedPot, s.HighestBet, i)
		}

		if s.CurrentTurn != 1 {
			t.Fatalf("expected currentTurn=1, got %d on node %d",
				s.CurrentTurn, i)
		}
		if s.Pots[0].Amount != 30 {
			t.Fatalf("expected pot=30, got %d on node %d",
				s.Pots[0].Amount, i)
		}
	}
}

func TestProposeReceiveBan(t *testing.T) {
	// create listeners and peers
	n := 3
	listeners, addresses := network.CreateListeners(n)
	peers := make([]*network.Peer, n)
	for i := 0; i < n; i++ {
		p := network.NewPeer(i, addresses, listeners[i], 30*time.Second)
		peers[i] = &p
	}

	fatal := make(chan error, n)
	nodes_chan := make(chan *ConsensusNode, n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			playersPK := make(map[int]ed25519.PublicKey)
			pub, priv, err := ed25519.GenerateKey(nil)
			if err != nil {
				fatal <- err
				return
			}
			p2p := network.NewP2P(peers[idx])
			psm := poker.PokerManager{}
			ldg := NewBlockchain()
			node := NewConsensusNode(pub, priv, playersPK, &psm, ldg, p2p)

			err = node.UpdatePeers()
			if err != nil {
				fatal <- err
				return
			}

			players := make([]poker.Player, 0, n)
			for k := range node.playersPK {
				player := poker.Player{
					Name:      "Player" + fmt.Sprint(k),
					Id:        k,
					Hand:      [2]poker.Card{},
					HasFolded: false,
					Bet:       0,
					Pot:       100,
				}
				players = append(players, player)
			}
			sort.Slice(players, func(i, j int) bool {
				return players[i].Id < players[j].Id
			})

			d := deck.Deck{DeckSize: 52, Peer: p2p}
			err = d.PrepareDeck()
			if err != nil {
				fatal <- err
				return
			}

			s := poker.Session{
				Board:       [5]poker.Card{},
				Players:     players,
				Deck:        d,
				Pots:        []poker.Pot{{Amount: 50, Eligible: []int{0, 1, 2}}},
				HighestBet:  50,
				Dealer:      2,
				CurrentTurn: 0,
				RoundID:     "preflop-1",
			}

			psm = *poker.NewPokerManager(&s)
			node.pokerSM = &psm

			nodes_chan <- node
			fatal <- nil
		}(i)
	}

	// Wait for all nodes to initialize
	for i := 0; i < n; i++ {
		if err := <-fatal; err != nil {
			t.Fatalf("node initialization failed: %v", err)
		}
	}
	close(fatal)
	close(nodes_chan)

	var nodes []*ConsensusNode
	for node := range nodes_chan {
		nodes = append(nodes, node)
	}

	// Initialize votes map for all nodes
	for i := 0; i < n; i++ {
		nodes[i].votes = make(map[int]Vote)
	}

	// start receiver goroutines for non-proposers
	done := make(chan struct{}, n-1)
	errChan := make(chan error, n-1)

	// Sync barrier: ensure all receivers are ready
	ready := make(chan struct{}, n-1)

	for i := 0; i < n; i++ {
		idx := nodes[i].pokerSM.FindPlayerIndex(nodes[i].network.GetRank())
		if idx != 0 {
			go func(nodeIdx int) {
				ready <- struct{}{} // Signal ready
				if err := nodes[nodeIdx].WaitForProposal(); err != nil {
					t.Logf("node %d receive error: %v", nodeIdx, err)
					errChan <- err
				} else {
					errChan <- nil
				}
				done <- struct{}{}
			}(i)
		}
	}

	// Wait for all receivers to be ready
	for i := 0; i < n-1; i++ {
		<-ready
	}

	// Small delay to ensure handlers are listening
	time.Sleep(100 * time.Millisecond)
	var bannedNodeId int
	// Now proposer sends
	for i := 0; i < n; i++ {
		idx := nodes[i].pokerSM.FindPlayerIndex(nodes[i].network.GetRank())
		if idx == 0 {
			bannedNodeId = nodes[i].network.GetRank()
			// proposer builds action and proposes
			pa := poker.PokerAction{
				RoundID:  "preflop-1",
				PlayerID: nodes[i].network.GetRank(),
				Type:     poker.ActionRaise,
				Amount:   30,
			}

			a, err := MakeAction(nodes[i].network.GetRank(), pa)
			if err != nil {
				t.Fatalf("%s", err.Error())
			}
			err = a.Sign(nodes[i].priv)
			if err != nil {
				t.Fatalf("%s", err.Error())
			}

			if err := nodes[i].ProposeAction(&a); err != nil {
				t.Fatalf("propose failed: %v", err)
			}
			break
		}
	}
	defer func() {
		for i := 0; i < n; i++ {
			if peers[i].Rank != bannedNodeId {
				err := peers[i].Close()
				if err != nil {
					t.Logf("error closing peer %d: %v", i, err)
				}
			}
		}
	}()

	// wait for receivers
	for i := 0; i < n-1; i++ {
		<-done
		if err := <-errChan; err != nil {
			t.Fatalf("receiver error: %v", err)
		}
	}

	// 1. Check that proposal was stored on each node
	for i := 0; i < 3; i++ {
		if nodes[i].proposal == nil {
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
	expectedPot := uint(50)
	for i := 0; i < 3; i++ {
		s := nodes[i].pokerSM.GetSession()

		if s.HighestBet != 50 {
			t.Fatalf("expected HighestBet=%d, got %d on node %d",
				expectedPot, s.HighestBet, i)
		}

		if s.CurrentTurn != 1 {
			t.Fatalf("expected currentTurn=1, got %d on node %d",
				s.CurrentTurn, i)
		}
		if s.Pots[0].Amount != expectedPot {
			t.Fatalf("expected pot=%d, got %d on node %d", expectedPot,
				s.Pots[0].Amount, i)
		}
	}

	// 4. Check that proposer was removed from session (banned)
	for i := 1; i < 3; i++ {
		if idx := nodes[i].pokerSM.FindPlayerIndex(bannedNodeId); idx != -1 {
			t.Fatalf("expected proposer to be banned, still found at index %d", idx)
		}
	}

}
