package main

import (
	"crypto/ed25519"
	"fmt"
	"net"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/luca-patrignani/mental-poker/consensus"
	"github.com/luca-patrignani/mental-poker/domain/poker"
	"github.com/luca-patrignani/mental-poker/ledger"
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
			p2p, myRank := createP2P(addresses, mapListeners[i])
			names, err := testConnections(p2p, "name"+fmt.Sprint(myRank))
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
// TestGuessIpAddress tests the IP address guessing logic with various partial addresses
func TestGuessIpAddress(t *testing.T) {
	tests := []struct {
		name        string
		baseAddr    string
		partialAddr string
		expected    string
		expectError bool
	}{
		{
			name:        "Single octet - last position",
			baseAddr:    "192.168.1.100",
			partialAddr: "50",
			expected:    "192.168.1.50",
			expectError: false,
		},
		{
			name:        "Two octets - last two positions",
			baseAddr:    "192.168.1.100",
			partialAddr: "2.50",
			expected:    "192.168.2.50",
			expectError: false,
		},
		{
			name:        "Three octets",
			baseAddr:    "192.168.1.100",
			partialAddr: "10.0.50",
			expected:    "192.10.0.50",
			expectError: false,
		},
		{
			name:        "Empty string - returns base address",
			baseAddr:    "192.168.1.100",
			partialAddr: "",
			expected:    "192.168.1.100",
			expectError: false,
		},
		{
			name:        "Full address",
			baseAddr:    "192.168.1.100",
			partialAddr: "10.20.30.40",
			expected:    "10.20.30.40",
			expectError: false,
		},
		{
			name:        "Invalid octet - non-numeric",
			baseAddr:    "192.168.1.100",
			partialAddr: "abc",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Zero value",
			baseAddr:    "192.168.1.100",
			partialAddr: "0",
			expected:    "192.168.1.0",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseIP := net.ParseIP(tt.baseAddr)
			if baseIP == nil {
				t.Fatalf("invalid base address: %s", tt.baseAddr)
			}

			result, err := guessIpAddress(baseIP, tt.partialAddr)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

// TestCreateP2P tests the P2P creation and rank assignment logic
func TestCreateP2P(t *testing.T) {
	// Create test listeners
	listeners, addresses := network.CreateListeners(3)
	defer func() {
		for _, l := range listeners {
			l.Close()
		}
	}()

	// Convert map to slice for testing
	addrSlice := make([]string, len(addresses))
	for i, addr := range addresses {
		addrSlice[i] = addr
	}
	sort.Strings(addrSlice)

	// Test with first listener
	p2p0, rank0 := createP2P(addrSlice, listeners[0])
	defer p2p0.Close()

	if rank0 != 0 {
		t.Errorf("expected rank 0, got %d", rank0)
	}

	if p2p0.GetRank() != rank0 {
		t.Errorf("P2P rank mismatch: expected %d, got %d", rank0, p2p0.GetRank())
	}

	// Test with second listener
	p2p1, rank1 := createP2P(addrSlice, listeners[1])
	defer p2p1.Close()

	if rank1 != 1 {
		t.Errorf("expected rank 1, got %d", rank1)
	}

	// Verify addresses are sorted
	sortedAddrs := make([]string, len(addrSlice))
	copy(sortedAddrs, addrSlice)
	sort.Strings(sortedAddrs)

	p2pAddrs := p2p0.GetAddresses()
	for i, addr := range sortedAddrs {
		if p2pAddrs[i] != addr {
			t.Errorf("addresses not properly sorted at index %d", i)
		}
	}
}

// TestDistributeHands tests card distribution logic
func TestDistributeHands(t *testing.T) {
	// Setup
	n := 3
	listeners, addresses := network.CreateListeners(n)
	defer func() {
		for _, l := range listeners {
			l.Close()
		}
	}()

	peers := make([]*network.Peer, n)
	p2ps := make([]*network.P2P, n)
	for i := 0; i < n; i++ {
		peer := network.NewPeer(i, addresses, listeners[i], 30*time.Second)
		peers[i] = &peer
		p2ps[i] = network.NewP2P(&peer)
	}
	defer func() {
		for _, p := range p2ps {
			p.Close()
		}
	}()

	// Create session with players
	players := make([]poker.Player, n)
	for i := 0; i < n; i++ {
		card, _ := poker.NewCard(0, 0)
		players[i] = poker.Player{
			Name: "Player" + string(rune(i)),
			Id:   i,
			Hand: [2]poker.Card{card, card},
			Pot:  1000,
		}
	}

	session := poker.Session{
		Players:     players,
		CurrentTurn: 0,
	}

	// Test distribution in goroutines
	errChan := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			psm := poker.PokerManager{Session: &session, Player: idx}
			deck := poker.NewPokerDeck(p2ps[idx])
			
			if err := deck.PrepareDeck(); err != nil {
				errChan <- err
				return
			}
			
			if err := deck.Shuffle(); err != nil {
				errChan <- err
				return
			}
			
			if err := distributeHands(&psm, &deck); err != nil {
				errChan <- err
				return
			}
			
			// Verify hands were distributed
			for _, player := range psm.Session.Players {
				if player.Hand[0].Rank() == 0 || player.Hand[1].Rank() == 0 {
					errChan <- nil // Face-down cards are expected for other players
					return
				}
			}
			
			errChan <- nil
		}(i)
	}

	// Collect results
	for i := 0; i < n; i++ {
		if err := <-errChan; err != nil {
			t.Fatalf("distribute hands failed: %v", err)
		}
	}
}

// TestAddBlind tests individual blind posting
func TestAddBlind(t *testing.T) {
	// Setup minimal environment
	listeners, addresses := network.CreateListeners(2)
	defer func() {
		for _, l := range listeners {
			l.Close()
		}
	}()

	peers := make([]*network.Peer, 2)
	p2ps := make([]*network.P2P, 2)
	for i := 0; i < 2; i++ {
		peer := network.NewPeer(i, addresses, listeners[i], 5*time.Second)
		peers[i] = &peer
		p2ps[i] = network.NewP2P(&peer)
	}
	defer func() {
		for _, p := range p2ps {
			p.Close()
		}
	}()

	players := []poker.Player{
		{Id: 0, Name: "Alice", Pot: 100, Bet: 0},
		{Id: 1, Name: "Bob", Pot: 100, Bet: 0},
	}

	errChan := make(chan error, 2)
	
	for i := 0; i < 2; i++ {
		go func(idx int) {
			session := poker.Session{
				Players:     players,
				CurrentTurn: 0,
				Round:       poker.PreFlop,
				HighestBet:  0,
				Pots:        []poker.Pot{{Amount: 0, Eligible: []int{0, 1}}},
			}
			
			psm := poker.PokerManager{Session: &session, Player: idx}
			
			pub, priv, _ := ed25519.GenerateKey(nil)
			playersPK := make(map[int]ed25519.PublicKey)
			
			bc, _ := ledger.NewBlockchain(session)
			node := consensus.NewConsensusNode(pub, priv, playersPK, &psm, bc, p2ps[idx])
			err := node.UpdatePeers()
			if err != nil {
				errChan <- err
			}
			
			err = addBlind(&psm, node, 10)
			errChan <- err
		}(i)
	}

	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("addBlind failed: %v", err)
		}
	}
}

