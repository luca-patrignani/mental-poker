package poker

import "testing"



func TestManager_ValidateWrongRound(t *testing.T) {
	session := &Session{
		RoundID: "round1",
		Players: []Player{{Id: 1, Name: "Alice"}},
		CurrentTurn: 0,
	}
	
	sm := NewPokerManager(session)
	
	action := &PokerAction{
		RoundID:  "round2",
		PlayerID: 1,
		Type:     ActionCheck,
		Amount:   0,
	}
	
	payload, _ := action.ToConsensusPayload()
	err := sm.Validate(payload)
	
	if err == nil {
		t.Fatal("expected error for wrong round")
	}
}

func TestManager_ValidatePlayerNotInSession(t *testing.T) {
	session := &Session{
		RoundID: "round1",
		Players: []Player{{Id: 1, Name: "Alice"}},
		CurrentTurn: 0,
	}
	
	sm := NewPokerManager(session)
	
	action := &PokerAction{
		RoundID:  "round1",
		PlayerID: 999,
		Type:     ActionCheck,
		Amount:   0,
	}
	
	payload, _ := action.ToConsensusPayload()
	err := sm.Validate(payload)
	
	if err == nil {
		t.Fatal("expected error for player not in session")
	}
}

func TestManager_ValidateWrongTurn(t *testing.T) {
	session := &Session{
		RoundID: "round1",
		Players: []Player{
			{Id: 1, Name: "Alice"},
			{Id: 2, Name: "Bob"},
		},
		CurrentTurn: 0,
	}
	
	sm := NewPokerManager(session)
	
	action := &PokerAction{
		RoundID:  "round1",
		PlayerID: 2, // Bob trying to act when it's Alice's turn
		Type:     ActionCheck,
		Amount:   0,
	}
	
	payload, _ := action.ToConsensusPayload()
	err := sm.Validate(payload)
	
	if err == nil {
		t.Fatal("expected error for wrong turn")
	}
}

func TestManager_ApplyValidAction(t *testing.T) {
	session := &Session{
		RoundID: "round1",
		Players: []Player{
			{Id: 1, Name: "Alice", Pot: 100, Bet: 0},
		},
		CurrentTurn: 0,
		HighestBet:  0,
	}
	
	sm := NewPokerManager(session)
	
	action := &PokerAction{
		RoundID:  "round1",
		PlayerID: 1,
		Type:     ActionBet,
		Amount:   50,
	}
	
	payload, _ := action.ToConsensusPayload()
	err := sm.Apply(payload)
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if session.Players[0].Bet != 50 {
		t.Fatalf("expected bet 50, got %d", session.Players[0].Bet)
	}
}

func TestManager_GetCurrentPlayer(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Id: 100, Name: "Alice"},
			{Id: 200, Name: "Bob"},
		},
		CurrentTurn: 1,
	}
	
	sm := NewPokerManager(session)
	
	currentPlayer := sm.GetCurrentPlayer()
	if currentPlayer != 200 {
		t.Fatalf("expected player 200, got %d", currentPlayer)
	}
}

func TestManager_FindPlayerIndex(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Id: 100, Name: "Alice"},
			{Id: 200, Name: "Bob"},
			{Id: 300, Name: "Carol"},
		},
	}
	
	sm := NewPokerManager(session)
	
	idx := sm.FindPlayerIndex(200)
	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}
	
	idx = sm.FindPlayerIndex(999)
	if idx != -1 {
		t.Fatalf("expected -1 for non-existent player, got %d", idx)
	}
}

func TestManager_SnapshotAndRestore(t *testing.T) {
	session := &Session{
		RoundID: "round1",
		Players: []Player{{Id: 1, Name: "Alice", Pot: 100}},
		HighestBet: 50,
	}
	
	sm := NewPokerManager(session)
	
	snapshot, err := sm.Snapshot()
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	
	// Modify session
	session.HighestBet = 200
	
	// Restore
	newSession := &Session{}
	newSm := NewPokerManager(newSession)
	err = newSm.Restore(snapshot)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	
	if newSm.session.HighestBet != 50 {
		t.Fatalf("expected restored highest bet 50, got %d", newSm.session.HighestBet)
	}
}

func TestManager_NotifyBan(t *testing.T) {
	session := &Session{
		RoundID: "round1",
	}
	
	sm := NewPokerManager(session)
	
	payload, err := sm.NotifyBan(123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	action, err := FromConsensusPayload(payload)
	if err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}
	
	if action.Type != ActionBan || action.PlayerID != 123 {
		t.Fatal("ban notification has wrong content")
	}
}