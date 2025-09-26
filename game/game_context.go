package game

import (
	"github.com/luca-patrignani/mental-poker/blockchain"
	"github.com/luca-patrignani/mental-poker/communication"
	"github.com/luca-patrignani/mental-poker/poker"
)

type GameContext struct {
	Node       *communication.Node
	Blockchain *blockchain.Blockchain
	Session    *poker.Session
}
