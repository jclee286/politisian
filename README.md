# Project Development Purpose

This project is planned for people around the world who dream of innovation in politics and distrust of political systems.
There are many shortcomings to creating it alone, and we hope that all countries will participate together,
with developers from each country collaborating to complete this project.
We strongly request that you participate as a Collaborator to change the world together.


# Politician Republic

> **Anonymous Politician Coin Exchange**  
> Traditional email/password login + Hybrid blockchain wallet system

## 🏛️ Project Overview

Politician Republic is a decentralized exchange that perfectly protects developer anonymity while allowing users to trade politician coins for real money (USDT/USDC).

### Core Features
- **🔒 Fully Anonymous Operation**: Complete developer information blocking by removing OAuth
- **💰 Real Asset Trading**: Trade politician coins with Polygon-based USDT/USDC
- **⚡ Hybrid Structure**: Politician coins (self-hosted/free) + Stablecoins (Polygon)
- **🏪 Order Book Trading**: Real-time buy/sell order book-based trading
- **💎 Escrow System**: Guaranteed safe trading

## 🏗️ System Architecture

```
┌─────────────────────────────────────────┐
│           Integrated Wallet (UI)        │
├─────────────────────────────────────────┤
│ Politician Coins (Self-blockchain - Free, Fast) │
│ - Moon Jae-in Coin: 1,000 tokens       │  
│ - Yoon Suk-yeol Coin: 500 tokens       │
├─────────────────────────────────────────┤
│ Stable Coins (Polygon - Real Assets)    │
│ - USDT: $1,000                         │
│ - USDC: $500                           │
│ - MATIC: 10 tokens (for fees)          │
└─────────────────────────────────────────┘
```

### Technology Stack
- **Backend**: Go + CometBFT (self-hosted blockchain)
- **Frontend**: Vanilla HTML/CSS/JavaScript
- **External**: Polygon Network (USDT/USDC)
- **API**: Etherscan API V2 (transaction verification)
- **Deployment**: Docker + GitHub Actions CI/CD

## 📊 Trading System

### Order Book Exchange
```
Buy Orders                     Sell Orders
Price   Quantity               Price   Quantity
1,100   500 tokens ←─ Best Bid 1,150   300 tokens ← Best Ask
1,090   800 tokens             1,160   700 tokens
1,080   1,200 tokens           1,170   500 tokens
```

### Escrow Safety Mechanism
- **Buy Orders**: USDT/USDC frozen → Receive politician coins when filled
- **Sell Orders**: Politician coins frozen → Receive USDT/USDC when filled
- **On Failure**: Automatic unfreezing

## 💳 Wallet System

### Polygon Stablecoin Wallet
- **Address Format**: `0x1234...abcd` (Ethereum compatible)
- **Supported Tokens**: USDT, USDC, MATIC
- **Deposit Method**: Binance → Polygon Network withdrawal
- **Contract Addresses**:
  - USDT: `0xc2132D05D31c914a87C6611C10748AEb04B58e8F`
  - USDC: `0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174`

### Self-Hosted Politician Coin Wallet
- **Free Trading**: Fee-free instant transfers
- **Fast Processing**: 5-second block generation
- **Secure Storage**: CometBFT consensus algorithm

## 🚀 Deployment and Execution

### Local Development
```bash
# Install dependencies
go mod tidy

# Run application
go run main.go

# Access in web browser
http://localhost:8080
```

### Docker Execution
```bash
# Build and run container
docker-compose up --build

# Run in background
docker-compose up -d
```

### Production Deployment
```bash
# Auto-deploy when pushing to GitHub
git push origin main
# → GitHub Actions automatically deploys to server
```

## 📁 Project Structure

