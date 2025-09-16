package poker

// GamePhase represents the current state of the poker game
type GamePhase string

const (
	StateWaitingForPlayers GamePhase = "waiting_for_players"
	StatePreFlop           GamePhase = "pre_flop"
	StateFlop              GamePhase = "flop"
	StateTurn              GamePhase = "turn"
	StateRiver             GamePhase = "river"
	StateShowdown          GamePhase = "showdown"
	StateHandComplete      GamePhase = "hand_complete"
)

// BettingState tracks the betting within a round
type BettingState string

const (
	BettingNotStarted BettingState = "not_started"
	BettingInProgress BettingState = "in_progress"
	BettingComplete   BettingState = "complete"
)
