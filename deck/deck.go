package deck

import (
	"github.com/luca-patrignani/mental-poker/common"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/suites"
)

type Deck struct {
	DeckSize       int
	CardCollection []kyber.Point //a
	EncryptedDeck []kyber.Point  //b
	peer         common.Peer
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

// func (d *Deck) allToAllMultiple(bufferSend []kyber.Point, root int) ([]kyber.Point, error){
	// if (d.peer.Rank)
	// dataSend, err := bufferSend[0].MarshalBinary()
	// if err != nil {
		// return suite.Point(), err
	// }
	// ataResponse, err := d.peer.AllToAll(dataH)
	// if err != nil {
		// return suite.Point(), err
	// }

	// hArray := make([]kyber.Point, len(d.peer.Addresses))
	// for i := 0; i < len(ataResponse); i++ {
		// hArray[i] = suite.Point()
		// err := hArray[i].UnmarshalBinary([]byte(ataResponse[i]))
		// if err != nil {
			// return suite.Point(), err
		// }
	// }

// }

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

func (d *Deck) broadcastMultiple(bufferSend []kyber.Point, root int) ([]kyber.Point, error) {
	dataSend := make([]byte, len(bufferSend))
	for i:=0; i < len(bufferSend); i++ {
		temp, err := bufferSend[i].MarshalBinary()
		if err != nil {
			return nil, err
		}
		dataSend[i] = temp
	}
	ataResponse, err := d.peer.Broadcast(dataSend,root)

}
