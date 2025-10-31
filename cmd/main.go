package main

import (
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"

	"github.com/luca-patrignani/mental-poker/consensus"
	"github.com/luca-patrignani/mental-poker/domain/poker"
	"github.com/luca-patrignani/mental-poker/ledger"
	"github.com/luca-patrignani/mental-poker/network"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <ip>\n", os.Args[0])
		os.Exit(1)
	}

	// Create a new slog handler with the default PTerm logger
	handler := pterm.NewSlogHandler(&pterm.DefaultLogger)

	// Create a new slog logger with the handler
	logger := slog.New(handler)

	pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithStyle("M", pterm.FgRed.ToStyle()),
		putils.LettersFromStringWithStyle("ental ", pterm.FgDarkGray.ToStyle()),
		putils.LettersFromStringWithStyle("P", pterm.FgRed.ToStyle()),
		putils.LettersFromStringWithStyle("oker", pterm.FgDarkGray.ToStyle()),
	).Render()

	// Create an interactive text input with single line input mode and show it
	name, _ := pterm.DefaultInteractiveTextInput.WithDefaultText("Enter your username").WithDefaultValue(" ").Show()

	// Print a blank line for better readability
	pterm.Println()

	// Print the user's answer with an info prefix
	pterm.Info.Printfln("Your username: %s", name)

	ip := os.Args[1]
	l, err := net.Listen("tcp", ip+":0")
	if err != nil {
		logger.Error("failed to listen on address", "address:"+ip, err.Error())
		panic(err)
	}
	info := "Listening on " + l.Addr().String()

	pterm.Info.Println(info)

	// Print two new lines as spacer.
	pterm.Print("\n")

	addresses := []string{l.Addr().String()}
	for {
		addr, _ := pterm.DefaultInteractiveTextInput.WithDefaultText("Enter his address and port in ipaddr:port format. When done, type done").WithDefaultValue("").Show()
		if addr == "done" {
			break
		}
		// Print a blank line for better readability
		pterm.Println()
		ipaddr, port, err := net.SplitHostPort(addr)
		if err != nil {
			logger.Error("invalid address format: " + addr + "\n error: " + err.Error())
			continue
		}
		localIp, _, err := net.SplitHostPort(l.Addr().String())
		if err != nil {
			panic(err)
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
	pterm.Success.Printfln("Succesfully connected with %d players", len(names)-1)
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
	if err := node.UpdatePeers(); err != nil {
		panic(err)
	}
	//area, _ := pterm.DefaultArea.Start()
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
		if err := postBlinds(&pokerManager, node, 5); err != nil {
			panic(err)
		}
		
		printState(pokerManager)
		for {
			var panel pterm.Panel
			if err := inputAction(pokerManager, *node, myRank); err != nil {
				logger.Error(err.Error())
				panic(err)
			}
			round := pokerManager.Session.Round
			if round == poker.Showdown {
				err := showCards(&pokerManager, &deck)
				if err != nil {
					logger.Error(err.Error())
				}
				panel, err = getWinnerPanel(pokerManager)
				if err != nil {
					logger.Error(err.Error())
				}
				printState(pokerManager, panel)
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
			//area.Update()
			printState(pokerManager)
		}
		logger.Info("Starting a new match")
		pokerManager.PrepareNextMatch()
	}
	//area.Stop()

}

func testConnections(p2p *network.P2P, name string) ([]string, error) {
	byteNames, err := p2p.AllToAllwithTimeout([]byte(name), 60*time.Second)
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, name := range byteNames {
		names = append(names, string(name))
	}
	return names, nil
}

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
		30*time.Second,
	)
	return network.NewP2P(&peer), myRank
}

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
	/*err = addBlind(psm, node, smallBlind*2)
	if err != nil {
		return err
	}*/
	return nil
}

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
		fmt.Printf("%s post blind with %d\n",psm.Session.Players[idx].Name,amount)
	} else {
		err := node.WaitForProposal()
		if err != nil {
			return err
		}
		fmt.Printf("Not %s post blind with %d\n",psm.Session.Players[idx].Name,amount)

	}
	return nil
}

