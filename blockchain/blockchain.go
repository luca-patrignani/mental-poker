package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/luca-patrignani/mental-poker/communication"
)

type Block struct {
	Index    int
	PrevHash string
	Hash     string
	Action   communication.Action
	Votes    []communication.VoteMsg
	Ts       int64
}

// Calcolo hash includendo tutti i dati principali
func CalculateHash(block Block) string {
	data := fmt.Sprintf("%d%s%v%d%v", block.Index, block.PrevHash, block.Action, block.Ts, block.Votes)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Un singolo struct (o slice) rappresenta la chain
type Blockchain struct {
	Blocks []Block
}
