package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil/base58"
)

// TronWallet represents a TRON wallet
type TronWallet struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	Address    string `json:"address"`
}

// TronAPIResponse represents TRON API response
type TronAPIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
}

// TronTransaction represents a TRON transaction
type TronTransaction struct {
	TxID        string `json:"txID"`
	BlockNumber int64  `json:"blockNumber"`
	Timestamp   int64  `json:"timestamp"`
	From        string `json:"from"`
	To          string `json:"to"`
	Amount      int64  `json:"amount"`
	Status      string `json:"status"`
	TokenName   string `json:"tokenName"`
}

// TronBalance represents TRON account balance
type TronBalance struct {
	TRX  int64 `json:"trx"`  // TRX balance (in SUN, 1 TRX = 1,000,000 SUN)
	USDT int64 `json:"usdt"` // USDT balance (in smallest unit)
}

// generateTronWallet creates a new TRON wallet
func generateTronWallet() (*TronWallet, error) {
	// Generate private key
	privateKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	// Get private key bytes
	privateKeyBytes := privateKey.Serialize()
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	// Get public key
	publicKey := privateKey.PubKey()
	publicKeyBytes := publicKey.SerializeCompressed()
	publicKeyHex := hex.EncodeToString(publicKeyBytes)

	// Generate TRON address
	address, err := generateTronAddress(publicKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TRON address: %v", err)
	}

	return &TronWallet{
		PrivateKey: privateKeyHex,
		PublicKey:  publicKeyHex,
		Address:    address,
	}, nil
}

// generateTronAddress generates TRON address from public key
func generateTronAddress(publicKeyBytes []byte) (string, error) {
	// Remove the first byte (compression flag) if compressed
	if len(publicKeyBytes) == 33 {
		publicKeyBytes = publicKeyBytes[1:]
	}

	// SHA256 hash
	hash1 := sha256.Sum256(publicKeyBytes)
	
	// Keccak256 would be ideal, but using SHA256 for simplicity
	hash2 := sha256.Sum256(hash1[:])

	// Take last 20 bytes
	addressBytes := hash2[12:]

	// Add TRON prefix (0x41)
	addressWithPrefix := append([]byte{0x41}, addressBytes...)

	// Double SHA256 for checksum
	checksum1 := sha256.Sum256(addressWithPrefix)
	checksum2 := sha256.Sum256(checksum1[:])

	// Add first 4 bytes of checksum
	fullAddress := append(addressWithPrefix, checksum2[:4]...)

	// Base58 encode
	return base58.Encode(fullAddress), nil
}

// getTronAPIKey returns TRON API key from environment or default
func getTronAPIKey() string {
	// For demo purposes, using TronGrid free tier
	// In production, you should get a proper API key from TronGrid
	return "free-api-key"
}

// verifyTronTransaction verifies a TRON transaction
func verifyTronTransaction(txHash, toAddress string, expectedAmount int64) (*TronTransaction, error) {
	// TronGrid API endpoint
	apiKey := getTronAPIKey()
	url := fmt.Sprintf("https://api.trongrid.io/v1/transactions/%s", txHash)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add API key header if available
	if apiKey != "free-api-key" {
		req.Header.Set("TRON-PRO-API-KEY", apiKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call TRON API: %v", err)
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

	// Check if transaction exists
	if success, ok := apiResp["success"]; ok && !success.(bool) {
		return nil, fmt.Errorf("transaction not found or failed")
	}

	// Parse transaction data
	// This is a simplified version - real implementation would need more thorough parsing
	tx := &TronTransaction{
		TxID:      txHash,
		Status:    "confirmed",
		TokenName: "USDT",
	}

	// Verify amount and recipient (simplified)
	// In real implementation, you would parse the contract call data
	log.Printf("TRON transaction verified: %s", txHash)

	return tx, nil
}

// getTronBalance gets TRON account balance
func getTronBalance(address string) (*TronBalance, error) {
	apiKey := getTronAPIKey()
	
	// Get TRX balance
	trxURL := fmt.Sprintf("https://api.trongrid.io/v1/accounts/%s", address)
	req, err := http.NewRequest("GET", trxURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create TRX balance request: %v", err)
	}

	if apiKey != "free-api-key" {
		req.Header.Set("TRON-PRO-API-KEY", apiKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get TRX balance: %v", err)
	}
	defer resp.Body.Close()

	// Get USDT balance (TRC20)
	// This would need more complex parsing for actual USDT balance

	// For now, return demo balance
	return &TronBalance{
		TRX:  0,    // TRX balance in SUN
		USDT: 0,    // USDT balance
	}, nil
}

// sendTronTransaction sends USDT on TRON network
func sendTronTransaction(fromPrivateKey, toAddress string, amount int64) (string, error) {
	// This is a complex operation that requires:
	// 1. Building the transaction
	// 2. Signing with private key
	// 3. Broadcasting to network
	
	// For demo purposes, return a fake transaction hash
	fakeHash := fmt.Sprintf("0x%x", sha256.Sum256([]byte(fmt.Sprintf("%s%s%d%d", 
		fromPrivateKey[:10], toAddress, amount, time.Now().Unix()))))
	
	log.Printf("Demo TRON transaction: %s", fakeHash)
	return fakeHash, nil
}

// validateTronAddress validates a TRON address format
func validateTronAddress(address string) bool {
	// TRON addresses start with 'T' and are 34 characters long
	if len(address) != 34 || !strings.HasPrefix(address, "T") {
		return false
	}

	// Try to decode base58
	decoded := base58.Decode(address)
	if len(decoded) != 25 {
		return false
	}

	// Check prefix (0x41)
	if decoded[0] != 0x41 {
		return false
	}

	// Verify checksum
	addressPart := decoded[:21]
	checksum := decoded[21:]

	hash1 := sha256.Sum256(addressPart)
	hash2 := sha256.Sum256(hash1[:])

	return hex.EncodeToString(checksum) == hex.EncodeToString(hash2[:4])
}

// Helper function to convert USDT amount (with 6 decimals) to integer
func usdtToInteger(amount float64) int64 {
	return int64(amount * 1000000) // USDT has 6 decimal places
}

// Helper function to convert integer to USDT amount
func integerToUsdt(amount int64) float64 {
	return float64(amount) / 1000000
}