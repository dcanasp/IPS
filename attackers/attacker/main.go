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

func worker(wg *sync.WaitGroup, client *http.Client, method string, url string, id int) {
	defer wg.Done()
	for {
		payload := payload{
			Name:        "string",
			Description: "string",
			Password:    "string",
			Email:       "string",
			Phone:       "string",
			Address:     "string",
			City:        "string",
			State:       "string",
			Zip:         "string",
		}

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("[worker %d] json marshal error: %v\n", id, err)
			continue
		}

		req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
		if err != nil {
			fmt.Printf("[worker %d] request error: %v\n", id, err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Length", fmt.Sprintf("%d", len(bodyBytes)))

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[worker %d] error: %v\n", id, err)
			continue
		}
		fmt.Println("[worker", id, "]", "fired request to", url, "status code:", resp.StatusCode)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		time.Sleep(time.Duration(30 * time.Second))
	}
}

func main() {
	baseURL := getEnv("target_host", "http://ips:8080")
	routeStr := getEnv("target_routes", "/ddos/login")
	methodStr := getEnv("target_method", "POST")
	concurrency := getEnv("concurrency", "10")

	routes := strings.Split(routeStr, ",")
	numWorkers := 10
	fmt.Sscanf(concurrency, "%d", &numWorkers)

	rand.Seed(time.Now().UnixNano())

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(&wg, client, methodStr, baseURL+routes[0], i)
	}

	wg.Wait()
}
