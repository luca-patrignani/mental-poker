package communication

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

// helpers used by tests
func mustKeypair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate keypair: %v", err)
	}
	return pub, priv
}
