package main

import (
	"crypto/ed25519"
	"fmt"
	"net"
	"os"
	"sort"
	"time"

	"github.com/luca-patrignani/mental-poker/consensus"
	"github.com/luca-patrignani/mental-poker/domain/deck"
	"github.com/luca-patrignani/mental-poker/domain/poker"
	"github.com/luca-patrignani/mental-poker/ledger"
	"github.com/luca-patrignani/mental-poker/network"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <ip>\n", os.Args[0])
		os.Exit(1)
	}
	ip := os.Args[1]
	l, err := net.Listen("tcp", ip+":0")
	if err != nil {
		panic(err)
	}
	if _, err := fmt.Printf("Listening on %s\n", l.Addr().String()); err != nil {
		panic(err)
	}
	fmt.Println("Name:")
	var name string
	fmt.Scanln(&name)
	addresses := []string{l.Addr().String()}
	names := []string{name}
	for {
		fmt.Println("Give me a player's name. If done, type 'done'")
		var playerName string
		fmt.Scanln(&playerName)
		if playerName == "done" {
			break
		}
		names = append(names, playerName)
		fmt.Println("Give me its address and port in ipaddr:port format")
		var addr string
		fmt.Scanln(&addr)
		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid address %q: %v\n", addr, err)
			continue
		}
		addresses = append(addresses, tcpAddr.String())
	}
	sort.Slice(names, func(i, j int) bool {
		return addresses[i] < addresses[j]
	})
	sort.Slice(addresses, func(i, j int) bool {
		return addresses[i] < addresses[j]
	})
	players := make([]poker.Player, len(names))
	for i := range names {
		players[i] = poker.Player{
			Name: names[i],
			Id:   i,
			Hand: [2]poker.Card{},
			Pot: 1000,
		}
	}
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
	messages, err := p2p.AllToAllwithTimeout([]byte("Hello from "+name), 60*time.Second)
	if err != nil {
		panic(err)
	}
	fmt.Println("Received messages:")
	for i, msg := range messages {
		fmt.Printf("From %s: %s\n", mapAddresses[i], string(msg))
	}
	deck := deck.Deck{
		DeckSize: 52,
		Peer:     p2p,
	}
	session := poker.Session{
		Board: [5]poker.Card{},
		Players: players,
		Deck: deck,
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
	pokerManager := poker.NewPokerManager(&session)
	consensusNode := consensus.NewConsensusNode(
		pub, priv,
		map[int]ed25519.PublicKey{myRank: pub},
		pokerManager,
		blockchain,
		p2p,
	)
	consensusNode.UpdatePeers()
}