func inputAction(pokerManager poker.PokerManager, consensusNode consensus.ConsensusNode, myRank int) error {
	actions := []string{"Fold", "Check", "Call", "Raise", "AllIn"}
	raiseAmount := "0"
	selectedAction := ""
	var action consensus.Action
	area, _ := pterm.DefaultArea.Start()
	if pokerManager.Session.CurrentTurn == uint(pokerManager.FindPlayerIndex(myRank)) {
		for {
			var err error
			selectedAction, _ = pterm.DefaultInteractiveSelect.WithDefaultText("Select your next action").WithOptions(actions).Show()
			if selectedAction == "Raise" {
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
			if val := pokerManager.Validate(action.Payload); val != nil {
				area.Update()
				pterm.Error.Printfln("Invalid action: %s", val.Error())
				continue
			}

			if confirm, _ := pterm.DefaultInteractiveConfirm.WithDefaultText(fmt.Sprintf("Confirm to %s?", selectedAction)).WithDefaultValue(true).Show(); confirm {
				break
			}
			area.Update()
			pterm.Info.Println("Action cancelled.")
		}
		area.Stop()
		err := action.Sign(consensusNode.GetPriv())
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

func getWinnerPanel(psm poker.PokerManager) (pterm.Panel, error) {
	winners, err := psm.GetWinners()
	if err != nil {
		return pterm.Panel{}, err
	}
	pbox := pterm.DefaultBox.WithHorizontalPadding(4).WithTopPadding(1).WithBottomPadding(1)
	infoString := ""
	if len(winners) == 1 {
		var id int
		finalAmount := 0
		for winner, amount := range winners {
			id = winner
			finalAmount += int(amount)
		}
		info, err := printSingleWinnerInfo(psm, id, finalAmount)
		if err != nil {
			return pterm.Panel{}, err
		}
		infoString += info
	} else {
		for winner, amount := range winners {
			info, err := printSingleWinnerInfo(psm, winner, int(amount))
			if err != nil {
				return pterm.Panel{}, err
			}
			infoString += info
		}
	}
	return pterm.Panel{Data: pbox.WithTitle(pterm.LightGreen("|SHOWDOWN|")).WithTitleTopCenter().Sprintf(infoString)}, nil
}

func printSingleWinnerInfo(psm poker.PokerManager, id int, amount int) (string, error) {
	s := psm.GetSession()
	idx := psm.FindPlayerIndex(id)
	p := s.Players[idx]
	hand, err := s.DescribeHand(idx)
	if err != nil {
		return "", err
	}
	playerString := pterm.Sprintfln("%s won %d with %s", pterm.LightCyan(p.Name), amount, hand)
	return playerString, nil
}

func printState(psm poker.PokerManager, additionalPanel ...pterm.Panel) {
	s := psm.GetSession()
	var panels []pterm.Panel
	var mainPlayer pterm.Panel
	for _, p := range s.Players {
		if p.Id != psm.Player {
			pInfo := printPlayerInfo(p)
			panel := pterm.Panel{Data: pInfo}
			panels = append(panels, panel)
		} else {

			mainPlayer = pterm.Panel{Data: printMainInfo(p)}
		}
	}
	board := pterm.Panel{Data: pterm.DefaultHeader.WithBackgroundStyle(pterm.BgGreen.ToStyle()).Sprintf(printBoardInfo(s.Board[:], psm.Session.Round, s.Pots))}
	dashboard := []pterm.Panel{mainPlayer}
	dashboard = append(dashboard, additionalPanel...)

	pterm.DefaultPanel.WithPanels([][]pterm.Panel{
		panels,
		{board},
		dashboard,
	}).Render()
}

func printPlayerInfo(p poker.Player) string {
	pbox := pterm.DefaultBox.WithHorizontalPadding(4).WithTopPadding(1).WithBottomPadding(1)
	var active string
	if p.HasFolded {
		active = pterm.LightRed("Folded")
	} else {
		active = pterm.LightGreen("Active")
	}
	return pbox.WithTitle(p.Name).WithTitleTopLeft().Sprintf("Current Bet: %d\nBankroll: %d\n%s - %s\n%s", p.Bet, p.Pot, p.Hand[0].String(), p.Hand[1].String(), active)
}

func printMainInfo(p poker.Player) string {
	pbox := pterm.DefaultBox.WithHorizontalPadding(10).WithTopPadding(1).WithBottomPadding(1)
	var active string
	if p.HasFolded {
		active = pterm.LightRed("Folded")
	} else {
		active = pterm.LightGreen("Active")
	}
	return pbox.WithTitle(p.Name).WithTitleTopLeft().Sprintf("Current Bet: %d\nBankroll: %d\n%s - %s\n%s", p.Bet, p.Pot, p.Hand[0].String(), p.Hand[1].String(), active)
}

func printBoardInfo(b []poker.Card, round poker.Round, pots []poker.Pot) string {
	board := ""
	for _, c := range b {
		board += c.String() + " - "
	}
	for i, p := range pots {
		board += " Pot" + strconv.Itoa(i) + ": " + strconv.Itoa(int(p.Amount)) + " | "
	}

	return board + string(round)
}
