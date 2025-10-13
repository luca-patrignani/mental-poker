package application

import (
	"github.com/luca-patrignani/mental-poker/consensus"
	"github.com/luca-patrignani/mental-poker/domain/poker"
	"github.com/luca-patrignani/mental-poker/ledger"
)

// Trasformalo in GameOrchestrator con responsabilità chiare:
type GameOrchestrator struct {
	consensus    *consensus.ConsensusNode // Gestisce consenso
	stateMachine *poker.PokerManager      // Stato del gioco
	blockchain   *ledger.Blockchain       // Log immutabile
}
