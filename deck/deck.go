package deck

import (
	"encoding/json"
	"fmt"

	"github.com/luca-patrignani/mental-poker/common"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/suites"
)

type Deck struct {
	DeckSize       int
	CardCollection []kyber.Point //a
	EncryptedDeck  []kyber.Point //b
	peer           common.Peer
}

var suite suites.Suite = suites.MustFind("Ed25519")

// Protocol 1: Deck Preparation
// Generate the deck as a set of encrypted values in a cyclic group
func (d *Deck) PrepareDeck() ([]kyber.Point, error) {
	// Initialize deck
	deck := make([]kyber.Point, d.DeckSize)
	// Generate encrypted values for each card
	for i := 0; i <= d.DeckSize; i++ {
		card, err := d.generateRandomElement() // Encrypt card as a^(i)
		if err != nil {
			return deck, err
		}
		deck[i] = card
	}

	return deck, nil
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

	dataG, err := gPrime.MarshalBinary()
	if err != nil {
		return suite.Point(), err
	}
	// TODO: remove _ once done and remove string()
	gArray, err := d.peer.AllToAll(dataG)
	_ = gArray
	if err != nil {
		return suite.Point(), err
	}

	hPrime := suite.Point().Mul(lambda, hj)
	dataH, err := hPrime.MarshalBinary()
	if err != nil {
		return suite.Point(), err
	}
	ataResponse, err := d.peer.AllToAll(dataH)
	if err != nil {
		return suite.Point(), err
	}

	hArray := make([]kyber.Point, len(d.peer.Addresses))
	for i := 0; i < len(ataResponse); i++ {
		hArray[i] = suite.Point()
		err := hArray[i].UnmarshalBinary([]byte(ataResponse[i]))
		if err != nil {
			return suite.Point(), err
		}
	}

	//TODO: ZKA (optional)

	hResult := hArray[0]
	for i := 1; i < len(hArray); i++ {
		hResult.Add(hResult, hArray[i])
	}

	return hResult, nil
}

func (d *Deck) allToAllSingle(bufferSend kyber.Point) ([]kyber.Point, error) {
	dataSend, err := bufferSend.MarshalBinary()
	if err != nil {
		return nil, err
	}
	ataResponse, err := d.peer.AllToAll(dataSend)
	if err != nil {
		return nil, err
	}

	dataReceived := make([]kyber.Point, len(d.peer.Addresses))
	for i := 0; i < len(d.peer.Addresses); i++ {
		dataReceived[i] = suite.Point()
		err := dataReceived[i].UnmarshalBinary([]byte(ataResponse[i]))
		if err != nil {
			return nil, err
		}
	}
	return dataReceived, nil
}

func (d *Deck) broadcastMultiple(bufferSend []kyber.Point, root int, size int) ([]kyber.Point, error) {
	var jsonData []byte
	if d.peer.Rank == root {
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
	dataRecv, err := d.peer.Broadcast(jsonData, root)
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
