package consensus

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"time"
)

func (a *Action) serialize() ([]byte, error) {
	tmp := *a
	tmp.Signature = nil
	return json.Marshal(tmp)
}

func (v *Vote) serialize() ([]byte, error) {
	tmp := *v
	tmp.Signature = nil
	return json.Marshal(tmp)
}

func (a *Action) Sign(priv ed25519.PrivateKey) error {
	a.Timestamp = time.Now().UnixNano()
	b, err := a.serialize()
	if err != nil {
		return err
	}
	a.Signature = ed25519.Sign(priv, b)
	return nil
}

func (v *Vote) Sign(priv ed25519.PrivateKey) error {
	b, err := v.serialize()
	if err != nil {
		return err
	}
	v.Signature = ed25519.Sign(priv, b)
	return nil
}

func (a *Action) VerifySignature(pub ed25519.PublicKey) (bool, error) {
	if len(a.Signature) == 0 {
		return false, errors.New("missing signature")
	}
	b, err := a.serialize()
	if err != nil {
		return false, err
	}
	return ed25519.Verify(pub, b, a.Signature), nil
}

func (v *Vote) VerifySignature(pub ed25519.PublicKey) (bool, error) {
	if len(v.Signature) == 0 {
		return false, errors.New("missing signature")
	}
	b, err := v.serialize()
	if err != nil {
		return false, err
	}
	return ed25519.Verify(pub, b, v.Signature), nil
}
