package poker

import "fmt"

// GamePhase represents the current state of the poker game
type GamePhase string

const (
	StateWaitingForPlayers GamePhase = "waiting_for_players"
	StatePostBlinds        GamePhase = "post_blinds"
	StatePreFlop           GamePhase = "pre_flop"
	StateFlop              GamePhase = "flop"
	StateTurn              GamePhase = "turn"
	StateRiver             GamePhase = "river"
	StateShowdown          GamePhase = "showdown"
	StatePayout            GamePhase = "payout"
)

// BettingState tracks the betting within a round
type BettingState string

const (
	BettingNotStarted BettingState = "not_started"
	BettingInProgress BettingState = "in_progress"
	BettingComplete   BettingState = "complete"
)

// PokerFSM manages the state transitions of a poker game
type PokerFSM struct {
	//mu              sync.RWMutex
	currentPhase GamePhase
	bettingState BettingState
	session      *Session
	minPlayers   uint
	bigBlind     uint
	smallBlind   uint

	// Callbacks for state transitions
	onStateChange  func(old, new GamePhase)
	onBettingRound func(state GamePhase)
	onHandComplete func(winners []Player)
}

// NewPokerFSM creates a new poker state machine
func NewPokerFSM(minPlayers uint, smallBlind uint) *PokerFSM {
	return &PokerFSM{
		currentPhase: StateWaitingForPlayers,
		bettingState: BettingNotStarted,
		minPlayers:   minPlayers,
		bigBlind:     smallBlind * 2,
		smallBlind:   smallBlind,
	}
}

// SetSession attaches a game session to the FSM
func (fsm *PokerFSM) SetSession(session *Session) {
	//fsm.mu.Lock()
	//defer fsm.mu.Unlock()
	fsm.session = session
}

// GetCurrentState returns the current game state
func (fsm *PokerFSM) GetCurrentBettingState() BettingState {
	//fsm.mu.RLock()
	//defer fsm.mu.RUnlock()
	return fsm.bettingState
}

// GetCurrentPhase returns the complete phase information
func (fsm *PokerFSM) GetCurrentPhase() GamePhase {
	//fsm.mu.RLock()
	//defer fsm.mu.RUnlock()
	return fsm.currentPhase
}

func (fsm *PokerFSM) postBlinds() error {
	numPlayers := len(fsm.session.Players)
	if numPlayers < 2 {
		return fmt.Errorf("need at least 2 players")
	}

	// Small blind
	sbIdx := (int(fsm.session.Dealer) + 1) % numPlayers
	if fsm.session.Players[sbIdx].Pot < fsm.smallBlind {
		return fmt.Errorf("player %d cannot afford small blind", sbIdx)
	}
	//cacciare node.propose bet
	// Big blind
	bbIdx := (int(fsm.session.Dealer) + 2) % numPlayers
	if fsm.session.Players[bbIdx].Pot < fsm.bigBlind {
		return fmt.Errorf("player %d cannot afford big blind", bbIdx)
	}
	//node propose bet

	// Set turn to player after big blind
	fsm.session.CurrentTurn = uint((bbIdx + 1) % numPlayers)

	return nil
}
