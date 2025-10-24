package main

import (
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sort"
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
	pterm.DefaultHeader.WithFullWidth().Printfln("Your username: %s", name)
	
	ip := os.Args[1]
	l, err := net.Listen("tcp", ip+":0")
	if err != nil {
		logger.Error("failed to listen on address", "address:"+ ip, err.Error())
		panic(err)
	}
	info := "Listening on "+ l.Addr().String()
	
	pterm.DefaultHeader.WithFullWidth().Println(info)

	// Print two new lines as spacer.
	pterm.Print("\n")

	addresses := []string{l.Addr().String()}
	for {
		addr, _ := pterm.DefaultInteractiveTextInput.WithDefaultText("Enter his address and port in ipaddr:port format. If done, type done").WithDefaultValue("").Show()
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
			fmt.Printf("Your rank is %d\n", i)
			myRank = i
		}
	}
	peer := network.NewPeer(
		myRank,
		mapAddresses,
		l,
		30*time.Second,
	)
	p2p := network.NewP2P(&peer)
	names, err := p2p.AllToAllwithTimeout([]byte(name), 60*time.Second)
	if err != nil {
		panic(err)
	}
	fmt.Println("Received messages:")
	for i, name := range names {
		fmt.Printf("From %s: %s\n", mapAddresses[i], string(name))
	}
	players := make([]poker.Player, len(names))
	for i := range names {
		players[i] = poker.Player{
			Name: string(names[i]),
			Id:   i,
			Hand: [2]poker.Card{},
			Pot: 1000,
		}
	}
	deck := poker.NewPokerDeck(p2p)
	deck.Shuffle()
	session := poker.Session{
		Board: [5]poker.Card{},
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
		Player: myRank,
	}
	consensusNode := consensus.NewConsensusNode(
		pub, priv,
		map[int]ed25519.PublicKey{myRank: pub},
		&pokerManager,
		blockchain,
		p2p,
	)
	action, err := consensus.MakeAction(myRank, pokerManager.ActionAllIn())
	if err != nil {
		panic(err)
	}
	if err := consensusNode.ProposeAction(&action); err != nil {
		panic(err)
	}
	consensusNode.WaitForProposal()
}
