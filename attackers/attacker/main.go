package main

import (
	"fmt"
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
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("[worker %d] request error: %v\n", id, err)
			continue
		}
		// if resp.StatusCode == 403 {
		// 	fmt.Printf("[worker %d] was banned 403 Forbidden, stopping\n", id)
		// 	return
		// }
		fmt.Print("[worker", id, "]", "fired request to", url, "status code:", resp.StatusCode, "\n")
		defer resp.Body.Close() // Ensure the response body is closed
	}
}

func main() {
	baseURL := getEnv("target_host", "http://ips:8080")
	routeStr := getEnv("target_routes", "/")
	methodStr := getEnv("target_method", "GET")
	concurrency := getEnv("concurrency", "1")

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
