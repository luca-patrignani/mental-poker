package consensus

import "github.com/luca-patrignani/mental-poker/domain/poker"

// StateMachine è l'interfaccia che il consensus layer usa per applicare azioni
// Il dominio poker implementerà questa interfaccia senza sapere del consensus
type StateMachine interface {
	// Validate verifica se un'azione è valida nello stato corrente
	Validate(payload poker.PokerAction) error

	// Apply applica un'azione validata allo stato
	Apply(payload poker.PokerAction) error

	// GetCurrentActor ritorna chi deve agire ora
	GetCurrentPlayer() int

	FindPlayerIndex(id int) int

	NotifyBan(id int) (poker.PokerAction, error)

	GetSession() *poker.Session
	// Restore ripristina lo stato da uno snapshot
	Restore(data []byte) error
}

// Ledger è l'interfaccia per registrare azioni committate
type Ledger interface {
	// Append aggiunge un nuovo blocco con l'azione committata
	Append(session poker.Session, action poker.PokerAction, votes []Vote, proposerID int, quorum int, extra ...map[string]string) error

	// Verify verifica l'integrità della chain
	Verify() error
}

// NetworkLayer astrae la comunicazione P2P
type NetworkLayer interface {
	// Broadcast invia dati a tutti i peer
	Broadcast(data []byte, root int) ([]byte, error)

	// AllToAll invia dati da tutti a tutti
	AllToAll(data []byte) ([][]byte, error)

	// GetRank ritorna il rank di questo nodo
	GetRank() int

	// GetPeerCount ritorna il numero di peer
	GetPeerCount() int

	Close() error
}
