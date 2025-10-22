package main

import (
	"fmt"
	"net"
	"os"
	"sort"
	"time"

	"github.com/luca-patrignani/mental-poker/network"
)


func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <ip>\n", os.Args[0])
		os.Exit(1)
	}
	ip := os.Args[1]
	l, err := net.Listen("tcp", ip + ":0")
	if err != nil {
		panic(err)
	}
	if _, err := fmt.Printf("Listening on %s\n", l.Addr().String()); err != nil {
		panic(err)
	}
	fmt.Println("Username:")
	var username string
	fmt.Scanln(&username)
	addresses := []string{l.Addr().String()}
	for {
		fmt.Println("Give me the other's players addresses and port in ipaddr:port format. If done, type 'done'")
		var addr string
		fmt.Scanln(&addr)
		if addr == "done" {
			break
		}
		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid address %q: %v\n", addr, err)
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
	messages, err := p2p.AllToAllwithTimeout([]byte("Hello from " + username), 60*time.Second)
	if err != nil {
		panic(err)
	}
	fmt.Println("Received messages:")
	for i, msg := range messages {
		fmt.Printf("From %s: %s\n", mapAddresses[i], string(msg))
	}
}