```
politisian/
├── 🚀 main.go                    # Application entry point
├── 📁 app/                       # Blockchain application
│   ├── abci.go                   # ABCI transaction processing
│   ├── app.go                    # Application initialization
│   └── state.go                  # State management
├── 📁 server/                    # HTTP API server
│   ├── auth.go                   # Traditional authentication (email/password)
│   ├── handlers.go               # Basic API handlers
│   ├── polygon_handlers.go       # USDT/USDC deposit/withdrawal API
│   ├── polygon_wallet.go         # Polygon wallet generation/verification
│   ├── trade_handlers.go         # Exchange API
│   └── server.go                 # Server routing
├── 📁 frontend/                  # Web interface
│   ├── index.html               # Dashboard/Exchange
│   ├── login.html               # Login
│   └── signup.html              # Sign up
├── 📁 pkg/types/                 # Data structures
│   └── types.go                 # Common type definitions
├── 🐳 Dockerfile                 # Container image
├── 🔧 docker-compose.yml         # Development environment setup
└── 📚 README.md                  # This document
```

## 🔐 Security and Anonymity

### Developer Anonymity Guarantee
- **Complete OAuth Removal**: Block third-party authentication like Google login
- **Traditional Authentication**: Use only email/password
- **Minimal Personal Information**: Skip email verification

### User Security
- **PIN-Based Wallet**: Dual security (login + PIN)
- **bcrypt Hashing**: Encrypted password storage
- **Escrow Protection**: Automatic refund on failed transactions
- **Session Management**: Memory-based session store

## 💰 Economic Model

### Politician Coins
- **Supply**: Fixed 10,000 tokens per politician
- **Initial Distribution**: 100 tokens each for 3 selected politicians upon signup
- **Trading**: Tradeable with USDT/USDC
- **Price**: Market-determined based on order book

### Fee Structure
- **Politician Coin Trading**: Free (self-hosted blockchain)
- **USDT/USDC Deposits**: Only Binance withdrawal fees (~1 USDT)
- **USDT/USDC Withdrawals**: Polygon network fees (~0.1 USDT)

## 🔗 External Integration

### Polygon Network
- **RPC**: `https://polygon-rpc.com`
- **Transaction Verification**: Etherscan API V2
- **Supported Wallets**: MetaMask, Trust Wallet, etc.

### API Key Configuration
```go
// Environment variable setup
export ETHERSCAN_API_KEY="your_api_key_here"

// Or direct configuration in code
func getPolygonAPIKey() string {
    return "RTKWX1EIEXG3V59WFU9MKTNHQIRKKCNS2U"
}
```

## 🎯 User Scenarios

### New User
1. **Sign Up**: Enter email/password + profile information
2. **Select Politicians**: Receive 300 tokens for selecting 3 politicians
3. **Wallet Creation**: Automatic Polygon address generation
4. **USDT Deposit**: Binance → Polygon → Our address

### Trading
1. **Buy Order**: Place buy order for politician coins with USDT
2. **Escrow**: USDT automatically frozen
3. **Matching**: Execute when matched with sell orders at same price
4. **Settlement**: Receive politician coins and deduct USDT

### Withdrawal
1. **Withdrawal Request**: Enter USDT/USDC withdrawal address
2. **PIN Authentication**: Enter wallet PIN
3. **Polygon Transfer**: Send tokens to actual blockchain
4. **Confirmation**: Receive in MetaMask etc. within 1-3 minutes

## 📈 Development Roadmap

### ✅ Completed Features
- Traditional email/password authentication system
- Polygon-based USDT/USDC wallet
- Order book exchange system
- Escrow safety mechanism
- Real API key integration
- Docker deployment automation

### 🚧 In Progress
- UI/UX improvements and integrated wallet display
- Real-time price charts
- Mobile optimization

### 📋 Future Plans
- Support for various stablecoins (BUSD, DAI, etc.)
- Leverage trading features
- Politician coin staking rewards
- Social features (follow, community)

## 🤝 Contribution Guide

### Development Environment Setup
```bash
# Clone repository
git clone https://github.com/jclee286/politisian.git
cd politisian

# Install Go modules
go mod tidy

# Run locally
go run main.go
```

### Commit Conventions
- `feat:` Add new features
- `fix:` Bug fixes
- `refactor:` Code refactoring
- `docs:` Documentation updates





**🛡️ Disclaimer**: Developer anonymity is guaranteed, and we are not responsible for user trading losses.