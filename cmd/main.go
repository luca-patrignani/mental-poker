package main

import (
	"crypto/ed25519"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"

	"github.com/luca-patrignani/mental-poker/consensus"
	"github.com/luca-patrignani/mental-poker/domain/poker"
	"github.com/luca-patrignani/mental-poker/ledger"
	"github.com/luca-patrignani/mental-poker/network"
)


var timeout = 30 * time.Second

const defaultPort = 53550

func main() {
	timeoutFlag := flag.Uint("timeout", 30, "timeout in seconds")
	portFlag := flag.Uint("port", defaultPort, "port to listen on")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: %s <ip> [OPTIONS] %v\n", os.Args[0], os.Args)
		os.Exit(1)
	}

	timeout = time.Duration(*timeoutFlag) * time.Second
	port := *portFlag

	ip := flag.Arg(0)

	// Create a new slog handler with the default PTerm logger
	handler := pterm.NewSlogHandler(&pterm.DefaultLogger)

	// Create a new slog logger with the handler
	logger := slog.New(handler)
	pterm.Print("\n")

	title, err := pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithStyle("M", pterm.FgRed.ToStyle()),
		putils.LettersFromStringWithStyle("ental ", pterm.FgDarkGray.ToStyle()),
		putils.LettersFromStringWithStyle("P", pterm.FgRed.ToStyle()),
		putils.LettersFromStringWithStyle("oker", pterm.FgDarkGray.ToStyle()),
	).Srender()
	if err != nil {
		logger.Error(err.Error())
	}
	pterm.Print(title)
	// Create an interactive text input with single line input mode and show it
	name, _ := pterm.DefaultInteractiveTextInput.WithDefaultText("Enter your username").WithDefaultValue(" ").Show()

	// Print a blank line for better readability
	pterm.Println()

	// Print the user's answer with an info prefix
	pterm.Info.Printfln("Your username: %s", name)
	info := "Listening on "
	localIp := ""
	l, err := net.Listen("tcp", ip+":"+strconv.Itoa(int(port)))
	if err != nil {
		logger.Warn(err.Error())
		var fatalErr error
		l, fatalErr = net.Listen("tcp", ip+":0")
		if fatalErr != nil {
			panic(err)
		}
		log := fmt.Sprintf("New port choosen for listening: %s", l.Addr().String())
		logger.Info(log)
		localIp = l.Addr().String()
		info += localIp
	} else {
		localIp, _, err = net.SplitHostPort(l.Addr().String())
		if err != nil {
			panic(err)
		}
		info += localIp
	}

	pterm.Info.Println(info)

	// Print two new lines as spacer.
	pterm.Print("\n")

	addresses := []string{l.Addr().String()}
	for {
		addr, _ := pterm.DefaultInteractiveTextInput.
			WithDefaultText("Enter the last number of the addresses of the players separated by Enter. After that, type done").
			WithDefaultValue("").Show()

		if addr == "done" {
			break
		}
		// Print a blank line for better readability
		pterm.Println()
		localIp, _, err := net.SplitHostPort(l.Addr().String())
		if err != nil {
			panic(err)
		}
		ipaddr, port, err := splitHostPort(addr, defaultPort)
		if err != nil {
			logger.Error("invalid address format: " + addr + "\n error: " + err.Error())
			continue
		}

		guessedAddr, err := guessIpAddress(net.ParseIP(localIp), ipaddr)
		if err != nil {
			logger.Error("could not guess address for: " + addr + "\n error: " + err.Error())
			continue
		}
		tcpAddr, err := net.ResolveTCPAddr("tcp", guessedAddr.String()+":"+port)
		if err != nil {
			errMsg := "invalid address:" + addr + "\n error: " + err.Error()
			logger.Error(errMsg)
			continue
		}
		addresses = append(addresses, guessedAddr.String()+":"+strconv.Itoa(tcpAddr.Port))
	}
	p2p, myRank := createP2P(addresses, l)
	pterm.Info.Printfln("Your rank is %d\n", myRank)
	spinner, _ := pterm.DefaultSpinner.Start("Trying to establish the connections with the other players...")

	names, err := testConnections(p2p, name)
	if err != nil {
		spinner.Fail()
		panic(err)
	}
	spinner.Success()
	pterm.Success.Printfln("Succesfully discovered with %d players", len(names)-1)
	for i, name := range names {
		msg := fmt.Sprintf(" %s: %s", p2p.GetAddresses()[i], string(name))
		logger.Info(msg)
	}
	card, _ := poker.NewCard(0, 0)
	players := make([]poker.Player, len(names))
	for i := range names {
		players[i] = poker.Player{
			Name: string(names[i]),
			Id:   i,
			Hand: [2]poker.Card{card, card},
			Pot:  1000,
		}
	}
	deck := poker.NewPokerDeck(p2p)
	err = deck.PrepareDeck()
	if err != nil {
		panic(err)
	}
	session := poker.Session{
		Board:       [5]poker.Card{},
		Players:     players,
		Round:       poker.PreFlop,
		HighestBet:  0,
		Dealer:      0,
		CurrentTurn: 1,
	}

	blockchain, err := ledger.NewBlockchain(session)
	if err != nil {
		panic(err)
	}
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	pokerManager := poker.PokerManager{
		Session: &session,
		Player:  myRank,
	}
	node := consensus.NewConsensusNode(
		pub, priv,
		map[int]ed25519.PublicKey{myRank: pub},
		&pokerManager,
		blockchain,
		p2p,
	)
	spinner, _ = pterm.DefaultSpinner.Start("Exchanging keys with the other players...")

	if err := node.UpdatePeers(); err != nil {
		spinner.Fail()
		panic(err)
	}
	spinner.Success()

	area, _ := pterm.DefaultArea.Start()
	for {
		spinner, _ := pterm.DefaultSpinner.Start("Shuffling the cards ...")

		if err := deck.Shuffle(); err != nil {
			spinner.Fail()
			panic(err)
		}
		spinner.Success()

		spinner, _ = pterm.DefaultSpinner.Start("Distribute hand cards ...")

		if err := distributeHands(&pokerManager, &deck); err != nil {
			spinner.Fail()
			panic(err)
		}
		spinner.Success()
		spinner, _ = pterm.DefaultSpinner.Start("Posting blinds ...")
		if err := postBlinds(&pokerManager, node, 5); err != nil {
			spinner.Fail()
			panic(err)
		}
		spinner.Success()

		printState(pokerManager)
		for {
			var panel pterm.Panel
			if err := inputAction(pokerManager, *node, myRank); err != nil {
				logger.Error(err.Error())
				panic(err)
			}
			b, err := blockchain.GetLatest()
			if err != nil {
				logger.Error(err.Error())
			}
			actionPanel := getActionPanel(b.Action, pokerManager)

			round := pokerManager.Session.Round
			if round == poker.Showdown {
				if !session.OnePlayerRemained() {
					err := showCards(&pokerManager, &deck)
					if err != nil {
						logger.Error(err.Error())
					}
				}
				panel, err = getWinnerPanel(pokerManager)
				if err != nil {
					logger.Error(err.Error())
				}
				area.Update()
				printState(pokerManager, panel, actionPanel)
				if err := applyShowdown(pokerManager, *node, myRank); err != nil {
					panic(err)
				}
				break
			}

			if round == poker.Flop && pokerManager.Session.Board[0].Rank() == 0 {
				err := cardOnBoard(&pokerManager, &deck, 0)
				if err != nil {
					panic(err)
				}
				err = cardOnBoard(&pokerManager, &deck, 1)
				if err != nil {
					panic(err)
				}
				err = cardOnBoard(&pokerManager, &deck, 2)
				if err != nil {
					panic(err)
				}
			}
			if round == poker.Turn && pokerManager.Session.Board[3].Rank() == 0 {
				err := cardOnBoard(&pokerManager, &deck, 3)
				if err != nil {
					panic(err)
				}
			}
			if round == poker.River && pokerManager.Session.Board[4].Rank() == 0 {
				err := cardOnBoard(&pokerManager, &deck, 4)
				if err != nil {
					panic(err)
				}
			}
			area.Update()
			printState(pokerManager, actionPanel)
		}
		leave, leaveList, err := askForLeavers(pokerManager, *node, deck, *p2p)
		if err != nil {
			panic(err)
		}
		for _, name := range leaveList {
			log := fmt.Sprintf("%s left the game", pterm.Cyan(name))
			logger.Warn(log)
		}
		if leave {
			break
		}
		pRemained := len(pokerManager.Session.Players)
		if pRemained <= 1 {
			if pRemained == 1 {
				pterm.Info.Printfln("Last player remained: %s", pokerManager.Session.Players[0].Name)
			}
			break
		}

		logger.Info("Starting a new match")
		pokerManager.PrepareNextMatch()
	}

	area.Stop()
	pterm.Println("Thank you for playing...")
	pterm.Print(title)

}

