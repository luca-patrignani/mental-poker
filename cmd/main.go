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
		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			errMsg := "invalid address:" + addr + "\n error: " + err.Error()
			logger.Error(errMsg)
			continue
		}
		addresses = append(addresses, tcpAddr.String())
	}
	sort.Slice(addresses, func(i, j int) bool {
		return addresses[i] < addresses[j]
	})
	var myRank int
	mapAddresses := make(map[int]string)
	for i, addr := range addresses {
		mapAddresses[i] = addr
		if mapAddresses[i] == l.Addr().String() {
			pterm.Info.Printfln("Your rank is %d\n", i)
			myRank = i
		}
	}
	peer := network.NewPeer(
		myRank,
		mapAddresses,
		l,
		30*time.Second,
	)

	spinner, _ := pterm.DefaultSpinner.Start("Trying to establish a connnections with the other players...")

	p2p := network.NewP2P(&peer)
	names, err := p2p.AllToAllwithTimeout([]byte(name), 60*time.Second)
	if err != nil {
		spinner.Fail()
		panic(err)
	}
	spinner.Success()
	pterm.Success.Printfln("Succesfully connected with %d players", len(names)-1)
	for i, name := range names {
		msg := fmt.Sprintf(" %s: %s", mapAddresses[i], string(name))
		logger.Info(msg)
	}
	players := make([]poker.Player, len(names))
	for i := range names {
		players[i] = poker.Player{
			Name: string(names[i]),
			Id:   i,
			Hand: [2]poker.Card{},
			Pot:  1000,
		}
	}
	deck := poker.NewPokerDeck(p2p)
	deck.Shuffle()
	session := poker.Session{
		Board:   [5]poker.Card{},
		Players: players,
		RoundID: poker.MakeRoundID(poker.PreFlop),
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
	consensusNode := consensus.NewConsensusNode(
		pub, priv,
		map[int]ed25519.PublicKey{myRank: pub},
		&pokerManager,
		blockchain,
		p2p,
	)
	actions := []string{"Fold", "Check", "Call", "Raise", "AllIn"}

	raiseAmount := "0"
	selectedAction := ""
	var action consensus.Action
	area, _ := pterm.DefaultArea.Start()
	for {

		selectedAction, _ = pterm.DefaultInteractiveSelect.WithDefaultText("Select your next action").WithOptions(actions).Show()
		if selectedAction == "Raise" {

			defVal := strconv.Itoa(int(pokerManager.Session.HighestBet))
			raiseAmount, _ = pterm.DefaultInteractiveTextInput.WithDefaultText("Enter the amount to raise").WithDefaultValue(defVal).Show()
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
		if err :=  pokerManager.Validate(action.Payload); err != nil {
			area.Update()
			pterm.Error.Printfln("Invalid action: %s", err.Error())
			continue
		}


		if confirm, _ := pterm.DefaultInteractiveConfirm.WithDefaultText(fmt.Sprintf("Confirm to %s?", selectedAction)).WithDefaultValue(true).Show(); confirm {
			break
		}
		area.Update()
		pterm.Info.Println("Action cancelled.")
	}
	area.Stop()

	

	if err != nil {
		panic(err)
	}

	if err := consensusNode.ProposeAction(&action); err != nil {
		panic(err)
	}
	consensusNode.WaitForProposal()
}
