package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type Result struct {
	Port  int
	State bool
}

func scanPort(host string, port int, timeout time.Duration) Result {
	target := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", target, timeout)

	if err != nil {
		return Result{Port: port, State: false}
	}

	conn.Close()
	return Result{Port: port, State: true}
}

func scanPorts(host string, start, end int, timeout time.Duration) []Result {
	var results []Result
	var wg sync.WaitGroup

	// Create a buffered channel to collect results
	resultChan := make(chan Result, end-start+1)

	//Create a semaphore to limit concurrent goroutines
	// This prevents us from opening too many connections at once
	semaphore := make(chan struct{}, 100) //limit to 100 concurrent goroutines

	// Launch goroutines for each port
	for port := start; port <= end; port++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()

			// Acquire a semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }() // Release the semaphore

			result := scanPort(host, p, timeout)
			resultChan <- result
		}(port)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results from channel
	for result := range resultChan {
		if result.State {
			results = append(results, result)
		}
	}

	return results
}

func main() {

	host := "localhost"
	startport := 1
	endport := 1024
	timeout := time.Millisecond * 500
	fmt.Printf("Scanning host: %s from port %d to %d\n", host, startport, endport)

	startTime := time.Now()
	results := scanPorts(host, startport, endport, timeout)
	elapsed := time.Since((startTime))

	fmt.Printf("\nScan completed in %s\n", elapsed)
	fmt.Printf("Found %d open ports:\n", len(results))
	for _, result := range results {
		fmt.Printf("Port %d is open\n", result.Port)
	}
}
