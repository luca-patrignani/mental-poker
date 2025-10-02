package consensus

// StateMachine è l'interfaccia che il consensus layer usa per applicare azioni
// Il dominio poker implementerà questa interfaccia senza sapere del consensus
type StateMachine interface {
	// Validate verifica se un'azione è valida nello stato corrente
	Validate(payload []byte) error

	// Apply applica un'azione validata allo stato
	Apply(payload []byte) error

	// GetCurrentActor ritorna chi deve agire ora
	GetCurrentPlayer() int

	FindPlayerIndex(id int) int

	NotifyBan(id int) ([]byte, error)

	// Snapshot crea uno snapshot dello stato corrente
	Snapshot() []byte

	// Restore ripristina lo stato da uno snapshot
	Restore(data []byte) error
}

// Ledger è l'interfaccia per registrare azioni committate
type Ledger interface {
	// Append aggiunge un nuovo blocco con l'azione committata
	Append(action *Action, votes []Vote, quorum int) error

	// GetLatest ritorna l'ultimo blocco
	GetLatest() (Block, error)

	// Verify verifica l'integrità della chain
	Verify() error
}

// Block rappresenta un blocco nel ledger
type Block struct {
	Index     int
	PrevHash  string
	Hash      string
	Action    *Action
	Votes     []Vote
	Timestamp int64
}

// NetworkLayer astrae la comunicazione P2P
type NetworkLayer interface {
	// Broadcast invia dati a tutti i peer
	Broadcast(data []byte, root int) ([]byte, error)

	// AllToAll invia dati da tutti a tutti
	AllToAll(data []byte) ([][]byte, error)

	// GetRank ritorna il rank di questo nodo
	GetId() int

	// GetPeerCount ritorna il numero di peer
	GetPeerCount() int

	Close() error
}
