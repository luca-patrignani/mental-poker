package ledger

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/luca-patrignani/mental-poker/consensus"
	"github.com/luca-patrignani/mental-poker/domain/poker"
	"github.com/luca-patrignani/mental-poker/network"
)

// createTestSession creates a realistic poker session with initialized peers.
// It returns the session, all P2P instances (needed to keep them alive), and any error.
// The caller is responsible for closing the P2P instances when done.
func createTestSession(n int) (poker.Session, []*network.P2P, error) {
	// Create board cards
	c1, err := poker.NewCard(poker.Diamond, 5)
	if err != nil {
		return poker.Session{}, nil, fmt.Errorf("failed to create card 1: %w", err)
	}
	c2, err := poker.NewCard(poker.Diamond, poker.King)
	if err != nil {
		return poker.Session{}, nil, fmt.Errorf("failed to create card 2: %w", err)
	}
	c3, err := poker.NewCard(poker.Heart, poker.Queen)
	if err != nil {
		return poker.Session{}, nil, fmt.Errorf("failed to create card 3: %w", err)
	}
	c4, err := poker.NewCard(poker.Heart, 4)
	if err != nil {
		return poker.Session{}, nil, fmt.Errorf("failed to create card 4: %w", err)
	}
	c5, err := poker.NewCard(poker.Spade, poker.King)
	if err != nil {
		return poker.Session{}, nil, fmt.Errorf("failed to create card 5: %w", err)
	}
	board := [5]poker.Card{c1, c2, c3, c4, c5}

	// Create listeners and addresses BEFORE spawning goroutines
	listeners, addresses := network.CreateListeners(n)

	errChan := make(chan error, n)
	playerChan := make(chan poker.Player, n)
	p2pChan := make(chan *network.P2P, n)

	// Spawn goroutines for each player
	for i := 0; i < n; i++ {
		go func(index int) {
			// Create peer with the correct index
			peer := network.NewPeer(index, addresses, listeners[index], 30*time.Second)

			// Create P2P adapter
			p2p := network.NewP2P(&peer)

			// Create player struct
			player := poker.Player{
				Name:      "Player" + fmt.Sprintf("%d", index),
				Id:        index,
				Hand:      [2]poker.Card{}, // Empty until cards are drawn
				Pot:       1000,
				Bet:       0,
				HasFolded: false,
			}

			// Send results back
			errChan <- nil
			playerChan <- player
			p2pChan <- p2p
		}(i) // CRITICAL FIX 2: Pass i as argument
	}

	// Collect all errors
	for i := 0; i < n; i++ {
		err := <-errChan
		if err != nil {
			return poker.Session{}, nil, err
		}
	}

	// Collect all players and P2P instances
	var players []poker.Player
	var p2ps []*network.P2P

	for i := 0; i < n; i++ {
		players = append(players, <-playerChan)
		p2ps = append(p2ps, <-p2pChan)
	}

	// Sort players by ID for consistent ordering
	sort.Slice(players, func(i, j int) bool {
		return players[i].Id < players[j].Id
	})

	// Initialize eligible players for pots
	eligiblePlayers := make([]int, n)
	for i := 0; i < n; i++ {
		eligiblePlayers[i] = i
	}

	// Create the session
	session := poker.Session{
		Board:       board,
		Players:     players,
		Pots:        []poker.Pot{{Amount: 0, Eligible: eligiblePlayers}},
		HighestBet:  0,
		Dealer:      0,
		CurrentTurn: 0,
		Round:     "preflop-1",
	}

	return session, p2ps, nil
}

