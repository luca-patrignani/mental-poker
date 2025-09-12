package blockchain

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"time"
)

type ActionType string

const (
	ActionBet    ActionType = "bet"
	ActionCall   ActionType = "call"
	ActionRaise  ActionType = "raise"
	ActionFold   ActionType = "fold"
	ActionCheck  ActionType = "check"
	ActionReveal ActionType = "reveal"
)

// Action is the transaction that goes into the replicated log / block proposal
type Action struct {
	RoundID   string     `json:"round_id"`
	PlayerID  string     `json:"player_id"`
	Type      ActionType `json:"type"`
	Amount    uint       `json:"amount"`
	Ts        int64      `json:"ts"`
	Signature []byte     `json:"sig,omitempty"`
}

func (a *Action) Sign(priv ed25519.PrivateKey) error {
	a.Ts = time.Now().UnixNano()
	b, err := a.signingBytes()
	if err != nil {
		return err
	}
	a.Signature = ed25519.Sign(priv, b)
	return nil
}

func (a *Action) VerifySignature(pub ed25519.PublicKey) (bool, error) {
	if len(a.Signature) == 0 {
		return false, errors.New("missing signature")
	}
	b, err := a.signingBytes()
	if err != nil {
		return false, err
	}
	return ed25519.Verify(pub, b, a.Signature), nil
}

// return serialized bytes of the fields covered by the signature
func (a *Action) signingBytes() ([]byte, error) {
	type sAction struct {
		RoundID  string     `json:"round_id"`
		PlayerID string     `json:"player_id"`
		Type     ActionType `json:"type"`
		Amount   uint       `json:"amount"`
		Ts       int64      `json:"ts"`
	}
	s := sAction{
		RoundID:  a.RoundID,
		PlayerID: a.PlayerID,
		Type:     a.Type,
		Amount:   a.Amount,
		Ts:       a.Ts,
	}
	return json.Marshal(s)
}