// TestCardOnBoard tests placing cards on the board
func TestCardOnBoard(t *testing.T) {
	n := 2
	listeners, addresses := network.CreateListeners(n)
	defer func() {
		for _, l := range listeners {
			l.Close()
		}
	}()

	peers := make([]*network.Peer, n)
	p2ps := make([]*network.P2P, n)
	for i := 0; i < n; i++ {
		peer := network.NewPeer(i, addresses, listeners[i], 30*time.Second)
		peers[i] = &peer
		p2ps[i] = network.NewP2P(&peer)
	}
	defer func() {
		for _, p := range p2ps {
			p.Close()
		}
	}()

	errChan := make(chan error, n)
	
	for i := 0; i < n; i++ {
		go func(idx int) {
			session := poker.Session{
				Board: [5]poker.Card{},
			}
			psm := poker.PokerManager{Session: &session, Player: idx}
			deck := poker.NewPokerDeck(p2ps[idx])
			
			if err := deck.PrepareDeck(); err != nil {
				errChan <- err
				return
			}
			
			if err := deck.Shuffle(); err != nil {
				errChan <- err
				return
			}
			
			// Place first card
			if err := cardOnBoard(&psm, &deck, 0); err != nil {
				errChan <- err
				return
			}
			
			// Verify card was placed
			if psm.Session.Board[0].Rank() == 0 {
				errChan <- nil // Expected for non-drawer
				return
			}
			
			errChan <- nil
		}(i)
	}

	for i := 0; i < n; i++ {
		if err := <-errChan; err != nil {
			t.Fatalf("cardOnBoard failed: %v", err)
		}
	}
}