// Test the connections with all the other players by exchanging their names
// and return the list of names
func testConnections(p2p *network.P2P, name string) ([]string, error) {
	byteNames, err := p2p.AllToAllwithTimeout([]byte(name), timeout)
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, name := range byteNames {
		names = append(names, string(name))
	}
	return names, nil
}

// Create the P2P network and determine the rank of the current player
// by sorting the addresses
func createP2P(addresses []string, l net.Listener) (p2p *network.P2P, myRank int) {
	sort.Slice(addresses, func(i, j int) bool {
		return addresses[i] < addresses[j]
	})
	mapAddresses := make(map[int]string)
	for i, addr := range addresses {
		mapAddresses[i] = addr
		if mapAddresses[i] == l.Addr().String() {
			myRank = i
		}
	}
	peer := network.NewPeer(
		myRank,
		mapAddresses,
		l,
		timeout,
	)
	return network.NewP2P(&peer), myRank
}

// Distribute two cards to each player
func distributeHands(psm *poker.PokerManager, deck *poker.PokerDeck) error {
	for i := range psm.Session.Players {
		card1, err := deck.DrawCard(i)
		if err != nil {
			return err
		}
		psm.Session.Players[i].Hand[0] = *card1
		card2, err := deck.DrawCard(i)
		if err != nil {
			return err
		}
		psm.Session.Players[i].Hand[1] = *card2
	}
	return nil
}

