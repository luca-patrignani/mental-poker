package common

import (
	"fmt"

	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/proof"
	"go.dedis.ch/kyber/v3/util/random"
)

// DLRepWitness rappresenta la conoscenza del segreto r
type DLRepWitness struct {
	R kyber.Scalar // Segreto condiviso
}

// DLRepStatement rappresenta l'affermazione da verificare
type DLRepStatement struct {
	G      kyber.Point // Generatore g
	H      kyber.Point // Generatore h
	Gprime kyber.Point // Punto pubblico R = g^r
	Hprime kyber.Point // Punto pubblico L = h^r
}

// ProveDLREP genera una ZKP per il logaritmo discreto condiviso
func ProveDLREP(suite kyber.Group, witness *DLRepWitness, statement *DLRepStatement) ([]byte, error) {
	// Setup: Definizione dei commitment e challenge
	prover := proof.Rep([]kyber.Point{statement.G, statement.H}, // Generatori g, h
		[]kyber.Point{statement.Gprime, statement.Hprime}, // Punti pubblici R, L
	)

	// Segreti associati
	secrets := []kyber.Scalar{witness.R}

	// Genera la prova
	proofData, err := prover.Prove(suite, random.New(), secrets)
	if err != nil {
		return nil, fmt.Errorf("errore nella creazione della prova: %v", err)
	}
	return proofData, nil
}

// VerifyDLREP verifica la validità della ZKP
func VerifyDLREP(suite kyber.Group, proofData []byte, statement *DLRepStatement) error {
	// Setup del verificatore
	verifier := proof.Rep{
		G: []kyber.Point{statement.G, statement.H},           // Generatori g, h
		H: []kyber.Point{statement.Gprime, statement.Hprime}, // Punti pubblici R, L
	}

	// Verifica la prova
	err := verifier.Verify(suite, proofData)
	if err != nil {
		return fmt.Errorf("la prova non è valida: %v", err)
	}
	return nil
}
