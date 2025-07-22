package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

// getEnv function remains the same
func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// generateNextPath generates sequential paths like a, b, ..., z, aa, ab, ...
func generateNextPath(currentPath string) string {
	if currentPath == "" {
		return "a" // Start with 'a' if no current path
	}

	runes := []rune(currentPath)
	n := len(runes)
	i := n - 1

	// Iterate from the rightmost character
	for i >= 0 {
		if runes[i] == 'z' {
			runes[i] = 'a' // Reset to 'a' if 'z'
			i--            // Move to the next character to the left
		} else {
			runes[i]++ // Increment the character
			return string(runes)
		}
	}

	// If all characters were 'z' (e.g., "z", "zz"), prepend 'a'
	return "a" + string(runes)
}

// worker for Endpoint Exploration Attacker
// This worker now generates sequential paths for exploration.
func worker(wg *sync.WaitGroup, client *http.Client, method string, baseURL string, id int) {
	defer wg.Done()

	currentPathSegment := "" // Each worker maintains its own path segment to generate

	for {
		// Generate the next path segment (e.g., "a", "b", "aa")
		currentPathSegment = generateNextPath(currentPathSegment)
		// Construct the full URL. Assuming paths are relative to the root.
		// You might want to prepend a base path like "/explore/" if your IPS expects it.
		targetURL := baseURL + "/" + currentPathSegment

		resp, err := client.Get(targetURL)
		if err != nil {
			fmt.Printf("[worker %d] error: %v\n", id, err)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 403 {
			fmt.Printf("[worker %d] was banned 403 Forbidden, stopping\n", id)
			return
		}
		fmt.Println("[worker", id, "]", "fired request to", targetURL, "status code:", resp.StatusCode)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// Introduce a delay between requests to simulate exploration rather than pure DDoS
		// This can be tuned to be faster or slower depending on the desired exploration speed.
	}
}

func main() {
	baseURL := getEnv("target_host", "http://ips:8080")
	methodStr := getEnv("target_method", "GET") // GET is most common for exploration/scanning
	concurrency := getEnv("concurrency", "5")   // Lower concurrency for exploration

	numWorkers := 5 // Default concurrency
	fmt.Sscanf(concurrency, "%d", &numWorkers)

	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	client := &http.Client{
		Timeout: 10 * time.Second, // Increased timeout for potentially slower responses from diverse endpoints
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		// Workers now generate their own paths, so no routes slice needed here
		go worker(&wg, client, methodStr, baseURL, i)
	}

	wg.Wait()
}
