package blockchain

import "github.com/luca-patrignani/mental-poker/poker"

type Block struct {
	ActivePlayer int
	Board [5]poker.Card
	Players []poker.Player
}
