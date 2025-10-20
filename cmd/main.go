package main

/*
import (

	"crypto/ed25519"
	"fmt"
	"log"
	"time"

	"github.com/luca-patrignani/mental-poker/application"

)
*/
func main() {
}

// Setup network
//TODO
/*

func printGameState(session *poker.Session) {
	fmt.Println("--- Current Game State ---")
	fmt.Printf("Round: %s\n", session.RoundID)
	fmt.Printf("Highest Bet: %d\n", session.HighestBet)
	fmt.Printf("Current Turn: Player %d\n", session.CurrentTurn)
	fmt.Println("Players:")
	for _, p := range session.Players {
		status := "Active"
		if p.HasFolded {
			status = "Folded"
		}
		fmt.Printf("  Player %d: Pot=%d, Bet=%d, Status=%s\n",
			p.Rank, p.Pot, p.Bet, status)
	}
	for _, c := range session.Board {
		fmt.Printf("%s\t", c)
	}
	fmt.Println("--------------------------")
}

func onActionCommit(action *poker.PokerAction) {
	fmt.Printf("✓ Action committed: Player %s -> %s (amount: %d)\n",
		action.PlayerID, action.Type, action.Amount)
}

func onPlayerBanned(playerID string, reason string) {
	fmt.Printf("⚠ Player %s banned: %s\n", playerID, reason)
}

func onRoundEnd(winners map[int]uint) {
	fmt.Println("🏆 Round ended! Winners:")
	for rank, amount := range winners {
		fmt.Printf("  Player %d won %d chips\n", rank, amount)
	}
}
*/
