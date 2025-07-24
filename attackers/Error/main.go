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

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func worker(wg *sync.WaitGroup, client *http.Client, baseURL string, loginPath string, id int) {
	defer wg.Done()

	targetURL := baseURL + loginPath
	method := "POST"

	for {

		incorrectPayload := map[string]interface{}{}

		bodyBytes, err := json.Marshal(incorrectPayload)

		if err != nil {
			fmt.Printf("[worker %d] json marshal error: %v\n", id, err)
			time.Sleep(1 * time.Second)
			continue
		}

		req, err := http.NewRequest(method, targetURL, bytes.NewReader(bodyBytes))
		if err != nil {
			fmt.Printf("[worker %d] request error: %v\n", id, err)
			time.Sleep(1 * time.Second)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Length", fmt.Sprintf("%d", len(bodyBytes)))
		req.Header.Set("User-Agent", "HighErrorAttackerBot/1.0")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[worker %d] error: %v\n", id, err)
			time.Sleep(time.Duration(rand.Intn(200)+50) * time.Millisecond)
			continue
		}

		_, err = io.ReadAll(resp.Body)
		// bodyContent, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("[worker %d] error reading response body: %v\n", id, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		// if string(bodyContent) == "Blocked by IPS." {
		// 	fmt.Println("[worker", id, "]", "Blocked by IPS, stopping")
		// 	return
		// }
		fmt.Printf("[worker %d] fired %s request to %s with incorrect payload. Status code: %d\n", id, method, targetURL, resp.StatusCode)

		time.Sleep(time.Duration(rand.Intn(100)+10) * time.Millisecond)
	}
}

func main() {
	baseURL := getEnv("target_host", "http://ips:8080")
	loginRoute := getEnv("target_login_route", "/login")
	concurrency := getEnv("concurrency", "10")

	numWorkers := 10
	fmt.Sscanf(concurrency, "%d", &numWorkers)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(&wg, client, baseURL, loginRoute, i)
	}

	wg.Wait()
}
