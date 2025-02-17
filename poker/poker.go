package poker

import (
	"errors"
	"fmt"
	"sort"

	"github.com/luca-patrignani/mental-poker/deck"
	"github.com/paulhankin/poker"
)

type Card struct {
	suit uint8
	rank uint8
}

// Deck is the rappresentation of a game session.
type Session struct {
	Board       [5]Card
	Hand        [2]Card
	Deck        deck.Deck
	}

func NewCard(suit uint8, rank uint8) (Card, error) {
	if (suit > 3 || rank == 0 || rank > 13) {
		return Card{}, fmt.Errorf("invalid card %d, %d",suit,rank)
	}

	return Card{
		suit: suit,
		rank: rank,
	}, nil
}

func (c Card) Suit() uint8 {
	return c.suit
}

func (c Card) Rank() uint8 {
	return c.rank
}


// Convert the raw input card with the following suit order: ♣clubs -> ♦diamonds -> ♥hearts -> ♠spades
func convertCard(rawCard int) (Card, error) {
	if rawCard > 52 || rawCard < 1 {
		return Card{}, errors.New("the card to convert have an invalid value")
	}

	suit := uint8(((rawCard-1) / 13))
	rank := uint8(((rawCard - 1) % 13) + 1)
	card, err := NewCard(suit, rank)
	if err != nil {
		return Card{}, err
	}
	return card, nil
}

// Evaluate the final hand and return the peer rank of the winner
func (s *Session) WinnerEval() ([]int, error) {

	playerNum := len(s.Deck.Peer.Addresses)

	var finalHand [7]poker.Card
	var err error
	for i:=0; i<5; i++ {
		finalHand[i], err = poker.MakeCard(poker.Suit(s.Board[i].suit),poker.Rank(s.Board[i].rank))
		if err != nil {
			return []int{}, err
		}
	}
	finalHand[5], err = poker.MakeCard(poker.Suit(s.Hand[0].suit),poker.Rank(s.Hand[0].rank))
	if err != nil {
		return []int{}, err
	}
	finalHand[6], err = poker.MakeCard(poker.Suit(s.Hand[1].suit),poker.Rank(s.Hand[1].rank)) 
	if err != nil {
		return []int{}, err
	}

	score := poker.Eval7(&finalHand)


	// Create a slice of player indexes
	players := make([]int, playerNum)
	for i := range players {
		players[i] = i
	}

	// Sort indexes based on scores
	sort.Slice(players, func(i, j int) bool {
		return scores[players[i]] > scores[players[j]]
	})

	sort.Slice(scores, func(i, j int) bool {
		return scores[i] < scores[j]
	})

	// Check for ties
	winner := []int{players[0]}
	for i := 0; i < len(scores); i++ {
		if scores[i] == scores[i+1] {
			winner = append(winner, players[i])
		} else {
			i = len(scores)
		}
	}

	return winner, nil
}
