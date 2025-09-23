package communication

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/luca-patrignani/mental-poker/poker"
)

func TestNewEd25519KeypairAndSignVerify(t *testing.T) {
	pub, priv := mustKeypair(t)

	a := &Action{
		RoundID:  "r1",
		PlayerID: "alice",
		Type:     poker.ActionBet,
		Amount:   10,
	}
	// sign will set Ts
	if err := a.Sign(priv); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	// verify should succeed
	ok, err := a.VerifySignature(pub)
	if err != nil {
		t.Fatalf("Verify returned err: %v", err)
	}
	if !ok {
		t.Fatalf("signature verification failed")
	}
}

func TestVerifyFailsIfTampered(t *testing.T) {
	pub, priv := mustKeypair(t)

	a := &Action{
		RoundID:  "r1",
		PlayerID: "alice",
		Type:     poker.ActionRaise,
		Amount:   20,
	}
	if err := a.Sign(priv); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	// copy bytes & tamper a field not covered by signature? all fields used in signingBytes are signed.
	// Tamper with Amount (signed field)
	a.Amount = 999
	ok, err := a.VerifySignature(pub)
	if err != nil {
		t.Fatalf("Verify returned err: %v", err)
	}
	if ok {
		t.Fatalf("Tampered action should not verify")
	}
}

func TestMarshalUnmarshalAction(t *testing.T) {
	_, priv := mustKeypair(t)
	a := &Action{
		RoundID:  "round-abc",
		PlayerID: "bob",
		Type:     poker.ActionCall,
		Amount:   5,
	}
	if err := a.Sign(priv); err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("Bytes failed: %v", err)
	}

	var a2 Action
	if err := json.Unmarshal(b, &a2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify a2 signature (you need original pub but we can't get pub from priv here)
	// Basic checks:
	if a2.RoundID != a.RoundID || a2.PlayerID != a.PlayerID || a2.Type != a.Type || a2.Amount != a.Amount {
		t.Fatalf("unmarshaled action differs")
	}
}

func TestSigningBytesDeterministic(t *testing.T) {
	// build action and set Ts deterministically
	a := &Action{
		RoundID:  "r2",
		PlayerID: "carol",
		Type:     poker.ActionCheck,
		Amount:   0,
		Ts:       time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
	}
	b1, err := a.signingBytes()
	if err != nil {
		t.Fatalf("signingBytes err: %v", err)
	}
	// rebuild identical action and expect same bytes
	a2 := &Action{
		RoundID:  "r2",
		PlayerID: "carol",
		Type:     poker.ActionCheck,
		Amount:   0,
		Ts:       a.Ts,
	}
	b2, err := a2.signingBytes()
	if err != nil {
		t.Fatalf("signingBytes err: %v", err)
	}
	if string(b1) != string(b2) {
		t.Fatalf("signingBytes not deterministic: %s vs %s", string(b1), string(b2))
	}
}