// cleanupP2PInstances closes all P2P instances and their underlying peers.
// Call this with defer after creating a session.
func cleanupP2PInstances(p2ps []*network.P2P) error {
	var lastErr error
	for i := len(p2ps) - 1; i >= 0; i-- { // Close in reverse order
		if err := p2ps[i].Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// TestNewBlockchainWithInitialSession verifies that a new blockchain is correctly initialized
// with a genesis block containing the provided initial session. This ensures the blockchain
// starts with the correct game state.
func TestNewBlockchainWithInitialSession(t *testing.T) {
	n := 5
	initialSession, p2ps, err := createTestSession(n)
	defer func() {
		err := cleanupP2PInstances(p2ps)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()

	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	if err != nil {
		t.Fatalf("failed to prepare deck: %v", err)
	}

	bc, err := NewBlockchain(initialSession)
	if err != nil {
		t.Fatalf("failed to create blockchain : %v", err)
	}
	if bc == nil {
		t.Fatal("NewBlockchain returned nil")
	}

	if len(bc.blocks) != 1 {
		t.Fatalf("expected 1 block (genesis), got %d", len(bc.blocks))
	}

	genesis := bc.blocks[0]
	if genesis.Index != 0 {
		t.Fatalf("genesis index should be 0, got %d", genesis.Index)
	}

	if genesis.PrevHash != "0" {
		t.Fatalf("genesis PrevHash should be '0', got %s", genesis.PrevHash)
	}

	if genesis.Action.Type != "genesis" {
		t.Fatalf("genesis action type should be 'genesis', got %s", genesis.Action.Type)
	}

	if len(genesis.Votes) != 0 {
		t.Fatalf("genesis should have no votes, got %d", len(genesis.Votes))
	}

	if genesis.Hash == "" {
		t.Fatal("genesis block should have a hash")
	}

	// Verify the session is correctly stored in the genesis block
	if genesis.Session.Round != initialSession.Round {
		t.Fatalf("genesis session RoundID should be '%s', got '%s'", initialSession.Round, genesis.Session.Round)
	}

	if len(genesis.Session.Players) != len(initialSession.Players) {
		t.Fatalf("genesis session should have %d players, got %d", len(initialSession.Players), len(genesis.Session.Players))
	}

	if genesis.Session.Dealer != initialSession.Dealer {
		t.Fatalf("genesis session Dealer should be %d, got %d", initialSession.Dealer, genesis.Session.Dealer)
	}

	if genesis.Session.HighestBet != initialSession.HighestBet {
		t.Fatalf("genesis session HighestBet should be %d, got %d", initialSession.HighestBet, genesis.Session.HighestBet)
	}
}

// TestNewBlockchainWithDifferentSessions verifies that different initial sessions produce
// different genesis block hashes. This ensures each blockchain instance is unique to its game.
func TestNewBlockchainWithDifferentSessions(t *testing.T) {
	n := 5
	session1, p2ps1, err := createTestSession(n)
	if err != nil {
		t.Fatalf("failed to create test sessions 1: %v", err)
	}
	defer func() {
		err := cleanupP2PInstances(p2ps1)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()
	session2, p2ps2, err := createTestSession(n)
	if err != nil {
		t.Fatalf("failed to create test sessions 2: %v", err)
	}
	defer func() {
		err := cleanupP2PInstances(p2ps2)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()
	session2.Round = "different-round"

	bc1, err := NewBlockchain(session1)
	if err != nil {
		t.Fatalf("failed to create blockchain 1: %v", err)
	}
	bc2, err := NewBlockchain(session2)
	if err != nil {
		t.Fatalf("failed to create blockchain 2: %v", err)
	}
	if bc1.blocks[0].Hash == bc2.blocks[0].Hash {
		t.Fatal("different sessions should produce different genesis block hashes")
	}
}

// TestAppendValidBlock verifies that a valid block can be successfully appended to the chain.
// This test ensures the core append functionality works with valid data and maintains chain integrity.
func TestAppendValidBlock(t *testing.T) {
	n := 5
	initialSession, p2ps, err := createTestSession(n)
	defer func() {
		err := cleanupP2PInstances(p2ps)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	bc, err := NewBlockchain(initialSession)
	if err != nil {
		t.Fatalf("failed to create blockchain : %v", err)
	}
	session := initialSession
	session.CurrentTurn = 1
	action := poker.PokerAction{
		Round:  "round1",
		PlayerID: 1,
		Type:     poker.ActionBet,
		Amount:   50,
	}
	votes := []consensus.Vote{
		{ActionId: "action1", VoterID: 0, Value: consensus.VoteAccept},
		{ActionId: "action1", VoterID: 1, Value: consensus.VoteAccept},
	}

	err = bc.Append(session, action, votes, 1, 2)
	if err != nil {
		t.Fatalf("unexpected error appending valid block: %v", err)
	}

	if len(bc.blocks) != 2 {
		t.Fatalf("expected 2 blocks after append, got %d", len(bc.blocks))
	}

	newBlock := bc.blocks[1]
	if newBlock.Index != 1 {
		t.Fatalf("new block index should be 1, got %d", newBlock.Index)
	}

	if newBlock.PrevHash != bc.blocks[0].Hash {
		t.Fatal("new block's PrevHash should match previous block's hash")
	}

	if newBlock.Action.Amount != 50 {
		t.Fatalf("block action amount should be 50, got %d", newBlock.Action.Amount)
	}

	if len(newBlock.Votes) != 2 {
		t.Fatalf("block should have 2 votes, got %d", len(newBlock.Votes))
	}
}

// TestAppendBlockInsufficientVotes verifies that a block with fewer votes than the quorum
// requirement is rejected. This test ensures the consensus validation mechanism prevents
// invalid blocks from entering the chain.
func TestAppendBlockInsufficientVotes(t *testing.T) {
	n := 5
	initialSession, p2ps, err := createTestSession(n)
	defer func() {
		err := cleanupP2PInstances(p2ps)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	bc, err := NewBlockchain(initialSession)
	if err != nil {
		t.Fatalf("failed to create blockchain : %v", err)
	}
	session := initialSession
	session.CurrentTurn = 1
	action := poker.PokerAction{
		Round:  "round1",
		PlayerID: 1,
		Type:     poker.ActionBet,
		Amount:   50,
	}
	votes := []consensus.Vote{
		{ActionId: "action1", VoterID: 0, Value: consensus.VoteAccept},
	}

	// Try to append with quorum of 2 but only 1 vote
	err = bc.Append(session, action, votes, 0, 2)
	if err == nil {
		t.Fatal("expected error for insufficient votes, got nil")
	}

	if len(bc.blocks) != 1 {
		t.Fatalf("blockchain should still have 1 block, got %d", len(bc.blocks))
	}
}

// TestAppendWithExtraMetadata verifies that the extra metadata passed to Append is correctly
// stored in the block. This test ensures that optional contextual information (like ban reasons)
// is preserved in the ledger.
func TestAppendWithExtraMetadata(t *testing.T) {
	n := 5
	initialSession, p2ps, err := createTestSession(n)
	defer func() {
		err := cleanupP2PInstances(p2ps)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	bc, err := NewBlockchain(initialSession)
	if err != nil {
		t.Fatalf("failed to create blockchain : %v", err)
	}
	session := initialSession
	session.CurrentTurn = 1
	action := poker.PokerAction{
		Round:  "round1",
		PlayerID: 1,
		Type:     poker.ActionBet,
		Amount:   50,
	}
	votes := []consensus.Vote{
		{ActionId: "action1", VoterID: 0, Value: consensus.VoteAccept},
		{ActionId: "action1", VoterID: 1, Value: consensus.VoteAccept},
	}
	extraData := map[string]string{
		"reason": "player-disconnected",
		"info":   "unexpected-timeout",
	}

	err = bc.Append(session, action, votes, 1, 2, extraData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newBlock := bc.blocks[1]
	if newBlock.Metadata.Extra["reason"] != "player-disconnected" {
		t.Fatalf("expected extra reason 'player-disconnected', got %s", newBlock.Metadata.Extra["reason"])
	}

	if newBlock.Metadata.ProposerID != 1 {
		t.Fatalf("expected proposer ID 1, got %d", newBlock.Metadata.ProposerID)
	}
}

// TestGetLatestBlock verifies that GetLatest returns the most recent block in the chain.
// This test ensures the method correctly identifies the tail of the blockchain.
func TestGetLatestBlock(t *testing.T) {
	n := 5
	initialSession, p2ps, err := createTestSession(n)
	defer func() {
		err := cleanupP2PInstances(p2ps)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	bc, err := NewBlockchain(initialSession)
	if err != nil {
		t.Fatalf("failed to create blockchain : %v", err)
	}
	session := initialSession
	session.CurrentTurn = 1
	action := poker.PokerAction{Round: "r1", Type: poker.ActionBet, Amount: 50}
	votes := []consensus.Vote{
		{ActionId: "a1", VoterID: 0, Value: consensus.VoteAccept},
		{ActionId: "a1", VoterID: 1, Value: consensus.VoteAccept},
	}

	err = bc.Append(session, action, votes, 0, 2)

	if err != nil {
		t.Fatalf("unexpected error appending block: %v", err)
	}

	latest, err := bc.GetLatest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if latest.Index != 1 {
		t.Fatalf("latest block index should be 1, got %d", latest.Index)
	}

	if latest.Action.Amount != 50 {
		t.Fatalf("latest block action amount should be 50, got %d", latest.Action.Amount)
	}
}

// TestGetLatestEmptyBlockchain verifies that GetLatest returns an error when called on an
// empty blockchain. While a new blockchain always has a genesis block, this test protects
// against edge cases in concurrent scenarios.
func TestGetLatestEmptyBlockchain(t *testing.T) {
	bc := &Blockchain{blocks: []Block{}}

	_, err := bc.GetLatest()
	if err == nil {
		t.Fatal("expected error for empty blockchain, got nil")
	}
}

// TestGetByIndexValid verifies that GetByIndex correctly retrieves blocks by their index
// and maintains their integrity. This test ensures the block retrieval mechanism is reliable.
func TestGetByIndexValid(t *testing.T) {
	n := 5
	initialSession, p2ps, err := createTestSession(n)
	defer func() {
		err := cleanupP2PInstances(p2ps)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	bc, err := NewBlockchain(initialSession)
	if err != nil {
		t.Fatalf("failed to create blockchain : %v", err)
	}
	session := initialSession
	session.CurrentTurn = 1
	action := poker.PokerAction{Round: "r1", Type: poker.ActionBet, Amount: 50}
	votes := []consensus.Vote{
		{ActionId: "a1", VoterID: 0, Value: consensus.VoteAccept},
		{ActionId: "a1", VoterID: 1, Value: consensus.VoteAccept},
	}

	err = bc.Append(session, action, votes, 0, 2)
	if err != nil {
		t.Fatalf("unexpected error appending block: %v", err)
	}

	block, err := bc.GetByIndex(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if block.Index != 1 {
		t.Fatalf("expected block index 1, got %d", block.Index)
	}

	// Verify genesis block
	genesisBlock, err := bc.GetByIndex(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if genesisBlock.Action.Type != "genesis" {
		t.Fatal("genesis block should have type 'genesis'")
	}

	if genesisBlock.Session.Round != initialSession.Round {
		t.Fatalf("genesis block should preserve initial session RoundID")
	}
}

// TestGetByIndexOutOfRange verifies that GetByIndex returns an error for invalid indices.
// This test ensures proper boundary checking and prevents panics.
func TestGetByIndexOutOfRange(t *testing.T) {
	n := 5
	initialSession, p2ps, err := createTestSession(n)
	defer func() {
		err := cleanupP2PInstances(p2ps)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	bc, err := NewBlockchain(initialSession)
	if err != nil {
		t.Fatalf("failed to create blockchain : %v", err)
	}

	_, err = bc.GetByIndex(10)
	if err == nil {
		t.Fatal("expected error for out of range index, got nil")
	}

	_, err = bc.GetByIndex(-1)
	if err == nil {
		t.Fatal("expected error for negative index, got nil")
	}
}

// TestVerifyValidChain verifies that a valid blockchain passes all integrity checks.
// This test ensures the verification algorithm correctly validates chains.
func TestVerifyValidChain(t *testing.T) {
	n := 5
	initialSession, p2ps, err := createTestSession(n)
	defer func() {
		err := cleanupP2PInstances(p2ps)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	bc, err := NewBlockchain(initialSession)
	if err != nil {
		t.Fatalf("failed to create blockchain : %v", err)
	}
	session := initialSession
	session.CurrentTurn = 1

	// Add multiple blocks
	for i := 0; i < 3; i++ {
		action := poker.PokerAction{
			Round:  "round1",
			PlayerID: i,
			Type:     poker.ActionBet,
			Amount:   uint(50 + i*10),
		}
		votes := []consensus.Vote{
			{ActionId: "a" + string(rune(i)), VoterID: 0, Value: consensus.VoteAccept},
			{ActionId: "a" + string(rune(i)), VoterID: 1, Value: consensus.VoteAccept},
		}
		err = bc.Append(session, action, votes, i, 2)
		if err != nil {
			t.Fatalf("unexpected error appending block: %v", err)
		}
	}

	err = bc.Verify()
	if err != nil {
		t.Fatalf("valid blockchain verification failed: %v", err)
	}
}

// TestVerifyEmptyBlockchain verifies that verification fails on an empty blockchain.
// This protects against degenerate states.
func TestVerifyEmptyBlockchain(t *testing.T) {
	bc := &Blockchain{blocks: []Block{}}

	err := bc.Verify()
	if err == nil {
		t.Fatal("expected error for empty blockchain verification, got nil")
	}
}

// TestVerifyInvalidGenesis verifies that verification fails if the genesis block has an
// incorrect previous hash. This test ensures the root of trust is protected.
func TestVerifyInvalidGenesis(t *testing.T) {
	n := 5
	initialSession, p2ps, err := createTestSession(n)
	defer func() {
		err := cleanupP2PInstances(p2ps)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	bc, err := NewBlockchain(initialSession)
	if err != nil {
		t.Fatalf("failed to create blockchain : %v", err)
	}
	bc.blocks[0].PrevHash = "invalid"

	err = bc.Verify()
	if err == nil {
		t.Fatal("expected error for invalid genesis block, got nil")
	}
}

// TestVerifyTamperedBlockHash verifies that the verification detects when a block's hash
// has been tampered with. This test ensures cryptographic integrity is maintained.
func TestVerifyTamperedBlockHash(t *testing.T) {
	n := 5
	initialSession, p2ps, err := createTestSession(n)
	defer func() {
		err := cleanupP2PInstances(p2ps)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	bc, err := NewBlockchain(initialSession)
	if err != nil {
		t.Fatalf("failed to create blockchain : %v", err)
	}
	session := initialSession
	session.CurrentTurn = 1
	action := poker.PokerAction{Round: "r1", Type: poker.ActionBet, Amount: 50}
	votes := []consensus.Vote{
		{ActionId: "a1", VoterID: 0, Value: consensus.VoteAccept},
		{ActionId: "a1", VoterID: 1, Value: consensus.VoteAccept},
	}

	err = bc.Append(session, action, votes, 0, 2)
	if err != nil {
		t.Fatalf("unexpected error appending block: %v", err)
	}

	// Tamper with the block hash
	bc.blocks[1].Hash = "tamperedhash"

	err = bc.Verify()
	if err == nil {
		t.Fatal("expected error for tampered block hash, got nil")
	}
}

// TestVerifyBrokenChainLink verifies that verification detects when the previous hash
// link is broken. This test ensures chain continuity validation works.
func TestVerifyBrokenChainLink(t *testing.T) {
	n := 5
	initialSession, p2ps, err := createTestSession(n)
	defer func() {
		err := cleanupP2PInstances(p2ps)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	bc, err := NewBlockchain(initialSession)
	if err != nil {
		t.Fatalf("failed to create blockchain : %v", err)
	}
	session := initialSession
	session.CurrentTurn = 1
	action := poker.PokerAction{Round: "r1", Type: poker.ActionBet, Amount: 50}
	votes := []consensus.Vote{
		{ActionId: "a1", VoterID: 0, Value: consensus.VoteAccept},
		{ActionId: "a1", VoterID: 1, Value: consensus.VoteAccept},
	}

	err = bc.Append(session, action, votes, 0, 2)
	if err != nil {
		t.Fatalf("unexpected error appending block: %v", err)
	}

	err = bc.Append(session, action, votes, 0, 2)
	if err != nil {
		t.Fatalf("unexpected error appending block: %v", err)
	}

	// Break the chain link
	bc.blocks[1].PrevHash = "wronghash"

	err = bc.Verify()
	if err == nil {
		t.Fatal("expected error for broken chain link, got nil")
	}
}

// TestVerifyIndexDiscontinuity verifies that verification detects when block indices
// are not sequential. This test ensures the block order is maintained.
func TestVerifyIndexDiscontinuity(t *testing.T) {
	n := 5
	initialSession, p2ps, err := createTestSession(n)
	defer func() {
		err := cleanupP2PInstances(p2ps)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	bc, err := NewBlockchain(initialSession)
	if err != nil {
		t.Fatalf("failed to create blockchain : %v", err)
	}
	session := initialSession
	session.CurrentTurn = 1
	action := poker.PokerAction{Round: "r1", Type: poker.ActionBet, Amount: 50}
	votes := []consensus.Vote{
		{ActionId: "a1", VoterID: 0, Value: consensus.VoteAccept},
		{ActionId: "a1", VoterID: 1, Value: consensus.VoteAccept},
	}

	err = bc.Append(session, action, votes, 0, 2)
	if err != nil {
		t.Fatalf("unexpected error appending block: %v", err)
	}

	// Tamper with the block index
	bc.blocks[1].Index = 5

	err = bc.Verify()
	if err == nil {
		t.Fatal("expected error for index discontinuity, got nil")
	}
}

// TestAppendMultipleBlocks verifies that multiple blocks can be appended sequentially
// and maintain chain integrity throughout. This is a practical integration test.
func TestAppendMultipleBlocks(t *testing.T) {
	n := 5
	initialSession, p2ps, err := createTestSession(n)
	defer func() {
		err := cleanupP2PInstances(p2ps)
		if err != nil {
			t.Fatalf("failed to cleanup P2P instances: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	bc, err := NewBlockchain(initialSession)
	if err != nil {
		t.Fatalf("failed to create blockchain : %v", err)
	}
	session := initialSession
	session.CurrentTurn = 1

	for i := 0; i < 5; i++ {
		action := poker.PokerAction{
			Round:  "round1",
			PlayerID: i % 2,
			Type:     poker.ActionBet,
			Amount:   uint(50 + i*10),
		}
		votes := []consensus.Vote{
			{ActionId: "a" + string(rune(i)), VoterID: 0, Value: consensus.VoteAccept},
			{ActionId: "a" + string(rune(i)), VoterID: 1, Value: consensus.VoteAccept},
		}

		err := bc.Append(session, action, votes, i%2, 2)
		if err != nil {
			t.Fatalf("unexpected error at block %d: %v", i, err)
		}
	}

	if len(bc.blocks) != 6 { // 1 genesis + 5 appended
		t.Fatalf("expected 6 blocks, got %d", len(bc.blocks))
	}

	// Verify the entire chain
	err = bc.Verify()
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}
}