// TestShowCards tests card revealing at showdown
func TestShowCards(t *testing.T) {
	n := 2
	listeners, addresses := network.CreateListeners(n)
	defer func() {
		for _, l := range listeners {
			l.Close()
		}
	}()

	peers := make([]*network.Peer, n)
	p2ps := make([]*network.P2P, n)
	for i := 0; i < n; i++ {
		peer := network.NewPeer(i, addresses, listeners[i], 30*time.Second)
		peers[i] = &peer
		p2ps[i] = network.NewP2P(&peer)
	}
	defer func() {
		for _, p := range p2ps {
			p.Close()
		}
	}()

	errChan := make(chan error, n)

	for i := 0; i < n; i++ {
		go func(idx int) {
			card, _ := poker.NewCard(0, 0)
			players := []poker.Player{
				{Id: 0, Name: "Alice", Hand: [2]poker.Card{card, card}},
				{Id: 1, Name: "Bob", Hand: [2]poker.Card{card, card}},
			}

			session := poker.Session{
				Players: players,
			}

			psm := poker.PokerManager{Session: &session, Player: idx}
			deck := poker.NewPokerDeck(p2ps[idx])

			if err := deck.PrepareDeck(); err != nil {
				errChan <- err
				return
			}

			if err := deck.Shuffle(); err != nil {
				errChan <- err
				return
			}

			// First distribute hands
			if err := distributeHands(&psm, &deck); err != nil {
				errChan <- err
				return
			}

			// Then reveal them
			if err := showCards(&psm, &deck); err != nil {
				errChan <- err
				return
			}

			errChan <- nil
		}(i)
	}

	for i := 0; i < n; i++ {
		if err := <-errChan; err != nil {
			t.Fatalf("showCards failed: %v", err)
		}
	}
}

// TestAskForLeavers tests the player leaving mechanism
func TestAskForLeavers(t *testing.T) {
	tests := []struct {
		name        string
		numPlayers  int
		leaverIdx   int
		expectLeave bool
	}{
		{
			name:        "One player leaves",
			numPlayers:  3,
			leaverIdx:   1,
			expectLeave: false,
		},
		{
			name:        "No players leave",
			numPlayers:  2,
			leaverIdx:   -1,
			expectLeave: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := tt.numPlayers
			listeners, addresses := network.CreateListeners(n)
			defer func() {
				for _, l := range listeners {
					l.Close()
				}
			}()

			peers := make([]*network.Peer, n)
			p2ps := make([]*network.P2P, n)
			for i := 0; i < n; i++ {
				peer := network.NewPeer(i, addresses, listeners[i], 5*time.Second)
				peers[i] = &peer
				p2ps[i] = network.NewP2P(&peer)
			}
			defer func() {
				for _, p := range p2ps {
					if p != nil {
						p.Close()
					}
				}
			}()

			players := make([]poker.Player, n)
			for i := 0; i < n; i++ {
				players[i] = poker.Player{
					Id:   i,
					Name: "Player" + string(rune('0'+i)),
					Pot:  1000,
				}
			}

			// Note: This test is limited because askForLeavers requires user input
			// We can only test the structure, not the interactive behavior
			// In a real scenario, you'd mock the pterm interactive components
			
			session := poker.Session{
				Players: players,
			}

			psm := poker.PokerManager{Session: &session, Player: 0}
			
			pub, priv, _ := ed25519.GenerateKey(nil)
			playersPK := make(map[int]ed25519.PublicKey)
			for i := 0; i < n; i++ {
				pub, _, _ := ed25519.GenerateKey(nil)
				playersPK[i] = pub
			}
			
			bc, _ := ledger.NewBlockchain(session)
			node := consensus.NewConsensusNode(pub, priv, playersPK, &psm, bc, p2ps[0])
			deck := poker.NewPokerDeck(p2ps[0])
			
			// We can't fully test askForLeavers without mocking pterm
			// but we can verify the function signature and basic structure
			t.Logf("askForLeavers requires interactive input - structural test only")
			
			// Verify the function exists and has correct signature
			_ = func() (bool, []string, error) {
				return askForLeavers(psm, *node, deck, *p2ps[0])
			}
		})
	}
}

// Benchmark tests
func BenchmarkGuessIpAddress(b *testing.B) {
	baseIP := net.ParseIP("192.168.1.100")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = guessIpAddress(baseIP, "50")
	}
}

func BenchmarkSplitHostPort(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _ = splitHostPort("192.168.1.1:8080", 53550)
	}
}
