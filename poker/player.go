package poker

type Player struct {
	Name string
	Rank int
	Hand [2]Card
	HasFolded bool
	Bet uint 	// The amount of money bet in the current betting round
	Pot uint
}
