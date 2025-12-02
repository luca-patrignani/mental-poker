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


# Mental Poker 🃏

**Play Texas Hold'em poker without a dealer - peer-to-peer, decentralized, and cryptographically secure**

[![Latest Release](https://img.shields.io/github/v/release/lhttps://github.com/luca-patrignani/mental-poker.23.5-blue.n

Mental Poker allows you to play fair Texas Hold'em poker games over a network without needing a trusted dealer or server. All players participate equally in shuffling and dealing cards using cryptographic protocols.

## What is Mental Poker?

Mental Poker is a way to play card games remotely without physical cards or a trusted dealer. Using cryptography, players can shuffle and deal cards in a way that guarantees:

- **Fairness** - No one can cheat or manipulate the deck
- **Privacy** - You only see your own cards until the showdown
- **No Central Authority** - No server or dealer controls the game

## Features

- 🎮 **Full Texas Hold'em** - Complete poker rules with all betting rounds
- 🔐 **Cryptographically Secure** - Provably fair card dealing
- 🌐 **Peer-to-Peer** - No central server required
- 💰 **Chip Management** - Track player balances and bets
- 📋 **Game History** - All actions are logged for transparency
- 🖥️ **Beautiful CLI** - Styled terminal interface with interactive menus
- 🎯 **Auto-discovery** - Automatically finds other players on your network

## Installation

### Option 1: Download Pre-built Binary (Recommended)

Download the latest release for your operating system:

**[→ Go to Releases Page](https://github.com/luca-patrignani/mental-poker/releases)**

Available for:
- **Linux** (amd64): `MentalPoker-linux-amd64`
- **macOS** (Apple Silicon): `MentalPoker-darwin-arm64`
- **Windows** (amd64): `MentalPoker-windows-amd64.exe`

#### Linux/macOS:
```bash
# Download the binary for your platform
chmod +x MentalPoker-*
./MentalPoker-*
```

#### Windows:
Simply double-click `MentalPoker-windows-amd64.exe` or run it from the command prompt.

### Option 2: Build from Source

**Prerequisites:**
- Go 1.23.5 or higher
- Git

```bash
# Clone the repository
git clone https://github.com/luca-patrignani/mental-poker.git

# Navigate to the project directory
cd mental-poker/v3

# Build the application
go build -o MentalPoker ./cmd

# Run it
./MentalPoker
```

## How to Play

### Setting Up a Game

Mental Poker uses **automatic peer discovery** on your local network. All players should be on the same network (LAN or VPN).

**Step 1: Each player starts the application**

```bash
./MentalPoker <your-ip-address>
```

Replace `<your-ip-address>` with your local IP address (e.g., `192.168.1.100`).

To find your IP address:
- **Linux/macOS**: `ifconfig` or `ip addr`
- **Windows**: `ipconfig`

**Step 2: Enter your username**

When prompted, type your name and press Enter.

**Step 3: Wait for all players**

The application will automatically discover other players on the network. You'll see messages like:
```
ℹ Discovered player Alice at address 192.168.1.101:53550
ℹ Discovered player Bob at address 192.168.1.102:53550
```

**Step 4: Confirm when ready**

Once all players are connected, confirm with `y` when asked:
```
? Are all players connected? (y/N)
```

**Step 5: Play!**

The game will start automatically. Players are assigned initial bankrolls of **1000 chips** each.

### Playing the Game

The game follows standard Texas Hold'em rules:

1. **Blinds are posted** - Small blind and big blind are automatically posted
2. **Cards are dealt** - Each player receives 2 hole cards (only you can see yours)
3. **Betting rounds**:
   - **Pre-flop** - After receiving hole cards
   - **Flop** - After 3 community cards are revealed
   - **Turn** - After the 4th community card
   - **River** - After the 5th community card
4. **Showdown** - Remaining players reveal cards, best hand wins

### Available Actions

When it's your turn, you can:

- **Fold** - Give up your hand
- **Check** - Pass without betting (only if no one has bet)
- **Call** - Match the current bet
- **Raise** - Increase the bet (you'll be asked how much)
- **All-In** - Bet all your remaining chips

### Action Timeout

You have **30 seconds** (configurable) to make your decision. If time runs out, the game will automatically **check** or **fold** for you.

### End of Hand

After the showdown, you'll be asked if you want to continue:
```
? Ready for the next Round? (Y/n)
```

- **Yes** - Start a new hand
- **No** - Leave the game (your seat will be removed)

The game continues until only one player remains.

## Command Line Options

```bash
./MentalPoker <ip-address> [OPTIONS]

Required:
  <ip-address>        Your local IP address

Options:
  --port <port>       Port to listen on (default: 53550)
  --timeout <sec>     Action timeout in seconds (default: 30)
```

### Examples

**Basic usage:**
```bash
./MentalPoker 192.168.1.100
```

**Custom port:**
```bash
./MentalPoker 192.168.1.100 --port 12345
```

**Longer timeout (60 seconds):**
```bash
./MentalPoker 192.168.1.100 --timeout 60
```

## Game Rules

### Texas Hold'em Basics

- **2-10 players** supported
- Each player starts with **1000 chips**
- **Small blind**: 5 chips
- **Big blind**: 10 chips
- Standard hand rankings apply (Royal Flush > Straight Flush > Four of a Kind > ... > High Card)

### Hand Progression

1. Dealer button rotates clockwise after each hand
2. Small blind is posted by the player after the dealer
3. Big blind is posted by the player after the small blind
4. Action starts with the player after the big blind
5. Community cards are revealed progressively (flop, turn, river)

## Troubleshooting

### "Port already in use" error

Try a different port:
```bash
./MentalPoker 192.168.1.100 --port 12345
```

### Players can't discover each other

Make sure:
- All players are on the **same network**
- **Firewall isn't blocking** ports 53550-53551
- You're using the correct **local IP address** (not 127.0.0.1)

### Game freezes or times out

- Check your **network connection**
- Ensure all players have **stable connectivity**
- Try increasing the timeout: `--timeout 60`

### Connection refused

- Verify all players are using the **correct IP addresses**
- Check if any **antivirus or firewall** is blocking the connection

## Requirements

- **Network**: Local area network (LAN) or VPN
- **Ports**: 53550 (main), 53551 (discovery) - must be available
- **Players**: 2-10 players minimum/maximum

## License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

## Authors

- **Luca Patrignani** ([@luca-patrignani](https://github.com/luca-patrignani))
- **Marco Galeri** ([@Fre0Grella](https://github.com/Fre0Grella))

## Acknowledgments

Built using cryptographic protocols from academic research on mental poker, making it possible to play card games without a trusted dealer.

***

**🎮 Enjoy the game!**