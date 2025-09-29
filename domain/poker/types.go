package poker

import "github.com/luca-patrignani/mental-poker/domain/deck"

type Player struct {
	Name      string
	Id        int //rank
	Hand      [2]Card
	HasFolded bool
	Bet       uint // The amount of money bet in the current betting round
	Pot       uint
}

// PokerAction è l'azione specifica del dominio poker
type PokerAction struct {
	RoundID  string     `json:"round_id"`
	PlayerID int        `json:"player_id"`
	Type     ActionType `json:"type"`
	Amount   uint       `json:"amount"`
}

type ActionType string

const (
	ActionBet    ActionType = "bet"
	ActionCall   ActionType = "call"
	ActionRaise  ActionType = "raise"
	ActionAllIn  ActionType = "allin"
	ActionFold   ActionType = "fold"
	ActionCheck  ActionType = "check"
	ActionReveal ActionType = "reveal"
	ActionBan    ActionType = "ban"
)

// Deck is the rappresentation of a game session.
type Session struct {
	Board       [5]Card
	Players     []Player
	Deck        deck.Deck
	Pots        []Pot
	HighestBet  uint
	Dealer      uint
	CurrentTurn uint   // index into Players for who must act
	RoundID     string // identifier for the current betting round/hand
}

type Pot struct {
	Amount   uint
	Eligible []int // PlayerIDs che possono vincere questo piatto
}

const (
	Club    = 0
	Diamond = 1
	Heart   = 2
	Spade   = 3
)

const (
	Jack  = 11
	Queen = 12
	King  = 13
	Ace   = 1
)

type Card struct {
	suit uint8
	rank uint8
}