// Show the cards of each player
func showCards(psm *poker.PokerManager, deck *poker.PokerDeck) error {
	for i := range psm.Session.Players {
		card1 := psm.Session.Players[i].Hand[0]
		card1, err := deck.OpenCard(i, &card1)
		if err != nil {
			return err
		}
		psm.Session.Players[i].Hand[0] = card1

		card2 := psm.Session.Players[i].Hand[1]
		card2, err = deck.OpenCard(i, &card2)
		if err != nil {
			return err
		}
		psm.Session.Players[i].Hand[1] = card2

	}
	return nil
}

// Open a card in idx position on the board for all players
func cardOnBoard(psm *poker.PokerManager, deck *poker.PokerDeck, idx int) error {
	card, err := deck.DrawCard(0)
	if err != nil {
		return err
	}
	openCard, err := deck.OpenCard(0, card)
	if err != nil {
		return err
	}
	psm.Session.Board[idx] = openCard
	return nil
}

// helper function to post small and big blinds
func postBlinds(psm *poker.PokerManager, node *consensus.ConsensusNode, smallBlind uint) error {
	if len(psm.Session.Players) < 2 {
		return fmt.Errorf("not enough players to post blinds, at least 2 players are required, got %d", len(psm.Session.Players))
	}
	err := addBlind(psm, node, smallBlind)
	if err != nil {
		return err
	}
	err = addBlind(psm, node, smallBlind*2)
	if err != nil {
		return err
	}
	return nil
}

