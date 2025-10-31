package poker

import (
	"fmt"
)

// PokerManager is an adapter of Peer to the interface NetworkLayer
type PokerManager struct {
	Session *Session
	Player  int
}

// NewPokerManager creates a new PokerManager wrapping the provided poker session and
// implementing the consensus.StateMachine interface.

// Validate checks whether a poker action is valid in the current session state by verifying
// the round ID, player existence, turn order, and poker rules. Returns an error describing
// the validation failure, or nil if the action is valid.
func (psm *PokerManager) Validate(pa PokerAction) error {

	index := psm.FindPlayerIndex(pa.PlayerID)
	if err := checkPokerLogic(pa.Type, pa.Amount, psm.Session, index); err != nil {
		return err
	}

	if pa.Round != psm.Session.Round {
		return fmt.Errorf("wrong round: expected %s, got %s", psm.Session.Round, pa.Round)
	}

	if index == -1 {
		return fmt.Errorf("player %d not in session", pa.PlayerID)
	}

	if uint(index) != psm.Session.CurrentTurn {
		return fmt.Errorf("not player's turn: current turn %d, player index %d", psm.Session.CurrentTurn, index)
	}

	return nil
}

// Apply applies a validated poker action to the session state and advances the game accordingly.
// It delegates to ApplyAction with the player's session index. Returns an error if the player
// is not found in the session.
func (psm *PokerManager) Apply(pa PokerAction) error {
	idx := psm.FindPlayerIndex(pa.PlayerID)
	if idx == -1 {
		return fmt.Errorf("player not found")
	}

	return applyAction(pa.Type, pa.Amount, psm.Session, idx)
}

// GetCurrentPlayer returns the player index in the session of the player whose turn it is to act.
// Returns -1 if the current turn index is out of bounds.
func (psm *PokerManager) GetCurrentPlayer() int {
	print(psm.Session.CurrentTurn)
	if psm.Session.CurrentTurn >= uint(len(psm.Session.Players)) {
		return -1
	}
	return psm.Session.Players[psm.Session.CurrentTurn].Id
}

// NotifyBan creates a ban PokerAction for the specified player ID. Returns an error if the
// player is not found in the session.
func (psm *PokerManager) NotifyBan(id int) (PokerAction, error) {

	err := psm.FindPlayerIndex(id)
	if err == -1 {
		return PokerAction{}, fmt.Errorf("player not found")
	}
	pa := PokerAction{
		Round:    psm.Session.Round,
		PlayerID: id,
		Type:     ActionBan,
		Amount:   0,
	}

	return pa, nil
}

// FindPlayerIndex returns the session index of the player with the given ID, or -1 if not found.
func (psm *PokerManager) FindPlayerIndex(playerID int) int {
	return psm.Session.FindPlayerIndex(playerID)
}

// GetSession returns a pointer to the underlying poker session managed by this PokerManager.
func (psm *PokerManager) GetSession() *Session {
	return psm.Session
}

// return a map of winning player and their corresponding amount
func (psm *PokerManager) GetWinners() (map[int]uint, error) {
	if psm.Session.Round != Showdown {
		return nil, fmt.Errorf("cannot get winners before showdown")
	}
	return psm.Session.winnerEval()
}

func (psm *PokerManager) PrepareNextMatch() {
	c, _ := NewCard(0, 0)
	for i := range psm.Session.Players {
		psm.Session.Players[i].Hand[0] = c
		psm.Session.Players[i].Hand[1] = c
	}
	for i := range psm.Session.Board {
		psm.Session.Board[i] = c
	}
	psm.Session.setNextMatchDealer()
	psm.Session.LastToRaise = psm.Session.Dealer
	psm.Session.HighestBet = 0
	psm.Session.Pots = []Pot{{Amount: 0}}
}

func (psm *PokerManager) ActionFold() PokerAction {
	return PokerAction{
		Round:    psm.Session.Round,
		PlayerID: psm.Player,
		Type:     ActionFold,
		Amount:   0,
	}
}

func (psm *PokerManager) ActionCheck() PokerAction {
	return PokerAction{
		Round:    psm.Session.Round,
		PlayerID: psm.Player,
		Type:     ActionCheck,
		Amount:   0,
	}
}
func (psm *PokerManager) ActionCall() PokerAction {
	return PokerAction{
		Round:    psm.Session.Round,
		PlayerID: psm.Player,
		Type:     ActionCall,
		Amount:   psm.Session.HighestBet - psm.Session.Players[psm.FindPlayerIndex(psm.Player)].Bet,
	}
}
func (psm *PokerManager) ActionRaise(amount uint) PokerAction {
	return PokerAction{
		Round:    psm.Session.Round,
		PlayerID: psm.Player,
		Type:     ActionRaise,
		Amount:   amount,
	}
}
func (psm *PokerManager) ActionAllIn() PokerAction {
	return PokerAction{
		Round:    psm.Session.Round,
		PlayerID: psm.Player,
		Type:     ActionAllIn,
		Amount:   psm.Session.Players[psm.FindPlayerIndex(psm.Player)].Pot,
	}
}
func (psm *PokerManager) ActionShowdown() PokerAction {
	return PokerAction{
		Round:    psm.Session.Round,
		PlayerID: psm.Player,
		Type:     ActionShowdown,
		Amount:   0,
	}
}
func (psm *PokerManager) ActionBet(amount uint) PokerAction {
	return PokerAction{
		Round:    psm.Session.Round,
		PlayerID: psm.Player,
		Type:     ActionBet,
		Amount:   amount,
	}
}
