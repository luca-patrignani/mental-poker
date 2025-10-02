package poker

import (
	"encoding/json"
	"fmt"
)

// StateMachine implementa l'interfaccia consensus.StateMachine
// ma vive nel domain layer senza dipendenze dal consensus
type StateMachine struct {
	session *Session
}

func NewPokerStateMachine(session *Session) *StateMachine {
	return &StateMachine{session: session}
}

// Validate verifica se un'azione è valida nello stato corrente
func (psm *StateMachine) Validate(actionData []byte) error {
	pa, err := FromConsensusPayload(actionData)

	if err != nil {
		return fmt.Errorf("Wrong data format")
	}
	// Verifica round ID
	if pa.RoundID != psm.session.RoundID {
		return fmt.Errorf("wrong round: expected %s, got %s", psm.session.RoundID, pa.RoundID)
	}

	// Trova player index
	index := psm.FindPlayerIndex(pa.PlayerID)
	if index == -1 {
		return fmt.Errorf("player %d not in session", pa.PlayerID)
	}

	// Verifica turno
	if uint(index) != psm.session.CurrentTurn {
		return fmt.Errorf("not player's turn: current turn %d, player index %d", psm.session.CurrentTurn, index)
	}

	// Verifica logica poker
	return CheckPokerLogic(pa.Type, pa.Amount, psm.session, index)
}

// Apply applica un'azione validata allo stato
func (psm *StateMachine) Apply(actionData []byte) error {
	pa, err := FromConsensusPayload(actionData)
	if err != nil {
		return err
	}

	idx := psm.FindPlayerIndex(pa.PlayerID)
	if idx == -1 {
		return fmt.Errorf("player not found")
	}

	return ApplyAction(pa.Type, pa.Amount, psm.session, idx)
}

// GetCurrentActor ritorna l'ID del giocatore che deve agire
func (psm *StateMachine) GetCurrentPlayer() int {
	if psm.session.CurrentTurn >= uint(len(psm.session.Players)) {
		return -1
	}
	return psm.session.Players[psm.session.CurrentTurn].Id
}

func (psm *StateMachine) NotifyBan(id int) ([]byte, error) {
	pa := PokerAction{
		RoundID:  psm.session.RoundID,
		PlayerID: id,
		Type:     ActionBan,
		Amount:   0,
	}
	b, err := pa.ToConsensusPayload()
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Snapshot serializza lo stato corrente
func (psm *StateMachine) Snapshot() ([]byte, error) {
	return json.Marshal(psm.session)
}

// Restore ripristina lo stato da uno snapshot
func (psm *StateMachine) Restore(data []byte) error {
	return json.Unmarshal(data, &psm.session)
}

func (psm *StateMachine) FindPlayerIndex(playerID int) int {
	for i, p := range psm.session.Players {
		if p.Id == playerID {
			return i
		}
	}
	return -1
}

// GetSession espone la sessione (read-only idealmente)
func (psm *StateMachine) GetSession() *Session {
	return psm.session
}

// ToConsensusPayload serializza per il consensus layer
func (pa *PokerAction) ToConsensusPayload() ([]byte, error) {
	return json.Marshal(pa)
}

// FromConsensusPayload deserializza dal consensus layer
func FromConsensusPayload(data []byte) (*PokerAction, error) {
	var pa PokerAction
	err := json.Unmarshal(data, &pa)
	return &pa, err
}