// helper function to add a blind for the current player if it's his turn
func addBlind(psm *poker.PokerManager, node *consensus.ConsensusNode, amount uint) error {
	idx := psm.FindPlayerIndex(psm.Player)

	if idx == psm.GetCurrentPlayer() {
		var action consensus.Action
		var err error
		if psm.Session.Players[idx].Pot < amount {
			action, err = consensus.MakeAction(psm.Player, psm.ActionFold())
		} else {
			action, err = consensus.MakeAction(psm.Player, psm.ActionRaise(amount))
		}
		if err != nil {
			return err
		}
		err = action.Sign(node.GetPriv())
		if err != nil {
			return err
		}
		if err := node.ProposeAction(&action); err != nil {
			return err
		}
	} else {
		err := node.WaitForProposal()
		if err != nil {
			return err
		}
	}

	return nil
}

// Handle the input action from the user with a timeout
// If the user doesn't input an action before the timeout, a default action is proposed
func inputAction(pokerManager poker.PokerManager, consensusNode consensus.ConsensusNode, myRank int) error {
	var timedOut uint32 = 0 // use atomic access to avoid races
	isPlayerTurn := pokerManager.Session.CurrentTurn == uint(pokerManager.FindPlayerIndex(myRank))

	duration := timeout - 5*time.Second
	if duration <= 0 {
		duration = 1 * time.Second
	}
	if isPlayerTurn {
		deadline := time.Now().Add(duration)

		done := make(chan struct{})
		ticker := time.NewTicker(500 * time.Millisecond)

		// ensure goroutine is cancelled when this function returns
		defer func() {
			close(done)
			ticker.Stop()
		}()

		go func() {
			for {
				select {
				case <-ticker.C:
					if time.Now().After(deadline) {
						// mark timedOut in a race-free way; main goroutine can read this via
						// atomic.LoadUint32(&timedOut) == 1
						atomic.StoreUint32(&timedOut, 1)

						// Fallback automatic fold in case you also want the goroutine to propose:
						if isPlayerTurn {
							action, err := consensus.MakeAction(myRank, pokerManager.ActionCheck())
							if err != nil {
								panic(err)
							}
							if val := pokerManager.Validate(action.Payload); val != nil {
								action, err = consensus.MakeAction(myRank, pokerManager.ActionFold())
								if err != nil {
									panic(err)
								}
							}
							if err := action.Sign(consensusNode.GetPriv()); err != nil {
								panic(err)
							}
							err = consensusNode.ProposeAction(&action)
							if err != nil {
								panic(err)
							}
						}
						return
					}
				case <-done:
					return
				}
			}
		}()
	}
	actions := []string{"Fold", "Check", "Call", "Raise", "AllIn"}
	raiseAmount := "0"
	selectedAction := ""
	var action consensus.Action
	if isPlayerTurn {
		timeout := fmt.Sprintf("%d", duration/time.Second)
		text := pterm.Sprintf("Defaulting to Check/Fold in %s seconds ...", pterm.LightCyan(timeout))
		spinner, _ := pterm.DefaultSpinner.WithRemoveWhenDone(true).Start(text)
		area, _ := pterm.DefaultArea.Start()
		for {
			var err error
			selectedAction, _ = pterm.DefaultInteractiveSelect.WithDefaultText("Select your next action").WithOptions(actions).Show()
			if selectedAction == "Raise" {
				spinner.Stop()
				raiseAmount, _ = pterm.DefaultInteractiveTextInput.WithDefaultText("Enter the amount to raise").Show()
			}
			switch selectedAction {
			case "Fold":
				action, err = consensus.MakeAction(myRank, pokerManager.ActionFold())
			case "Check":
				action, err = consensus.MakeAction(myRank, pokerManager.ActionCheck())
			case "Call":
				action, err = consensus.MakeAction(myRank, pokerManager.ActionCall())
			case "Raise":
				raiseInt, _ := strconv.Atoi(raiseAmount)
				action, err = consensus.MakeAction(myRank, pokerManager.ActionRaise(uint(raiseInt)))
			case "AllIn":
				action, err = consensus.MakeAction(myRank, pokerManager.ActionAllIn())
			default:
				panic("unknown action")
			}
			if err != nil {
				area.Update()
				pterm.Error.Println("Error creating the action")
				continue
			}
			if timedOut := atomic.LoadUint32(&timedOut); timedOut == 1 {
				spinner.Stop()
				area.Update()
				pterm.Error.Println("Action timed out, defaulting to Check/Fold")
				return nil
			}
			if val := pokerManager.Validate(action.Payload); val != nil {
				area.Update()
				pterm.Error.Printfln("Invalid action: %s", val.Error())
				continue
			}
			spinner.Stop()
			if confirm, _ := pterm.DefaultInteractiveConfirm.WithDefaultText(fmt.Sprintf("Confirm to %s?", selectedAction)).WithDefaultValue(true).Show(); confirm {
				break
			}
			area.Update()
			pterm.Info.Println("Action cancelled.")
		}
		area.Stop()
		err := action.Sign(consensusNode.GetPriv())
		spinner.Stop()
		if err != nil {
			return err
		}
		return consensusNode.ProposeAction(&action)
	} else {
		currentName := pterm.LightCyan(pokerManager.GetSession().Players[pokerManager.GetCurrentPlayer()].Name)
		text := pterm.Sprintf("Waiting for %s to make an action ...", currentName)
		spinner, _ := pterm.DefaultSpinner.Start(text)
		err := consensusNode.WaitForProposal()
		if err != nil {
			spinner.Fail()
		} else {
			spinner.Success()
		}
		return err
	}
}

