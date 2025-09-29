package poker

import "fmt"

func CheckPokerLogic(a ActionType, amount uint, session *Session, idx int) error {
	switch a {
	case ActionFold:
		return nil
	case ActionBet:
		if session.Players[idx].Pot < amount {
			return fmt.Errorf("insufficient funds")
		}
	case ActionRaise:
		if session.Players[idx].Bet < session.HighestBet {
			return fmt.Errorf("raise must at least match highest bet")
		}
	case ActionCall:
		diff := session.HighestBet - session.Players[idx].Bet
		if diff > session.Players[idx].Pot {
			return fmt.Errorf("insufficient funds to call")
		}
	case ActionAllIn:
		remaining := session.Players[idx].Pot + session.Players[idx].Bet
		if remaining != amount {
			return fmt.Errorf("allin amount must match player's remaining pot")
		}
	case ActionCheck:
		if session.Players[idx].Bet != session.HighestBet {
			return fmt.Errorf("cannot check, must call, raise or fold")
		}
	default:
		return fmt.Errorf("unknown action")
	}
	return nil
}
