package main

import (
	"strconv"

	"github.com/luca-patrignani/mental-poker/domain/poker"
	"github.com/pterm/pterm"
)

// getActionPanel creates a panel displaying the last action performed in the game.
// The panel shows the player name and action details in a formatted box.
//
// Parameters:
//   - pa: The poker action to display
//   - psm: Poker manager containing session state
//
// Returns a pterm.Panel with formatted action information.
func getActionPanel(pa poker.PokerAction, psm poker.PokerManager) pterm.Panel {
	pbox := pterm.DefaultBox.WithHorizontalPadding(4).WithTopPadding(1).WithBottomPadding(1)
	actionString := ""
	p := psm.GetSession().Players
	switch pa.Type {
	case "raise":
		actionString = pterm.Sprintfln("%s raised by %d", p[psm.FindPlayerIndex(pa.PlayerID)].Name, pa.Amount)
	default:
		actionString = pterm.Sprintfln("%s performed action: %s", p[psm.FindPlayerIndex(pa.PlayerID)].Name, pa.Type)
	}
	return pterm.Panel{Data: pbox.WithTitle(pterm.LightYellow("|LAST ACTION|")).WithTitleTopCenter().Sprintf(actionString)}
}

// getWinnerPanel creates a panel showing the winner(s) of the game with their
// winnings and winning hand descriptions. Handles both single winners and split pots.
//
// Parameters:
//   - psm: Poker manager containing session state and winner information
//
// Returns a pterm.Panel with formatted winner information or an error.
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

// printSingleWinnerInfo formats information about a single winner for display.
// Shows the winner's name, amount won, and winning hand (if cards were shown).
//
// Parameters:
//   - psm: Poker manager containing session state
//   - id: Player ID of the winner
//   - amount: Amount won by the player
//
// Returns formatted string with winner information or an error.
func printSingleWinnerInfo(psm poker.PokerManager, id int, amount int) (string, error) {
	s := psm.GetSession()
	idx := psm.FindPlayerIndex(id)
	p := s.Players[idx]
	playerString := ""
	if p.Hand[0].Rank() == 0 || p.Hand[1].Rank() == 0 {
		playerString = pterm.Sprintfln("%s won %d Taking down the pot", pterm.LightCyan(p.Name), amount)
	} else {
		hand, err := s.DescribeHand(idx)
		if err != nil {
			return "", err
		}
		playerString = pterm.Sprintfln("%s won %d with %s", pterm.LightCyan(p.Name), amount, hand)
	}
	return playerString, nil
}

// printState prints the current game state including player information, board,
// and any additional panels. This is the main UI rendering function.
//
// Parameters:
//   - psm: Poker manager containing session state
//   - additionalPanel: Optional panels to display (e.g., action or winner panels)
func printState(psm poker.PokerManager, additionalPanel ...pterm.Panel) {
	s := psm.GetSession()
	var panels []pterm.Panel
	var mainPlayer pterm.Panel
	for _, p := range s.Players {
		if p.Id != psm.Player {
			pInfo := printPlayerInfo(p, false)
			panel := pterm.Panel{Data: pInfo}
			panels = append(panels, panel)
		} else {

			mainPlayer = pterm.Panel{Data: printPlayerInfo(p, true)}
		}
	}
	board := pterm.Panel{Data: printBoardInfo(s.Board[:], psm.Session.Round, s.Pots)}
	dashboard := []pterm.Panel{mainPlayer}
	dashboard = append(dashboard, additionalPanel...)

	pterm.DefaultPanel.WithPanels([][]pterm.Panel{
		panels,
		{board},
		dashboard,
	}).Render()
}

// printPlayerInfo formats a player's information for display in a box.
// Shows status (active/folded), current bet, bankroll, and hand cards.
//
// Parameters:
//   - p: Player to display
//   - main: If true, adds extra padding for the main player's box
//
// Returns formatted string with player information.
func printPlayerInfo(p poker.Player, main bool) string {
	hpadding := 4
	if main {
		hpadding = 10
	}
	pbox := pterm.DefaultBox.WithHorizontalPadding(hpadding).WithTopPadding(1).WithBottomPadding(1)
	var active string
	if p.HasFolded && p.BankRoll <= 0 {
		active = pterm.Cyan("Spectator Mode")
		return pbox.WithTitle(p.Name).WithTitleTopLeft().Sprintf("%s", active)
	}
	if p.HasFolded {
		active = pterm.LightRed("Folded")
	} else {
		active = pterm.LightGreen("Active")
	}
	hand := pterm.BgGreen.Sprintf("%s - %s", p.Hand[0].String(), p.Hand[1].String())
	return pbox.WithTitle(p.Name).WithTitleTopLeft().Sprintf("%s\nCurrent Bet: %d\nBankroll: %d\n%s\n", active, p.Bet, p.BankRoll, hand)
}

// printBoardInfo formats the community cards and pot information for display.
// Shows all board cards, pot amounts, and the current betting round.
//
// Parameters:
//   - b: Slice of board cards
//   - round: Current betting round
//   - pots: All active pots
//
// Returns formatted string with board and pot information.
func printBoardInfo(b []poker.Card, round poker.Round, pots []poker.Pot) string {
	board := ""
	for _, c := range b {
		board += c.String() + " - "
	}
	for i, p := range pots {
		board += " Pot" + strconv.Itoa(i) + ": " + strconv.Itoa(int(p.Amount)) + " | "
	}

	return pterm.BgGreen.Sprint("\n" + board + string(round) + "\n")
}