// apply the showdown action for the current player
func applyShowdown(psm poker.PokerManager, node consensus.ConsensusNode, myRank int) error {
	if psm.Session.CurrentTurn == uint(psm.FindPlayerIndex(myRank)) {
		action, err := consensus.MakeAction(psm.Player, psm.ActionShowdown())
		action.Sign(node.GetPriv())
		if err != nil {
			return err
		}
		if err := node.ProposeAction(&action); err != nil {
			return err
		}
	} else {
		err := node.WaitForProposal()
		if err != nil {
			return err
		}
	}
	return nil
}

// Ask all players if they want to leave the game or start a new round
// Return true if the current player wants to leave the game
// and the list of players that left the game
func askForLeavers(psm poker.PokerManager, node consensus.ConsensusNode, deck poker.PokerDeck, p2p network.P2P) (bool, []string, error) {
	area, _ := pterm.DefaultArea.Start()
	var playersThatLeft []string
	for {
		ready, _ := pterm.DefaultInteractiveConfirm.
			WithDefaultText("Ready for the next Round?").WithDefaultValue(true).Show()
		action := ""
		if ready {
			action = "start a new game?"
		} else {
			action = "leave the game?"
		}
		if confirm, _ := pterm.DefaultInteractiveConfirm.
			WithDefaultText(fmt.Sprintf("Confirm to %s", action)).WithDefaultValue(true).Show(); confirm {
			spinner, _ := pterm.DefaultSpinner.Start("Waiting for the other player to choose...")
			var response [][]byte
			var err error
			if ready {
				response, err = p2p.AllToAll([]byte{byte(1)})
			} else {
				response, err = p2p.AllToAll([]byte{byte(0)})
			}
			if err != nil {
				spinner.Fail()
				return true, []string{}, err
			}
			for i, r := range response {
				if len(psm.Session.Players) <= 1 {
					return true, playersThatLeft, nil
				}
				if psm.FindPlayerIndex(i) != -1 {
					if r[0] == byte(0) {
						err = deck.LeaveGame(i)
						if err != nil {
							spinner.Fail()
							return true, []string{}, err
						}
						node.RemoveNode(i)
						p2p.RemovePeer(i)
						p, err := psm.RemoveByID(i)
						if err != nil {
							spinner.Fail()
							return true, []string{}, err
						}
						playersThatLeft = append(playersThatLeft, p.Name)
						if i == p2p.GetRank() {
							err := p2p.Close()
							if err != nil {
								spinner.Fail()
								return true, []string{}, err
							}
						}
					}
				}
			}
			spinner.Success()
			break
		}
		area.Update()
		pterm.Info.Println("Action cancelled.")
	}

	return false, playersThatLeft, nil
}
