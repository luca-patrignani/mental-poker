package consensus

import (
	"crypto/ed25519"
	"encoding/json"
	"testing"

	"github.com/luca-patrignani/mental-poker/domain/poker"
)

func TestNewEd25519KeypairAndSignVerify(t *testing.T) {
	pub, priv,_ := ed25519.GenerateKey(nil)
	pa := poker.PokerAction{
		RoundID:  "preflop",
		PlayerID: 1,
		Type:     poker.ActionBet,
		Amount:   10,
	}
	payload, _ := pa.ToConsensusPayload()
	a := &Action{
		Id:         "test",
		PlayerID:    1,
		Payload:    payload,
	}
	// sign will set Ts
	if err := a.Sign(priv); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	// verify should succeed
	ok, err := a.VerifySignature(pub)
	print(ok)
	if err != nil {
		t.Fatalf("Verify returned err: %v", err)
	}
	if !ok {
		t.Fatalf("signature verification failed")
	}
}

func TestVerifyFailsIfTampered(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)

	pa := poker.PokerAction{
		RoundID: "river",
		PlayerID: 17,
		Type: poker.ActionAllIn,
		Amount: 100,
	}
	bpa,_ := pa.ToConsensusPayload()

	a,err := makeAction(17,bpa)
	if err != nil {
		t.Fatalf("Failed to make an action, %v",err)
	}
	if err := a.Sign(priv); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	// copy bytes & tamper a field not covered by signature? all fields used in signingBytes are signed.
	// Tamper with Amount (signed field)
	a.PlayerID= 10
	ok, err := a.VerifySignature(pub)
	if err != nil {
		t.Fatalf("Verify returned err: %v", err)
	}
	if ok {
		t.Fatalf("Tampered action should not verify")
	}
}

func TestMarshalUnmarshalAction(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	act := poker.PokerAction{
		RoundID: "preflop",
		PlayerID: 15,
		Type: poker.ActionBet,
		Amount: 150,
	}
	bAct, err := act.ToConsensusPayload()
	if err != nil {
		t.Fatalf("%v",err)
	}

	a,err := makeAction(15, bAct)
	if err != nil {
		t.Fatalf("Failed to make an action, %v",err)
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

	p2, _ := poker.FromConsensusPayload(a2.Payload)
	// Verify a2 signature (you need original pub but we can't get pub from priv here)
	// Basic checks:
	if *p2 != act || a2.PlayerID != a.PlayerID || a2.Id != a.Id {
		t.Fatalf("unmarshaled action differs")
	}
}
