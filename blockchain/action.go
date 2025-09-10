package blockchain

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"time"
)

// ActionType enumerates poker actions
type ActionType string

const (
	ActionBet    ActionType = "bet"
	ActionCall   ActionType = "call"
	ActionRaise  ActionType = "raise"
	ActionFold   ActionType = "fold"
	ActionCheck  ActionType = "check"
	ActionReveal ActionType = "reveal"
)

// Action is the tx that goes into the replicated log / block proposal
type Action struct {
	RoundID   string     `json:"round_id"`
	PlayerID  string     `json:"player_id"` // unique player name or pubkey hex
	Type      ActionType `json:"type"`
	Amount    uint       `json:"amount"`
	Ts        int64      `json:"ts"`
	Signature []byte     `json:"sig,omitempty"`
}

// SignAction signs the action with the given private key
func (a *Action) Sign(priv ed25519.PrivateKey) error {
	a.Ts = time.Now().UnixNano()
	b, err := a.signingBytes()
	if err != nil {
		return err
	}
	a.Signature = ed25519.Sign(priv, b)
	return nil
}

// VerifySignature verifies the action signature with pubkey
func (a *Action) VerifySignature(pub ed25519.PublicKey) (bool, error) {
	if a.Signature == nil || len(a.Signature) == 0 {
		return false, errors.New("missing signature")
	}
	b, err := a.signingBytes()
	if err != nil {
		return false, err
	}
	return ed25519.Verify(pub, b, a.Signature), nil
}

// signingBytes returns deterministic serialization used for signing
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

// Marshal/Unmarshal helpers; used by network and vote logic
func (a *Action) Bytes() ([]byte, error) {
	return json.Marshal(a)
}

func ActionFromBytes(b []byte) (*Action, error) {
	var a Action
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// Utility to generate a keypair for testing
func NewEd25519Keypair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	return pub, priv, err
}
