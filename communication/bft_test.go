package communication

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
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
	idsInt := make([]int, len(ids))
	for i, id := range ids {
		players[i] = poker.Player{
			Name:      id,
			Rank:      i,
			Hand:      [2]poker.Card{},
			HasFolded: false,
			Bet:       0,
			Pot:       100,
		}
		idsInt[i] = i
	}

	return poker.Session{
		Board:       [5]poker.Card{}, // empty board
		Players:     players,
		Deck:        deck.Deck{},
		Pots:        []poker.Pot{{Amount: 0, Eligible: idsInt}},
		HighestBet:  0,
		Dealer:      0,
		CurrentTurn: 0,
		RoundID:     "round-0",
	}
}

func makeSignedVote(t *testing.T, proposalID, voterID string, value VoteValue, priv ed25519.PrivateKey) VoteMsg {
	t.Helper()
	toSign, _ := json.Marshal(struct {
		ProposalID string    `json:"proposal_id"`
		VoterID    string    `json:"voter_id"`
		Value      VoteValue `json:"value"`
	}{proposalID, voterID, value})
	sig := ed25519.Sign(priv, toSign)
	return VoteMsg{ProposalID: proposalID, VoterID: voterID, Value: value, Reason: "test", Sig: sig}
}

// setSessionPlayers uses reflection to populate node.Session.Players with n
// players where each player has Rank==i and HasFolded==false. It also sets
// CurrentTurn to 0.
func setSessionPlayers(t *testing.T, node *Node, n int) {
	t.Helper()
	sv := reflect.ValueOf(&node.Session).Elem()
	f := sv.FieldByName("Players")
	if !f.IsValid() {
		t.Fatalf("Session.Players field not found via reflection")
	}
	elem := f.Type().Elem()
	slice := reflect.MakeSlice(f.Type(), n, n)
	for i := 0; i < n; i++ {
		pv := reflect.New(elem).Elem()
		if fv := pv.FieldByName("Rank"); fv.IsValid() && fv.CanSet() {
			fv.SetInt(int64(i))
		}
		if fv := pv.FieldByName("HasFolded"); fv.IsValid() && fv.CanSet() {
			fv.SetBool(false)
		}
		// set some default Pot and Bet fields if present
		if fv := pv.FieldByName("Pot"); fv.IsValid() && fv.CanSet() {
			// prefer unsigned sets
			switch fv.Kind() {
			case reflect.Uint, reflect.Uint32, reflect.Uint64:
				fv.SetUint(100)
			case reflect.Int, reflect.Int32, reflect.Int64:
				fv.SetInt(100)
			}
		}
		if fv := pv.FieldByName("Bet"); fv.IsValid() && fv.CanSet() {
			switch fv.Kind() {
			case reflect.Uint, reflect.Uint32, reflect.Uint64:
				fv.SetUint(0)
			case reflect.Int, reflect.Int32, reflect.Int64:
				fv.SetInt(0)
			}
		}
		slice.Index(i).Set(pv)
	}
	f.Set(slice)

	// set CurrentTurn = 0 if present
	if ct := sv.FieldByName("CurrentTurn"); ct.IsValid() && ct.CanSet() {
		switch ct.Kind() {
		case reflect.Uint, reflect.Uint32, reflect.Uint64:
			ct.SetUint(0)
		case reflect.Int, reflect.Int32, reflect.Int64:
			ct.SetInt(0)
		}
	}
}

func TestEnsureSameProposal(t *testing.T) {
	v1 := VoteMsg{ProposalID: "p1"}
	v2 := VoteMsg{ProposalID: "p1"}
	v3 := VoteMsg{ProposalID: "p2"}

	err, id := ensureSameProposal([]VoteMsg{})
	if err == nil {
		t.Fatalf("expected error for empty slice")
	}
	if id != "" {
		t.Fatalf("expected empty id for empty slice")
	}

	err, id = ensureSameProposal([]VoteMsg{v1, v2})
	if err != nil || id != "p1" {
		t.Fatalf("expected success for matching proposals, got %v, %s", err, id)
	}

	err, _ = ensureSameProposal([]VoteMsg{v1, v3})
	if err == nil {
		t.Fatalf("expected error for mismatched proposals")
	}
}

