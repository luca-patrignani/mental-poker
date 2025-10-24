package poker

import "fmt"

// CheckPokerLogic validates that a poker action complies with game rules for the given session
// and player. It checks constraints like sufficient funds, bet consistency, and turn validity
// specific to each action type. Returns an error if the action violates any rule.
func checkPokerLogic(a ActionType, amount uint, session *Session, idx int) error {
	switch a {
	case ActionFold:
		return nil
	case ActionBet:
		if session.Players[idx].Pot < amount {
			return fmt.Errorf("insufficient funds")
		}
	case ActionRaise:
		if amount <= session.HighestBet {
			return fmt.Errorf("raise must be higher than current highest bet")
		}
		if session.Players[idx].Pot < amount {
			return fmt.Errorf("insufficient funds to raise %d", amount)
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
	case ActionShowdown:
		if extractRoundName(session.RoundID) != Showdown {
			return fmt.Errorf("cannot showdown before river")
		}
	default:
		return fmt.Errorf("unknown action")
	}
	return nil
}
