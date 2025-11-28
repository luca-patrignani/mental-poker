package consensus

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"time"
)

// serialize returns the JSON marshaled form of the Action with the Signature field cleared
// to ensure the signature is not included in signed data.
func (a *Action) serialize() ([]byte, error) {
	tmp := *a
	tmp.Signature = nil
	return json.Marshal(tmp)
}

// serialize returns the JSON marshaled form of the Vote with the Signature field cleared
// to ensure the signature is not included in signed data.
func (v *Vote) serialize() ([]byte, error) {
	tmp := *v
	tmp.Signature = nil
	return json.Marshal(tmp)
}

// Sign signs the Action using the provided Ed25519 private key. It sets the current
// Unix nanosecond timestamp and generates a signature over the serialized action data.
func (a *Action) Sign(priv ed25519.PrivateKey) error {
	a.Timestamp = time.Now().UnixNano()
	b, err := a.serialize()
	if err != nil {
		return err
	}
	a.Signature = ed25519.Sign(priv, b)
	return nil
}

// Sign signs the Vote using the provided Ed25519 private key. It generates a signature
// over the serialized vote data.
func (v *Vote) Sign(priv ed25519.PrivateKey) error {
	b, err := v.serialize()
	if err != nil {
		return err
	}
	v.Signature = ed25519.Sign(priv, b)
	return nil
}

// VerifySignature verifies the Action's signature using the provided Ed25519 public key.
// Returns false if verification fails or an error if the signature is missing or serialization fails.
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

// VerifySignature verifies the Vote's signature using the provided Ed25519 public key.
// Returns false if verification fails or an error if the signature is missing or serialization fails.
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
