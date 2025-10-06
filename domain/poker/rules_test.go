package poker

import "testing"

func TestCheckPokerLogic_Bet_InsufficientFunds(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 50, Bet: 0},
		},
		HighestBet: 0,
	}

	err := CheckPokerLogic(ActionBet, 100, session, 0)
	if err == nil {
		t.Fatal("expected error for insufficient funds")
	}
}

func TestCheckPokerLogic_Bet_SufficientFunds(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 100, Bet: 0},
		},
		HighestBet: 0,
	}

	err := CheckPokerLogic(ActionBet, 50, session, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckPokerLogic_Raise_BelowHighestBet(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 100, Bet: 50},
		},
		HighestBet: 100,
	}

	err := CheckPokerLogic(ActionRaise, 20, session, 0)
	if err == nil {
		t.Fatal("expected error: raise must at least match highest bet")
	}
}

func TestCheckPokerLogic_Call_InsufficientFunds(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 30, Bet: 50},
		},
		HighestBet: 100,
	}

	err := CheckPokerLogic(ActionCall, 0, session, 0)
	if err == nil {
		t.Fatal("expected error for insufficient funds to call")
	}
}

func TestCheckPokerLogic_Call_SufficientFunds(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 100, Bet: 50},
		},
		HighestBet: 100,
	}

	err := CheckPokerLogic(ActionCall, 0, session, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckPokerLogic_Check_WhenBetRequired(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 100, Bet: 50},
		},
		HighestBet: 100,
	}

	err := CheckPokerLogic(ActionCheck, 0, session, 0)
	if err == nil {
		t.Fatal("expected error: cannot check when bet is required")
	}
}

func TestCheckPokerLogic_Check_Valid(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 100, Bet: 100},
		},
		HighestBet: 100,
	}

	err := CheckPokerLogic(ActionCheck, 0, session, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckPokerLogic_AllIn_WrongAmount(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 100, Bet: 50},
		},
		HighestBet: 0,
	}

	err := CheckPokerLogic(ActionAllIn, 100, session, 0)
	if err == nil {
		t.Fatal("expected error: allin amount must match remaining pot")
	}
}

func TestCheckPokerLogic_AllIn_CorrectAmount(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 100, Bet: 50},
		},
		HighestBet: 0,
	}

	err := CheckPokerLogic(ActionAllIn, 150, session, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckPokerLogic_Fold_AlwaysValid(t *testing.T) {
	session := &Session{
		Players: []Player{
			{Name: "Alice", Pot: 0, Bet: 0},
		},
		HighestBet: 100,
	}

	err := CheckPokerLogic(ActionFold, 0, session, 0)
	if err != nil {
		t.Fatalf("fold should always be valid: %v", err)
	}
}
