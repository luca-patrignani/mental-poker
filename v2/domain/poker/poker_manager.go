package poker

import (
	"fmt"
)

// PokerManager adapts a poker session to the consensus.StateManager interface.
// It validates actions against poker rules and applies state transitions.
type PokerManager struct {
	Session *Session // The current poker game state
	Player  int      // This node's player ID
}

// Validate checks whether a poker action is valid in the current session state.
//
// Validation checks:
//   - Round ID matches current game round
//   - Player exists in the session
//   - It's the player's turn to act
//   - Action complies with poker rules (sufficient funds, valid bet amounts, etc.)
//
// Returns nil if the action is valid, or an error describing the validation failure.
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

// Apply executes a validated poker action, modifying the session state.
//
// This method delegates to applyAction which handles:
//   - Updating player chip stacks
//   - Advancing the turn
//   - Progressing betting rounds
//   - Calculating side pots
//   - Determining winners
//
// Returns an error if the player is not found or if the state transition fails.
//
// Warning: This method assumes the action has already been validated.
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
	if psm.Session.CurrentTurn > uint(len(psm.Session.Players)) {
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
		psm.Session.Players[i].HasFolded = false
		psm.Session.Players[i].Bet = 0
		psm.Session.Players[i].Hand[0] = c
		psm.Session.Players[i].Hand[1] = c

		if psm.Session.Players[i].BankRoll <= 0 {
			psm.Session.Players[i].HasFolded = true
		}

	}
	for i := range psm.Session.Board {
		psm.Session.Board[i] = c
	}
	psm.Session.setNextMatchDealer()
	psm.Session.LastToRaise = psm.Session.Dealer
	psm.Session.HighestBet = 0
	psm.Session.Pots = []Pot{{Amount: 0}}
}

func (psm *PokerManager) RemoveByID(id int) (Player, error) {
	for i, p := range psm.Session.Players {
		if p.Id == id {
			psm.Session.Players = append(psm.Session.Players[:i], psm.Session.Players[i+1:]...)
			return p, nil
		}
	}
	return Player{}, fmt.Errorf("Player %d, not found", id) // nessun elemento rimosso se id non trovato
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
		Amount:   psm.Session.Players[psm.FindPlayerIndex(psm.Player)].BankRoll,
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