func TestCollectVotes(t *testing.T) {
	m := map[string]VoteMsg{
		"a": {VoterID: "a", Value: VoteAccept},
		"b": {VoterID: "b", Value: VoteReject},
		"c": {VoterID: "c", Value: VoteAccept},
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

func TestApplyActionToSessionAndValidate(t *testing.T) {
	pub, priv := mustKeypair(t)
	playersPK := map[string]ed25519.PublicKey{"0": pub}
	node := &Node{ID: "n1", Pub: pub, Priv: priv, PlayersPK: playersPK, N: 1, quorum: 1, proposals: make(map[string]ProposalMsg), votes: make(map[string]map[string]VoteMsg)}
	setSessionPlayers(t, node, 1)

	// prepare an action and sign it
	a := &Action{RoundID: "r1", PlayerID: "0", Type: poker.ActionBet, Amount: 10}
	err := a.Sign(priv)
	if err != nil {
		t.Fatalf("action sign failed: %v", err)
	}

	// invalid round
	aBad := *a
	aBad.RoundID = "other"
	if err := node.validateActionAgainstSession(&aBad); err == nil {
		t.Fatalf("expected wrong round error")
	}

	// out-of-turn: set CurrentTurn to 1 (non-existent)
	sv := reflect.ValueOf(&node.Session).Elem()
	if ct := sv.FieldByName("CurrentTurn"); ct.IsValid() && ct.CanSet() {
		switch ct.Kind() {
		case reflect.Uint, reflect.Uint32, reflect.Uint64:
			ct.SetUint(1)
		}
	}
	if err := node.validateActionAgainstSession(a); err == nil {
		t.Fatalf("expected out-of-turn error")
	}

	// restore turn and set amount 0 for bet
	if ct := sv.FieldByName("CurrentTurn"); ct.IsValid() && ct.CanSet() {
		switch ct.Kind() {
		case reflect.Uint, reflect.Uint32, reflect.Uint64:
			ct.SetUint(0)
		}
	}
	aZero := *a
	aZero.Amount = 0
	if err := node.validateActionAgainstSession(&aZero); err == nil {
		t.Fatalf("expected bad amount error")
	}

	// now test applyActionToSession: give player enough Pot
	// set player's Pot to 100
	playersField := sv.FieldByName("Players")
	if playersField.Len() == 0 {
		t.Fatalf("no players found")
	}
	p0 := playersField.Index(0)
	if pv := p0.FieldByName("Pot"); pv.IsValid() && pv.CanSet() {
		switch pv.Kind() {
		case reflect.Uint, reflect.Uint32, reflect.Uint64:
			pv.SetUint(100)
		case reflect.Int, reflect.Int32, reflect.Int64:
			pv.SetInt(100)
		}
	}

	// call applyActionToSession
	if err := node.applyActionToSession(a, 0); err != nil {
		t.Fatalf("applyActionToSession failed: %v", err)
	}
}

func TestProposalIDAndApplyCommitAndBanCert(t *testing.T) {
	// setup keys and node
	pubA, privA := mustKeypair(t)
	pubB, privB := mustKeypair(t)
	playersPK := map[string]ed25519.PublicKey{"1": pubA, "2": pubB}
	node := &Node{ID: "n1", Pub: pubA, Priv: privA, PlayersPK: playersPK, N: 2, quorum: 1, proposals: make(map[string]ProposalMsg), votes: make(map[string]map[string]VoteMsg)}
	setSessionPlayers(t, node, 2)

	// create action by player "1" and sign
	a := &Action{RoundID: "r1", PlayerID: "1", Type: poker.ActionCheck}
	err := a.Sign(privA)
	if err != nil {
		t.Fatalf("action sign failed: %v", err)
	}

	pid, err := proposalID(a)
	if err != nil || pid == "" {
		t.Fatalf("proposalID failed: %v", err)
	}

	// create a proposal and store it
	prop := makeProposalMsg(a, a.Signature)
	node.proposals[pid] = prop

	// create commit certificate with 1 vote (quorum==1)
	votes := []VoteMsg{makeSignedVote(t, pid, "2", VoteAccept, privB)}
	cert := makeCommitCertificate(&prop, votes)

	// applyCommit should succeed (playersPK contains actor pubkey and session has player)
	if err := node.applyCommit(cert); err != nil {
		t.Fatalf("applyCommit failed: %v", err)
	}
	// LastIndex should be incremented

	// Now test ban certificate validation and handling
	// create reject votes signed by player 2 and meet quorum=1
	rVotes := []VoteMsg{makeSignedVote(t, pid, "2", VoteReject, privB)}
	ban := makeBanCertificate(pid, "1", "test", rVotes)
	ok, err := node.validateBanCertificate(ban)
	if err != nil || !ok {
		t.Fatalf("validateBanCertificate failed: %v", err)
	}
	// handleBanCertificate should remove accused "1"
	if err := node.handleBanCertificate(ban); err != nil {
		t.Fatalf("handleBanCertificate failed: %v", err)
	}
	// ensure player "1" removed
	if idx := node.findPlayerIndex("1"); idx != -1 {
		t.Fatalf("expected player 1 removed, still found at %d", idx)
	}
}

// Negative tests for applyCommit error cases
func TestApplyCommitErrors(t *testing.T) {
	pub, priv := mustKeypair(t)
	playersPK := map[string]ed25519.PublicKey{"1": pub}
	node := &Node{ID: "n1", Pub: pub, Priv: priv, PlayersPK: playersPK, N: 1, quorum: 2, proposals: make(map[string]ProposalMsg), votes: make(map[string]map[string]VoteMsg)}
	setSessionPlayers(t, node, 1)

	// prepare a valid proposal
	a := &Action{RoundID: "r1", PlayerID: "1", Type: poker.ActionFold}
	err := a.Sign(priv)
	if err != nil {
		t.Fatalf("action sign failed: %v", err)
	}
	prop := makeProposalMsg(a, a.Signature)
	votes := []VoteMsg{} // none
	cert := makeCommitCertificate(&prop, votes)

	// not enough votes
	if err := node.applyCommit(cert); err == nil {
		t.Fatalf("expected error for not enough votes")
	}

	// unknown player in cert (remove pub)
	node.quorum = 1
	node.PlayersPK = map[string]ed25519.PublicKey{}
	votes = []VoteMsg{{}}
	cert = makeCommitCertificate(&prop, votes)
	if err := node.applyCommit(cert); err == nil {
		t.Fatalf("expected error for unknown player in cert")
	}

	// bad action signature
	node.PlayersPK = playersPK
	aBad := *a
	aBad.Signature = nil
	propBad := makeProposalMsg(&aBad, nil)
	cert = makeCommitCertificate(&propBad, []VoteMsg{{}})
	if err := node.applyCommit(cert); err == nil {
		t.Fatalf("expected error for bad action signature in cert")
	}

	// player not in session
	// change action player id to unknown
	aUnknown := *a
	aUnknown.PlayerID = "999"
	propUnknown := makeProposalMsg(&aUnknown, aUnknown.Signature)
	cert = makeCommitCertificate(&propUnknown, []VoteMsg{{}})
	if err := node.applyCommit(cert); err == nil {
		t.Fatalf("expected error for player not in session")
	}
}

// Full integration test: proposer sends proposal, followers receive, validate, vote, commit
func TestProposeReceive(t *testing.T) {
	// create listeners and peers
	listeners, addresses := common.CreateListeners(3)
	peers := make([]*common.Peer, 3)
	for i := 0; i < 3; i++ {
		p := common.NewPeer(i, addresses, listeners[i], 5*time.Second)
		peers[i] = &p
	}
	defer func() {
		for i := 0; i < 3; i++ {
			_ = peers[i].Close()
		}
	}()

	// generate keys and players map
	playersPK := make(map[string]ed25519.PublicKey)
	privs := make([]ed25519.PrivateKey, 3)
	ids := make([]string, 3)
	for i := 0; i < 3; i++ {
		pub, priv := mustKeypair(t)
		ids[i] = strconv.Itoa(peers[i].Rank)
		playersPK[ids[i]] = pub
		privs[i] = priv
	}

	// create nodes
	nodes := make([]*Node, 3)
	for i := 0; i < 3; i++ {
		nodes[i] = NewNode(peers[i], playersPK[ids[i]], privs[i], playersPK)
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
		PlayerID: strconv.Itoa(nodes[0].peer.Rank),
		Type:     poker.ActionBet,
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
		for i := 0; i < 3; i++ {
			_ = peers[i].Close()
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
			pkBytes,err := node.peer.AllToAll(b)
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
				Pots:        []poker.Pot{},
				HighestBet:  0,
				Dealer:      0,
				CurrentTurn: 0,
				RoundID:     "round-0",
			}
			node.Session = s
			nodes_chan <- node
			fatal <- nil
		}()
	}

	
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
		RoundID:  nodes[0].Session.RoundID,
		PlayerID: strconv.Itoa(nodes[0].peer.Rank),
		Type:     poker.ActionBet,
		Amount:   110, // too high, should trigger validation error
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
	if idx := nodes[0].findPlayerIndex(nodes[0].ID); idx != -1 {
		t.Fatalf("expected proposer to be banned, still found at index %d", idx)
	}
}
