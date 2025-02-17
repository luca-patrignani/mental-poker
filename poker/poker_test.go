package poker

import (
	//"errors"
	//"fmt"
	"testing"
	"time"

	"github.com/luca-patrignani/mental-poker/common"
	"github.com/luca-patrignani/mental-poker/deck"
	"github.com/paulhankin/poker"
)

// TODO: make more elaborate test
func TestConvertCard(t *testing.T) {
	expctCard, _ := poker.MakeCard(poker.Heart, 2)
	testCard, err := convertCard(28)
	if err != nil {
		t.Fatal(err)
	}
	if !testCard.Valid() {
		t.Fatal("card not valid")
	}
	if testCard != expctCard {
		errString := "expected " + expctCard.String() + ", get " + testCard.String()
		t.Fatal(errString)
	}

}
func TestWinnerEval(t *testing.T) {
	n := 10
	listeners, addresses := common.CreateListeners(n)
	errChan := make(chan error)
	winChan := make(chan []int, 10)
	for i := 0; i < n; i++ {
		go func() {
			deck := deck.Deck{
				DeckSize: 52,
				Peer: common.NewPeer(i, addresses, listeners[i], time.Hour),
			}
			session := Session{
				Board: [5]poker.Card{},
				Hand:  [2]poker.Card{},
				Deck:  deck,
			}
			defer deck.Peer.Close()
			err := deck.PrepareDeck()
			if err != nil {
				errChan <- err
				return
			}
			err = deck.Shuffle()
			if err != nil {
				errChan <- err
				return
			}

			for drawer := 0; drawer < n; drawer++ {
				cardA, err := deck.DrawCard(drawer)
				if err != nil {
					errChan <- err
					return
				}
				cardB, err := deck.DrawCard(drawer)
				if err != nil {
					errChan <- err
					return
				}
				if i == drawer {
					cardConvA, err := convertCard(cardA)
					if err != nil {
						errChan <- err
						return
					}
					cardConvB, err := convertCard(cardB)
					if err != nil {
						errChan <- err
						return
					}
					session.Hand[0] = cardConvA
					session.Hand[1] = cardConvB
					t.Logf("Player %d got %s and %s", drawer, cardConvA.String(), cardConvB.String())
				}
			}
			drawer := 0
			for board := 0; board < 5; board++ {
				card, err := deck.DrawCard(drawer)
				if err != nil && i == drawer {
					errChan <- err
					return
				}
				card, err = deck.OpenCard(0, card)
				if err != nil {
					errChan <- err
					return
				}
				cardRev, err := convertCard(card)
				if err != nil {
					errChan <- err
					return
				}
				session.Board[board] = cardRev
				t.Logf("board for player %d\n%s", i, append(session.Board[:], session.Hand[:]...))
			}
			winner, err := session.WinnerEval()
			if err != nil {
				errChan <- err
				return
			}
			winChan <- winner[:]
			//var finalHand [7]poker.Card
			//copy(finalHand[:5],session.Board[:])
			//copy(finalHand[5:],session.Hand[:])
			//score := poker.Eval7(&finalHand)
		}()
	}

	for i := 0; i < n; i++ {
		err := <-errChan
		if err != nil {
			t.Fatal(err)
		}
	}
	//winner := <-winChan
	//for i := 1; i < n; i++ {
	//	win := <-winChan
	//	for j :=
	//	if winner != win {
	//		t.Fatal(, c)
	//	}
	//}

}
