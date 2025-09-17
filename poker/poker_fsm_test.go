package poker

import (
	"testing"
)

func TestNewPokerFSM_InitialState(t *testing.T) {
	fsm := NewPokerFSM(2, 10)
	if fsm.GetCurrentPhase() != StateWaitingForPlayers {
		t.Errorf("expected initial phase to be %s, got %s", StateWaitingForPlayers, fsm.GetCurrentPhase())
	}
	if fsm.GetCurrentBettingState() != BettingNotStarted {
		t.Errorf("expected initial betting state to be %s, got %s", BettingNotStarted, fsm.GetCurrentBettingState())
	}
	if fsm.bigBlind != 20 {
		t.Errorf("expected big blind to be 20, got %d", fsm.bigBlind)
	}
	if fsm.smallBlind != 10 {
		t.Errorf("expected small blind to be 10, got %d", fsm.smallBlind)
	}
}

func setupSession(numPlayers int, dealerIdx int, pots []uint) *Session {
	players := make([]Player, numPlayers)
	for i := range players {
		players[i].Pot = pots[i]
	}
	return &Session{
		Players:     players,
		Dealer:      uint(dealerIdx),
		CurrentTurn: 0,
	}
}

func TestPokerFSM_postBlinds_Success(t *testing.T) {
	fsm := NewPokerFSM(2, 10)
	session := setupSession(3, 0, []uint{100, 100, 100})
	fsm.SetSession(session)
	err := fsm.postBlinds()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	expectedTurn := uint(0)
	if session.CurrentTurn != expectedTurn {
		t.Errorf("expected current turn to be %d, got %d", expectedTurn, session.CurrentTurn)
	}
}

func TestPokerFSM_postBlinds_NotEnoughPlayers(t *testing.T) {
	fsm := NewPokerFSM(2, 10)
	session := setupSession(1, 0, []uint{100})
	fsm.SetSession(session)
	err := fsm.postBlinds()
	if err == nil || err.Error() != "need at least 2 players" {
		t.Errorf("expected error for not enough players, got %v", err)
	}
}

func TestPokerFSM_postBlinds_SmallBlindCannotAfford(t *testing.T) {
	fsm := NewPokerFSM(2, 10)
	session := setupSession(3, 0, []uint{100, 5, 100})
	fsm.SetSession(session)
	err := fsm.postBlinds()
	if err == nil || err.Error() != "player 1 cannot afford small blind" {
		t.Errorf("expected error for small blind, got %v", err)
	}
}

func TestPokerFSM_postBlinds_BigBlindCannotAfford(t *testing.T) {
	fsm := NewPokerFSM(2, 10)
	session := setupSession(3, 0, []uint{100, 100, 10})
	fsm.SetSession(session)
	err := fsm.postBlinds()
	if err == nil || err.Error() != "player 2 cannot afford big blind" {
		t.Errorf("expected error for big blind, got %v", err)
	}
}
