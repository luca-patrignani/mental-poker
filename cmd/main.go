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
	// Setup crypto keys per tutti i player
	playerKeys := make(map[string]ed25519.PublicKey)
	pub, priv, _ := ed25519.GenerateKey(nil)

	// Crea peer per comunicazione
	peer := network.NewPeer(myRank, addresses, listeners[myRank], 30*time.Second)
	defer peer.Close()

	// Crea network adapter
	netAdapter := network.NewP2PAdapter(&peer)

	// Setup storage
	store := storage.NewMemoryStore()
	// O per persistenza su file:
	// store, err := storage.NewFileStore(fmt.Sprintf("blockchain_player%d.json", myRank))
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Crea sessione iniziale
	initialSession := createInitialSession(numPlayers)

	// Crea GameOrchestrator
	orchestrator, err := application.NewGameOrchestrator(
		fmt.Sprintf("%d", myRank),
		pub,
		priv,
		playerKeys,
		initialSession,
		netAdapter,
		store,
	)
	if err != nil {
		log.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orchestrator.Shutdown()

	// Setup callbacks
	orchestrator.SetCallbacks(
		onActionCommit,
		onPlayerBanned,
		onRoundEnd,
	)

	// Game loop
	fmt.Printf("Player %d starting game...\n", myRank)
	runGameLoop(orchestrator)
}

func runGameLoop(orc *application.GameOrchestrator) {
	for {
		session := orc.GetCurrentGameState()

		// Verifica se il gioco è finito
		activePlayers := 0
		for _, p := range session.Players {
			if !p.HasFolded && p.Pot > 0 {
				activePlayers++
			}
		}

		if activePlayers <= 1 {
			fmt.Println("Game over!")
			break
		}

		// Se è il mio turno, proponi un'azione
		if orc.IsMyTurn() {
			fmt.Printf("\n=== My turn (Player %s) ===\n", orc.GetPlayerID())
			printGameState(session)

			// Logica semplice: check se possibile, altrimenti call
			action, amount := decideAction(session, orc.GetPlayerID())

			fmt.Printf("Proposing action: %s (amount: %d)\n", action, amount)
			if err := orc.ProposeAction(action, amount); err != nil {
				log.Printf("Failed to propose action: %v", err)
				// Prova a fare fold in caso di errore
				orc.ProposeAction(poker.ActionFold, 0)
			}
		} else {
			// Attendi il turno degli altri
			currentActor := session.Players[session.CurrentTurn].Rank
			fmt.Printf("Waiting for player %d to act...\n", currentActor)

			if err := orc.WaitForTurn(); err != nil {
				log.Printf("Error waiting for turn: %v", err)
				break
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Verifica blockchain alla fine
	fmt.Println("\n=== Verifying blockchain integrity ===")
	if err := orc.VerifyBlockchain(); err != nil {
		log.Printf("Blockchain verification failed: %v", err)
	} else {
		fmt.Println("Blockchain verified successfully!")
	}

	// Mostra storico
	fmt.Println("\n=== Game History ===")
	history := orc.GetBlockchainHistory()
	for i, action := range history {
		fmt.Printf("Block %d: %+v\n", i+1, action)
	}
}

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
