This project aims to make it possible to play a fair game of  poker Texas hold'em without needing a trusted third party.

# Roadmap
## Deck
Create a library for handling a cards' deck without a TTP. It must support the following basic operations:
- Create a new shuffled deck
- Deal a faced-down card to a player
- Reveal a card.

The implementation will be based on Wei & Wang paper [A Fast Mental Poker Protocol](https://www.researchgate.net/publication/220334557_A_Fast_Mental_Poker_Protocol).

## Game mechanics
Create a library which implements the actual game of poker Texas hold'em.

## User Interface
Create a simple CLI for playing a game of poker with our friends. 

## Game log
Integrate the game mechanics' library with some sort of local DLT in order to keep track of players' actions and provide non-repudiability over them.

## Fiches transactions
Create a library will record fiches transactions between players and players' balances during the game. 

## Technologies involved
The technologies which are going to be used are
- Go as the main programming language
- Some sort of DLT, probably a blockchain over LAN, for storing players' choices during the game
