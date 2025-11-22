package poker

import "testing"

func TestManager_ValidateWrongRound(t *testing.T) {
	session := &Session{
		Round:       "round1",
		Players:     []Player{{Id: 1, Name: "Alice"}},
		CurrentTurn: 0,
	}

	sm := &PokerManager{session, 1}

	action := &PokerAction{
		Round:    "round2",
		PlayerID: 1,
		Type:     ActionCheck,
		Amount:   0,
	}

	payload := action
	err := sm.Validate(*payload)

	if err == nil {
		t.Fatal("expected error for wrong round")
	}
}

func TestManager_ValidatePlayerNotInSession(t *testing.T) {
	session := &Session{
		Round:       "round1",
		Players:     []Player{{Id: 999, Name: "Alice"}},
		CurrentTurn: 0,
	}

	sm := &PokerManager{session, 1}

	action := &PokerAction{
		Round:    "preflop",
		PlayerID: 999,
		Type:     ActionCheck,
		Amount:   0,
	}

	payload := action
	err := sm.Validate(*payload)

	if err == nil {
		t.Fatal("expected error for player not in session")
	}
}

func TestManager_ValidateWrongTurn(t *testing.T) {
	session := &Session{
		Round: "round1",
		Players: []Player{
			{Id: 1, Name: "Alice"},
			{Id: 2, Name: "Bob"},
		},
		CurrentTurn: 0,
	}

	sm := &PokerManager{session, 1}

	action := &PokerAction{
		Round:    "round1",
		PlayerID: 2, // Bob trying to act when it's Alice's turn
		Type:     ActionCheck,
		Amount:   0,
	}

	payload := action
	err := sm.Validate(*payload)

	if err == nil {
		t.Fatal("expected error for wrong turn")
	}
}

func TestManager_ApplyValidAction(t *testing.T) {
	session := &Session{
		Round: "preflop1",
		Players: []Player{
			{Id: 1, Name: "Alice", BankRoll: 100, Bet: 0},
			{Id: 2, Name: "Bob", BankRoll: 100, Bet: 0},
		},
		CurrentTurn: 0,
		HighestBet:  0,
	}

	sm := &PokerManager{session, 1}

	action := &PokerAction{
		Round:    "preflop1",
		PlayerID: 1,
		Type:     ActionBet,
		Amount:   50,
	}

	payload := action
	err := sm.Apply(*payload)

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

	sm := &PokerManager{session, 1}

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

	sm := &PokerManager{session, 1}

	idx := sm.FindPlayerIndex(200)
	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}

	idx = sm.FindPlayerIndex(999)
	if idx != -1 {
		t.Fatalf("expected -1 for non-existent player, got %d", idx)
	}
}

func TestManager_NotifyBan(t *testing.T) {
	session := &Session{
		Round:   "round1",
		Players: []Player{{Id: 123, Name: "Alice", BankRoll: 100}},
	}

	sm := &PokerManager{session, 1}

	payload, err := sm.NotifyBan(123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if payload.Type != ActionBan || payload.PlayerID != 123 {
		t.Fatal("ban notification has wrong content")
	}
}

func TestManager_RemoveById(t *testing.T) {
	session := &Session{
		Round: "preflop",
		Players: []Player{
			{Id: 123, Name: "Alice", BankRoll: 100},
			{Id: 32, Name: "Fabio", BankRoll: 100},
			{Id: 54, Name: "Gianni", BankRoll: 100},
			{Id: 2, Name: "Luca", BankRoll: 100}},
	}

	sm := &PokerManager{session, 54}
	p, err := sm.RemoveByID(54)

	if err != nil {
		t.Fatal(err.Error())
	}
	if p.Id != 54 {
		t.Fatalf("Removed the wrong player: expected 54, instead of %d", p.Id)
	}
	if len(sm.Session.Players) != 3 {
		t.Fatal("Player not removed")
	}
	for i, p := range sm.Session.Players {
		if p.Id == 54 {
			t.Fatalf("Player 54 was exepected to be removed, instead is present at index %d", i)
		}
	}
}
