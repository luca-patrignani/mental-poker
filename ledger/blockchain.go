package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/luca-patrignani/mental-poker/consensus"
	"github.com/luca-patrignani/mental-poker/domain/poker"
)

// Blockchain gestisce la catena di blocchi
type Blockchain struct {
	mu     sync.RWMutex
	blocks []Block
}

// NewBlockchain crea una nuova blockchain con genesis block
func NewBlockchain() *Blockchain {
	bc := &Blockchain{
		blocks: make([]Block, 0),
	}

	// Crea genesis block
	genesis := Block{
		Index:     0,
		Timestamp: time.Now().Unix(),
		PrevHash:  "0",
		Session:   poker.Session{},
		Action:    poker.PokerAction{Type: "genesis"},
		Votes:     []consensus.Vote{},
		Metadata:  Metadata{ProposerID: -1, Quorum: 0},
	}
	genesis.Hash = bc.calculateHash(genesis)
	bc.blocks = append(bc.blocks, genesis)

	return bc
}

// Append aggiunge un nuovo blocco validato
func (bc *Blockchain) Append(session poker.Session, pa poker.PokerAction, votes []consensus.Vote, proposerID int, quorum int, extra ...map[string]string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	var extraMsg map[string]string
	if len(extra) > 0 {
		extraMsg = extra[0]
	}
	latest := bc.blocks[len(bc.blocks)-1]

	newBlock := Block{
		Index:     latest.Index + 1,
		Timestamp: time.Now().Unix(),
		PrevHash:  latest.Hash,
		Session:   session,
		Action:    pa,
		Votes:     votes,
		Metadata: Metadata{
			ProposerID: proposerID,
			Quorum:     quorum,
			Extra:      extraMsg,
		},
	}

	newBlock.Hash = bc.calculateHash(newBlock)

	// Verifica che il blocco sia valido
	if err := bc.validateBlock(newBlock, latest); err != nil {
		return fmt.Errorf("invalid block: %w", err)
	}

	bc.blocks = append(bc.blocks, newBlock)

	return nil
}

// GetLatest ritorna l'ultimo blocco
func (bc *Blockchain) GetLatest() (Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if len(bc.blocks) == 0 {
		return Block{}, fmt.Errorf("blockchain is empty")
	}

	return bc.blocks[len(bc.blocks)-1], nil
}

// GetByIndex ritorna un blocco per indice
func (bc *Blockchain) GetByIndex(index int) (*Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if index < 0 || index >= len(bc.blocks) {
		return nil, fmt.Errorf("index out of range")
	}

	return &bc.blocks[index], nil
}

// GetAll ritorna tutti i blocchi
func (bc *Blockchain) GetAll() []Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]Block, len(bc.blocks))
	copy(result, bc.blocks)
	return result
}

// Verify verifica l'integrità dell'intera chain
func (bc *Blockchain) Verify() error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if len(bc.blocks) == 0 {
		return fmt.Errorf("empty blockchain")
	}

	// Verifica genesis
	if bc.blocks[0].PrevHash != "0" {
		return fmt.Errorf("invalid genesis block")
	}

	// Verifica ogni blocco
	for i := 1; i < len(bc.blocks); i++ {
		current := bc.blocks[i]
		previous := bc.blocks[i-1]

		if err := bc.validateBlock(current, previous); err != nil {
			return fmt.Errorf("block %d invalid: %w", i, err)
		}
	}

	return nil
}

// validateBlock verifica che un blocco sia valido rispetto al precedente
func (bc *Blockchain) validateBlock(current, previous Block) error {
	// Verifica indice
	if current.Index != previous.Index+1 {
		return fmt.Errorf("invalid index: expected %d, got %d", previous.Index+1, current.Index)
	}

	// Verifica prev hash
	if current.PrevHash != previous.Hash {
		return fmt.Errorf("invalid prev hash: expected %s, got %s", previous.Hash, current.PrevHash)
	}

	// Verifica hash corrente
	expectedHash := bc.calculateHash(current)
	if current.Hash != expectedHash {
		return fmt.Errorf("invalid hash: expected %s, got %s", expectedHash, current.Hash)
	}

	// Verifica quorum (almeno 2n/3 voti)
	if len(current.Votes) < current.Metadata.Quorum {
		return fmt.Errorf("insufficient votes: got %d, need %d", len(current.Votes), current.Metadata.Quorum)
	}

	return nil
}

// calculateHash calcola l'hash di un blocco
func (bc *Blockchain) calculateHash(block Block) string {
	// Serializza action
	actionBytes, _ := json.Marshal(block.Action)

	// Serializza votes
	votesBytes, _ := json.Marshal(block.Votes)

	// Concatena tutti i dati
	data := fmt.Sprintf("%d%d%s%s%s%d%d",
		block.Index,
		block.Timestamp,
		block.PrevHash,
		string(actionBytes),
		string(votesBytes),
		block.Metadata.ProposerID,
		block.Metadata.Quorum,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Length ritorna il numero di blocchi
func (bc *Blockchain) Length() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.blocks)
}

// GetHistory ritorna lo storico delle azioni (senza genesis)
func (bc *Blockchain) GetHistory() []interface{} {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	history := make([]interface{}, 0, len(bc.blocks)-1)
	for i := 1; i < len(bc.blocks); i++ {
		history = append(history, bc.blocks[i].Action)
	}
	return history
}

// Replay ricostruisce lo stato applicando tutte le azioni
func (bc *Blockchain) Replay(stateMachine StateMachine) error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	// Skip genesis block
	for i := 1; i < len(bc.blocks); i++ {
		pa, err := bc.blocks[i].Action.ToConsensusPayload()
		if err != nil {
			return fmt.Errorf("block %d: failed to serialize action: %w", i, err)
		}

		if err := stateMachine.Apply(pa); err != nil {
			return fmt.Errorf("block %d: failed to apply action: %w", i, err)
		}
	}

	return nil
}

// StateMachine interface per replay
type StateMachine interface {
	Apply(actionData []byte) error
}
