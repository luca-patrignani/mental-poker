package main

import (
	"crypto/ed25519"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"slices"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"

	"github.com/luca-patrignani/mental-poker/v2/consensus"
	"github.com/luca-patrignani/mental-poker/v2/domain/poker"
	"github.com/luca-patrignani/mental-poker/v2/ledger"
	"github.com/luca-patrignani/mental-poker/v2/network"
)

var timeout = 30 * time.Second

const defaultPort = 53550
const discoveryPort = 53551

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

	pinger, err := NewPinger(
		Info{
			Name: name,
			Address: l.Addr().String(),
		},
		time.Second,
	)
	if err != nil {
		panic(err)
	}
	pinger.Start()
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
	if err := pinger.Close(); err != nil {
		panic(err)
	}
	for info, lastPing := range pinger.PlayersStatus() {
		pterm.Info.Printfln("Discovered player %s at address %s at time %s", info.Name, info.Address, lastPing.String())
		if !slices.Contains(addresses, info.Address) {
			addresses = append(addresses, info.Address)
		}
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
			Name:     string(names[i]),
			Id:       i,
			Hand:     [2]poker.Card{card, card},
			BankRoll: 1000,
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
				if !session.EverybodyFolded() {
					if session.OnePlayerRemained() {
						err := showdownByAllIn(&pokerManager, &deck)
						if err != nil {
							logger.Error(err.Error())
						}
					}
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
		pokerManager.PrepareNextMatch()
		if pokerManager.Session.OnePlayerRemained() {
			if pokerManager.Session.Players[myRank].BankRoll > 0 {
				pterm.Success.Println("You are the last player remaining, Congratulations!")
				break
			} else {
				pterm.Error.Println("You have been eliminated, better luck next time!")
				break
			}
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
	}

	area.Stop()
	pterm.Println("Thank you for playing...")
	pterm.Print(title)
	pterm.Println()

}

// testConnections verifies network connectivity with all peers by exchanging names.
// This is called during initialization to ensure all players can communicate.
//
// Parameters:
//   - p2p: Network layer for peer-to-peer communication
//   - name: This player's username to broadcast
//
// Returns a slice of all player names in rank order, or an error if communication fails.
func testConnections(p2p *network.P2P, name string) ([]string, error) {
	byteNames, err := p2p.AllToAll([]byte(name))
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, name := range byteNames {
		names = append(names, string(name))
	}
	return names, nil
}

// createP2P creates the P2P network and determines the rank of the current player
// by sorting the addresses lexicographically. Lower addresses get lower ranks.
//
// Parameters:
//   - addresses: All player addresses (including local)
//   - l: Local network listener
//
// Returns the initialized P2P network and this player's rank.
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

// distributeHands deals two cards to each player using the mental poker protocol.
// Cards are drawn secretly and remain encrypted until revealed.
//
// Parameters:
//   - psm: Poker manager containing session state
//   - deck: Mental poker deck to draw from
//
// Returns an error if card distribution fails.
func distributeHands(psm *poker.PokerManager, deck *poker.PokerDeck) error {
	c, _ := poker.NewCard(0, 0)
	for i, p := range psm.Session.Players {
		if p.BankRoll <= 0 {

			psm.Session.Players[i].Hand[0] = c
			psm.Session.Players[i].Hand[1] = c
			continue
		}
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

// showCards reveals all players' hole cards using the mental poker protocol.
// This is called at showdown when cards must be compared.
//
// Parameters:
//   - psm: Poker manager containing session state
//   - deck: Mental poker deck containing the cards
//
// Returns an error if card revealing fails.
func showCards(psm *poker.PokerManager, deck *poker.PokerDeck) error {
	for i, p := range psm.Session.Players {
		if !p.HasFolded || p.BankRoll > 0 {
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
	}
	return nil
}

// cardOnBoard reveals a community card at the specified board position.
// The card is drawn and immediately revealed to all players.
//
// Parameters:
//   - psm: Poker manager containing session state
//   - deck: Mental poker deck to draw from
//   - idx: Board position (0-4) to place the card
//
// Returns an error if drawing or revealing fails.
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

// postBlinds posts the small and big blinds for the current hand.
// The two players after the dealer post blinds automatically.
//
// Parameters:
//   - psm: Poker manager containing session state
//   - node: Consensus node for proposing blind actions
//   - smallBlind: Amount for the small blind (big blind is 2x)
//
// Returns an error if posting blinds fails or there aren't enough players.
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

// addBlind posts a single blind for the current player if it's their turn.
// Automatically folds if the player lacks sufficient chips.
//
// Parameters:
//   - psm: Poker manager containing session state
//   - node: Consensus node for proposing the action
//   - amount: Blind amount to post
//
// Returns an error if the action fails.
func addBlind(psm *poker.PokerManager, node *consensus.ConsensusNode, amount uint) error {
	idx := psm.FindPlayerIndex(psm.Player)

	if idx == psm.GetCurrentPlayer() {
		var action consensus.Action
		var err error
		if psm.Session.Players[idx].BankRoll < amount {
			action, err = consensus.MakeAction(psm.Player, psm.ActionFold())
		} else {
			action, err = consensus.MakeAction(psm.Player, psm.ActionBet(amount))
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

// inputAction handles player input for their poker action with a timeout.
// If the player doesn't act before the timeout, automatically checks or folds.
//
// Parameters:
//   - pokerManager: Poker manager containing session state
//   - consensusNode: Consensus node for proposing actions
//   - myRank: This player's rank
//
// Returns an error if the action fails.
func inputAction(pokerManager poker.PokerManager, consensusNode consensus.ConsensusNode, myRank int) error {
	var timedOut uint32 = 0 // use atomic access to avoid races
	isPlayerTurn := pokerManager.Session.CurrentTurn == uint(pokerManager.FindPlayerIndex(myRank))

	duration := timeout - 5*time.Second
	if duration < 0 {
		duration = 9999 * time.Second
	}
	if isPlayerTurn {

		done := make(chan struct{})
		ticker := time.NewTicker(duration)

		// ensure goroutine is cancelled when this function returns
		defer func() {
			close(done)
			ticker.Stop()
		}()

		go func() {
			for {
				select {
				case <-ticker.C:
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
		text := "Waiting for the other player to make an action ..."
		if pokerManager.GetCurrentPlayer() >= 0 {
			currentName := pterm.LightCyan(pokerManager.GetSession().Players[pokerManager.GetCurrentPlayer()].Name)
			text = pterm.Sprintf("Waiting for %s to make an action ...", currentName)
		}
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

// applyShowdown executes the showdown action for the current player, distributing
// pots to winners and advancing the game state.
//
// Parameters:
//   - psm: Poker manager containing session state
//   - node: Consensus node for proposing the showdown
//   - myRank: This player's rank
//
// Returns an error if the showdown fails.
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

func showdownByAllIn(psm *poker.PokerManager, deck *poker.PokerDeck) error {
	for i, c := range psm.Session.Board {
		if c.Rank() == 0 {
			if err := cardOnBoard(psm, deck, i); err != nil {
				return err
			}
		}
	}
	return nil
}

// askForLeavers prompts all players whether they want to continue playing or leave.
// Players who leave are removed from the game through consensus.
//
// Parameters:
//   - psm: Poker manager containing session state
//   - node: Consensus node for coordinating departures
//   - deck: Mental poker deck for cleanup
//   - p2p: Network layer for coordinating with peers
//
// Returns whether this player is leaving, names of players who left, and any error.
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
