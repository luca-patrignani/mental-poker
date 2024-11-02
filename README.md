# Mental poker
This library aims to make it possible to play a fair game of Texas hold'em without needing a trusted third-party.

## Functional requirements
- Choose an initial dealer and the playing order
- Deal two cards face down to each player
- Record players actions for each round
- Deal community cards face up
- Come up with one or more winners at the end of each hand 

## Non-functional requirements
- Guarantee a fair card dealing such that a coalition, even if it is of the maximum size, shall not gain advantage over honest players by dealing
  - good cards to themselves
  - or bad cards to others.
- Guarantee that one player's cards are kept secret to the other players
- Provides non-repudiability over players choises.



## Optional functionalities
- Giving some amount of chips to each player
- Recording players bettings for each round
- Recording chip transactions between players.

