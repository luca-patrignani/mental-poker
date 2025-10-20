package poker

import (
	"fmt"
)

// PokerManager is an adapter of Peer to the interface NetworkLayer
type PokerManager struct {
	session *Session
}

// NewPokerManager creates a new PokerManager wrapping the provided poker session and
// implementing the consensus.StateMachine interface.
func NewPokerManager(session *Session) *PokerManager {
	return &PokerManager{session: session}
}

// Validate checks whether a poker action is valid in the current session state by verifying
// the round ID, player existence, turn order, and poker rules. Returns an error describing
// the validation failure, or nil if the action is valid.
func (psm *PokerManager) Validate(pa PokerAction) error {
	if pa.RoundID != psm.session.RoundID {
		return fmt.Errorf("wrong round: expected %s, got %s", psm.session.RoundID, pa.RoundID)
	}

	index := psm.FindPlayerIndex(pa.PlayerID)
	if index == -1 {
		return fmt.Errorf("player %d not in session", pa.PlayerID)
	}

	if uint(index) != psm.session.CurrentTurn {
		return fmt.Errorf("not player's turn: current turn %d, player index %d", psm.session.CurrentTurn, index)
	}

	return checkPokerLogic(pa.Type, pa.Amount, psm.session, index)
}

// Apply applies a validated poker action to the session state and advances the game accordingly.
// It delegates to ApplyAction with the player's session index. Returns an error if the player
// is not found in the session.
func (psm *PokerManager) Apply(pa PokerAction) error {
	idx := psm.FindPlayerIndex(pa.PlayerID)
	if idx == -1 {
		return fmt.Errorf("player not found")
	}

	return applyAction(pa.Type, pa.Amount, psm.session, idx)
}

// GetCurrentPlayer returns the player index in the session of the player whose turn it is to act.
// Returns -1 if the current turn index is out of bounds.
func (psm *PokerManager) GetCurrentPlayer() int {
	if psm.session.CurrentTurn >= uint(len(psm.session.Players)) {
		return -1
	}
	return psm.session.Players[psm.session.CurrentTurn].Id
}

// NotifyBan creates a ban PokerAction for the specified player ID. Returns an error if the
// player is not found in the session.
func (psm *PokerManager) NotifyBan(id int) (PokerAction, error) {

	err := psm.FindPlayerIndex(id)
	if err == -1 {
		return PokerAction{}, fmt.Errorf("player not found")
	}
	pa := PokerAction{
		RoundID:  psm.session.RoundID,
		PlayerID: id,
		Type:     ActionBan,
		Amount:   0,
	}

	return pa, nil
}

// FindPlayerIndex returns the session index of the player with the given ID, or -1 if not found.
func (psm *PokerManager) FindPlayerIndex(playerID int) int {
	return psm.session.FindPlayerIndex(playerID)
}

// GetSession returns a pointer to the underlying poker session managed by this PokerManager.
func (psm *PokerManager) GetSession() *Session {
	return psm.session
}

func (psm *PokerManager) GetWinners() (map[int]uint, error) {
	if extractRoundName(psm.session.RoundID) != Showdown {
		return nil, fmt.Errorf("cannot get winners before showdown")
	}
	return psm.session.winnerEval()
}
