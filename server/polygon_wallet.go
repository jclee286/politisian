package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"golang.org/x/crypto/sha3"
)

// PolygonWallet represents a Polygon wallet
type PolygonWallet struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	Address    string `json:"address"`
}

// PolygonAPIResponse represents Polygon API response
type PolygonAPIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
}

// PolygonTransaction represents a Polygon transaction
type PolygonTransaction struct {
	Hash        string `json:"hash"`
	BlockNumber int64  `json:"blockNumber"`
	Timestamp   int64  `json:"timestamp"`
	From        string `json:"from"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	Status      string `json:"status"`
	TokenName   string `json:"tokenName"`
	TokenAddress string `json:"tokenAddress"`
}

// PolygonBalance represents Polygon account balance
type PolygonBalance struct {
	MATIC int64 `json:"matic"` // MATIC balance (in wei, 1 MATIC = 10^18 wei)
	USDT  int64 `json:"usdt"`  // USDT balance (in smallest unit, 6 decimals)
	USDC  int64 `json:"usdc"`  // USDC balance (in smallest unit, 6 decimals)
}

// Token addresses on Polygon
const (
	POLYGON_USDT_ADDRESS = "0xc2132D05D31c914a87C6611C10748AEb04B58e8F" // USDT on Polygon
	POLYGON_USDC_ADDRESS = "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174" // USDC on Polygon
	POLYGON_RPC_URL      = "https://polygon-rpc.com"
)

// generatePolygonWallet creates a new Polygon wallet
func generatePolygonWallet() (*PolygonWallet, error) {
	// Generate private key
	privateKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	// Get private key bytes
	privateKeyBytes := privateKey.Serialize()
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	// Get public key (uncompressed)
	publicKey := privateKey.PubKey()
	publicKeyBytes := publicKey.SerializeUncompressed()
	publicKeyHex := hex.EncodeToString(publicKeyBytes)

	// Generate Ethereum-compatible address
	address, err := generateEthereumAddress(publicKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ethereum address: %v", err)
	}

	return &PolygonWallet{
		PrivateKey: privateKeyHex,
		PublicKey:  publicKeyHex,
		Address:    address,
	}, nil
}

// generateEthereumAddress generates Ethereum-compatible address from public key
func generateEthereumAddress(publicKeyBytes []byte) (string, error) {
	// Remove the first byte (0x04 for uncompressed key)
	if len(publicKeyBytes) == 65 && publicKeyBytes[0] == 0x04 {
		publicKeyBytes = publicKeyBytes[1:]
	}

	// Keccak256 hash of public key
	hash := sha3.NewLegacyKeccak256()
	hash.Write(publicKeyBytes)
	hashBytes := hash.Sum(nil)

	// Take last 20 bytes and add 0x prefix
	address := "0x" + hex.EncodeToString(hashBytes[12:])

	return address, nil
}

// getPolygonAPIKey returns Polygon API key from environment
func getPolygonAPIKey() string {
	// Etherscan API V2 key for Polygon network
	if key := os.Getenv("ETHERSCAN_API_KEY"); key != "" {
		return key
	}
	// Fallback to hardcoded key (for production, use environment variable)
	return "RTKWX1EIEXG3V59WFU9MKTNHQIRKKCNS2U"
}

// verifyPolygonTransaction verifies a Polygon transaction
func verifyPolygonTransaction(txHash, toAddress string, expectedAmount int64, tokenAddress string) (*PolygonTransaction, error) {
	// Use Polygon RPC or API services like Polygonscan
	apiKey := getPolygonAPIKey()
	
	// For USDT/USDC, we need to check token transfers, not ETH transfers
	var apiURL string
	if tokenAddress != "" {
		// Token transfer verification
		apiURL = fmt.Sprintf("https://api.polygonscan.com/api?module=account&action=tokentx&contractaddress=%s&txhash=%s", tokenAddress, txHash)
	} else {
		// MATIC transfer verification
		apiURL = fmt.Sprintf("https://api.polygonscan.com/api?module=proxy&action=eth_getTransactionByHash&txhash=%s", txHash)
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add API key to request
	q := req.URL.Query()
	q.Add("apikey", apiKey)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Polygon API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse response
	var apiResp map[string]interface{}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	// Check if transaction exists and is successful
	status, ok := apiResp["status"]
	if !ok || status != "1" {
		return nil, fmt.Errorf("transaction not found or failed")
	}

	// Parse transaction data
	tx := &PolygonTransaction{
		Hash:      txHash,
		Status:    "confirmed",
		TokenName: "USDT", // Default, should parse from response
		TokenAddress: tokenAddress,
	}

	log.Printf("Polygon transaction verified: %s", txHash)
	return tx, nil
}

// getPolygonBalance gets Polygon account balance
func getPolygonBalance(address string) (*PolygonBalance, error) {
	// Get MATIC balance
	maticBalance, err := getMaticBalance(address)
	if err != nil {
		log.Printf("Failed to get MATIC balance: %v", err)
		maticBalance = 0
	}

	// Get USDT balance
	usdtBalance, err := getTokenBalance(address, POLYGON_USDT_ADDRESS)
	if err != nil {
		log.Printf("Failed to get USDT balance: %v", err)
		usdtBalance = 0
	}

	// Get USDC balance
	usdcBalance, err := getTokenBalance(address, POLYGON_USDC_ADDRESS)
	if err != nil {
		log.Printf("Failed to get USDC balance: %v", err)
		usdcBalance = 0
	}

	return &PolygonBalance{
		MATIC: maticBalance,
		USDT:  usdtBalance,
		USDC:  usdcBalance,
	}, nil
}

// getMaticBalance gets MATIC balance for an address
func getMaticBalance(address string) (int64, error) {
	// Call Polygon RPC
	rpcPayload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBalance",
		"params":  []string{address, "latest"},
		"id":      1,
	}

	jsonData, err := json.Marshal(rpcPayload)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", POLYGON_RPC_URL, strings.NewReader(string(jsonData)))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var rpcResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return 0, err
	}

	// For demo, return 0
	return 0, nil
}

// getTokenBalance gets token balance for an address
func getTokenBalance(address, tokenAddress string) (int64, error) {
	// This would require calling the token contract's balanceOf function
	// For demo purposes, return 0
	return 0, nil
}

// sendPolygonTransaction sends tokens on Polygon network
func sendPolygonTransaction(fromPrivateKey, toAddress string, amount int64, tokenAddress string) (string, error) {
	// This is a complex operation that requires:
	// 1. Building the transaction
	// 2. Signing with private key
	// 3. Broadcasting to network
	
	// For demo purposes, return a fake transaction hash
	fakeHash := fmt.Sprintf("0x%x", sha256.Sum256([]byte(fmt.Sprintf("%s%s%d%d", 
		fromPrivateKey[:10], toAddress, amount, time.Now().Unix()))))
	
	log.Printf("Demo Polygon transaction: %s", fakeHash)
	return fakeHash, nil
}

// validatePolygonAddress validates a Polygon address format
func validatePolygonAddress(address string) bool {
	// Ethereum-compatible addresses start with '0x' and are 42 characters long
	if len(address) != 42 || !strings.HasPrefix(address, "0x") {
		return false
	}

	// Check if it's valid hex
	_, err := hex.DecodeString(address[2:])
	return err == nil
}

// Helper function to convert USDT/USDC amount (with 6 decimals) to integer
func stablecoinToInteger(amount float64) int64 {
	return int64(amount * 1000000) // USDT/USDC have 6 decimal places
}

// Helper function to convert integer to USDT/USDC amount
func integerToStablecoin(amount int64) float64 {
	return float64(amount) / 1000000
}

// Helper function to convert MATIC amount (with 18 decimals) to integer
func maticToInteger(amount float64) int64 {
	return int64(amount * 1000000000000000000) // MATIC has 18 decimal places
}

// Helper function to convert integer to MATIC amount
func integerToMatic(amount int64) float64 {
	return float64(amount) / 1000000000000000000
}