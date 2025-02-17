package deck

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/luca-patrignani/mental-poker/common"
	"go.dedis.ch/kyber/v4"
	"go.dedis.ch/kyber/v4/suites"
)

// Deck is the rappresentation of a game session.
type Deck struct {
	DeckSize       int
	CardCollection []kyber.Point //(a) The index of the array rappresent the value of the card.
	EncryptedDeck  []kyber.Point //(b)
	SecretKey      kyber.Scalar  //(x_j)
	lastDrawnCard  int
	Peer           common.Peer
}

var suite suites.Suite = suites.MustFind("Ed25519")

// Protocol 1: Deck Preparation
// Generate the deck as a set of encrypted values in a cyclic group
func (d *Deck) PrepareDeck() error {
	// Initialize deck
	deck := make([]kyber.Point, d.DeckSize+1)
	// Generate encrypted values for each card
	for i := 0; i <= d.DeckSize; i++ {
		card, err := d.generateRandomElement() // Encrypt card as a^(i)http
		if err != nil {
			return err
		}
		deck[i] = card
	}
	d.CardCollection = deck
	d.lastDrawnCard = 0
	return nil
}

// Protocol 2: Generate Random Element
// Generation of a random element in a distributed way to ensure secretness
func (d *Deck) generateRandomElement() (kyber.Point, error) {
	// initialize random generator of cyclic group G
	gj := suite.Point().Mul(suite.Scalar().Pick(suite.RandomStream()), nil)
	hj := suite.Point().Mul(suite.Scalar().Pick(suite.RandomStream()), nil)

	for gj.Equal(hj) {
		hj = suite.Point().Mul(suite.Scalar().Pick(suite.RandomStream()), nil)
	}

	lambda := suite.Scalar().Pick(suite.RandomStream()) // random lambda 0 < lambda < n

	gPrime := suite.Point().Mul(lambda, gj)

	_, err := d.allToAllSingle(gj)
	if err != nil {
		return nil, err
	}
	_, err = d.allToAllSingle(gPrime)
	if err != nil {
		return nil, err
	}
	_, err = d.allToAllSingle(hj)
	if err != nil {
		return nil, err
	}

	hPrime := suite.Point().Mul(lambda, hj)
	hArray, err := d.allToAllSingle(hPrime)
	if err != nil {
		return nil, err
	}

	//TODO: ZKA (optional)

	hResult := hArray[0]
	for i := 1; i < len(hArray); i++ {
		hResult.Add(hResult, hArray[i])
	}

	return hResult, nil
}

// Protocol 5: Card drawing
// It returns a face up card to the drawer player.
func (d *Deck) DrawCard(drawer int) (int, error) {
	d.lastDrawnCard++
	cj := d.EncryptedDeck[d.lastDrawnCard].Clone()
	for j := 0; j < len(d.Peer.Addresses); j++ {
		if j != drawer {
			xj_1 := suite.Scalar().Inv(d.SecretKey)
			cj.Mul(xj_1, cj)
		}
		var err error
		cj, err = d.broadcastSingle(cj, j)
		if err != nil {
			return 0, err
		}
		// if j != drawer {
		// 	// ZKA
		// }
	}
	if d.Peer.Rank != drawer {
		return 0, nil
	}
	xj_1 := suite.Scalar().Inv(d.SecretKey)
	cj.Mul(xj_1, cj)
	for i := 1; i <= d.DeckSize; i++ {
		if d.CardCollection[i].Equal(cj) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("card drawn not found")
}

// Protocol 6: Card Opening
// the player with rank player shows its card.
func (d *Deck) OpenCard(player int, card int) (int, error) {
	recv, err := d.Peer.Broadcast([]byte(strconv.Itoa(card)), player)
	if err != nil {
		return 0, err
	}
	cardRecv, err := strconv.Atoi(string(recv))
	if err != nil {
		return 0, err
	}
	return cardRecv, nil
}

// Broadcast of a single card from all the source
func (d *Deck) allToAllSingle(bufferSend kyber.Point) ([]kyber.Point, error) {
	dataSend, err := bufferSend.MarshalBinary()
	if err != nil {
		return nil, err
	}
	ataResponse, err := d.Peer.AllToAll(dataSend)
	if err != nil {
		return nil, err
	}

	dataReceived := make([]kyber.Point, len(d.Peer.Addresses))
	for i := 0; i < len(d.Peer.Addresses); i++ {
		dataReceived[i] = suite.Point()
		err := dataReceived[i].UnmarshalBinary([]byte(ataResponse[i]))
		if err != nil {
			return nil, err
		}
	}
	return dataReceived, nil
}

// Broadcast of multiple card from a single source
func (d *Deck) broadcastMultiple(bufferSend []kyber.Point, root int, size int) ([]kyber.Point, error) {
	var jsonData []byte
	if d.Peer.Rank == root {
		dataSend := make([][]byte, len(bufferSend))
		for i := 0; i < size; i++ {
			temp, err := bufferSend[i].MarshalBinary()
			if err != nil {
				return nil, err
			}
			dataSend[i] = temp
		}
		var err error
		jsonData, err = json.Marshal(dataSend)
		if err != nil {
			return nil, err
		}
	}
	dataRecv, err := d.Peer.Broadcast(jsonData, root)
	if err != nil {
		return nil, err
	}
	bufferRecv := make([][]byte, size)
	err = json.Unmarshal(dataRecv, &bufferRecv)
	if err != nil {
		return nil, err
	}

	pointsRecv := make([]kyber.Point, size)

	for i := 0; i < size; i++ {
		pointsRecv[i] = suite.Point()
		err := pointsRecv[i].UnmarshalBinary(bufferRecv[i])
		if err != nil {
			return nil, err
		}
	}

	return pointsRecv, nil
}

// Broadcast of a single card from a single source
func (d *Deck) broadcastSingle(bufferSend kyber.Point, root int) (kyber.Point, error) {
	bufferRecv, err := d.broadcastMultiple([]kyber.Point{bufferSend}, root, 1)
	if err != nil {
		return nil, err
	}
	if len(bufferRecv) != 1 {
		return nil, fmt.Errorf("bufferRecv must be of size 1")
	}
	return bufferRecv[0], nil
}
