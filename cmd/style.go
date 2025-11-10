package main

import (
	"strconv"

	"github.com/luca-patrignani/mental-poker/domain/poker"
	"github.com/pterm/pterm"
)

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

func printPlayerInfo(p poker.Player, main bool) string {
	hpadding := 4
	if main {
		hpadding = 10
	}
	pbox := pterm.DefaultBox.WithHorizontalPadding(hpadding).WithTopPadding(1).WithBottomPadding(1)
	var active string
	if p.HasFolded {
		active = pterm.LightRed("Folded")
	} else {
		active = pterm.LightGreen("Active")
	}
	hand := pterm.BgGreen.Sprintf("%s - %s", p.Hand[0].String(), p.Hand[1].String())
	return pbox.WithTitle(p.Name).WithTitleTopLeft().Sprintf("%s\nCurrent Bet: %d\nBankroll: %d\n%s\n", active, p.Bet, p.Pot, hand)
}

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
