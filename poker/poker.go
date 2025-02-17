package poker

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sort"

	"github.com/luca-patrignani/mental-poker/deck"
	"github.com/paulhankin/poker"
)

// Deck is the rappresentation of a game session.
type Session struct {
	Board       [5]poker.Card
	Hand        [2]poker.Card
	Deck        deck.Deck
	}

//TODO: Add interface for card, matching card struct of the package (non so come fare lucone :c)

// Convert the raw input card with the following suit order: ♣clubs -> ♦diamonds -> ♥hearts -> ♠spades
func convertCard(rawCard int) (poker.Card, error) {
	if rawCard > 52 || rawCard < 1 {
		return 0, errors.New("the card to convert have an invalid value")
	}

	suit := poker.Suit(uint8(((rawCard-1) / 13)))
	rank := poker.Rank(((rawCard - 1) % 13) + 1)
	card, err := poker.MakeCard(suit, rank)
	if err != nil {
		return 0, err
	}
	return card, nil
}

// Evaluate the final hand and return the peer rank of the winner
func (hb *Session) WinnerEval() ([]int, error) {

	playerNum := len(hb.Deck.Peer.Addresses)

	var finalHand [7]poker.Card
	copy(finalHand[:5], hb.Board[:])
	copy(finalHand[5:], hb.Hand[:])
	score := poker.Eval7(&finalHand)

	//Marshall the data
	bufferSend := new(bytes.Buffer)
	binary.Write(bufferSend, binary.BigEndian, score)

	byteScores, err := hb.Deck.Peer.AllToAll(bufferSend.Bytes())
	if err != nil {
		return []int{-1}, err
	}
	//Unmarshal the data
	var scores []int16
	for i := 0; i < len(byteScores); i++ {
		scores[i] = int16(binary.BigEndian.Uint16(byteScores[i]))
	}

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
