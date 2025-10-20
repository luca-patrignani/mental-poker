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


type Blockchain struct {
	mu     sync.RWMutex
	blocks []Block
}

// NewBlockchain creates a new blockchain with an initialized genesis block.
// The genesis block has index 0, previous hash "0", and empty action/votes arrays
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

// Append adds a new validated block to the blockchain. It calculates the block hash,
// validates the block against the previous block, and appends it. Returns an error if
// the block is invalid. The extra parameter can optionally contain additional metadata.
func (bc *Blockchain) append(session poker.Session, pa poker.PokerAction, votes []consensus.Vote, proposerID int, quorum int, extra ...map[string]string) error {
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

	if err := bc.validateBlock(newBlock, latest); err != nil {
		return fmt.Errorf("invalid block: %w", err)
	}

	bc.blocks = append(bc.blocks, newBlock)

	return nil
}

// GetLatest returns the most recently added block in the blockchain.
// Returns an error if the blockchain is empty.
func (bc *Blockchain) GetLatest() (Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if len(bc.blocks) == 0 {
		return Block{}, fmt.Errorf("blockchain is empty")
	}

	return bc.blocks[len(bc.blocks)-1], nil
}

// GetByIndex retrieves a block by its index in the chain. Returns an error if the index
// is out of range.
func (bc *Blockchain) GetByIndex(index int) (*Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if index < 0 || index >= len(bc.blocks) {
		return nil, fmt.Errorf("index out of range")
	}

	return &bc.blocks[index], nil
}

// Verify validates the integrity of the entire blockchain by checking the genesis block
// and verifying each subsequent block's hash, index continuity, and previous hash linkage.
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

// validateBlock verifies that a block is valid relative to the previous block. It checks
// index continuity, previous hash linkage, current hash validity, and quorum requirements.
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

// calculateHash computes the SHA256 hash of a block based on its index, timestamp, previous
// hash, action, votes, proposer ID, and quorum. The action and votes are JSON marshaled
// before hashing.
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
