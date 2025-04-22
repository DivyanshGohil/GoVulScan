package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type ScanResult struct {
	Port    int
	State   bool
	Service string
	Banner  string
	Version string
}

func grabBanner(host string, port int, timeout time.Duration) (string, error) {
	target := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", target, timeout)

	if err != nil {
		return "", err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(timeout))

	//some services need a trigger to send data
	// Send a simple HTTP request for web servers

	if port == 80 || port == 443 || port == 8080 {
		fmt.Fprint(conn, "HEAD / HTTP/1.0\r\n\r\n")
	} else {
		// For other services, just wait for the banner
		// Some services may require specific triggers
	}

	// Read the response
	reader := bufio.NewReader(conn)
	banner, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(banner), nil
}

func identifyService(port int, banner string) (string, string) {
	commonPorts := map[int]string{
		21:    "FTP",
		22:    "SSH",
		23:    "Telnet",
		25:    "SMTP",
		53:    "DNS",
		80:    "HTTP",
		110:   "POP3",
		143:   "IMAP",
		443:   "HTTPS",
		3306:  "MySQL",
		5432:  "PostgreSQL",
		6379:  "Redis",
		8080:  "HTTP-Proxy",
		27017: "MongoDB",
	}

	// Try to identify service from common ports
	service := "Unkown"
	if s, exists := commonPorts[port]; exists {
		service = s
	}

	version := "Unknown"

	lowerBanner := strings.ToLower(banner)

	//SSH version detection
	if strings.Contains(lowerBanner, "ssh") {
		service = "SSH"
		parts := strings.Split(banner, " ")
		if len(parts) > 2 {
			version = parts[1]
		}
	}

	//HTTP server detection
	if strings.Contains(lowerBanner, "http") || strings.Contains(lowerBanner, "apache") || strings.Contains(lowerBanner, "nginx") {
		if port == 443 {
			service = "HTTPS"
		} else {
			service = "HTTP"
		}

		// Try to find server info in format "Server: Apache/2.4.41"
		if strings.Contains(banner, "Server:") {
			parts := strings.Split(banner, "Server:")
			if len(parts) >= 2 {
				version = strings.TrimSpace(parts[1])
			}
		}
	}

	return service, version
}

// type ScanResult struct {
// 	Port  int
// 	State bool
// }

func scanPort(host string, port int, timeout time.Duration) ScanResult {
	target := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", target, timeout)

	if err != nil {
		return ScanResult{Port: port, State: false}
	}

	conn.Close()

	banner, err := grabBanner(host, port, timeout)

	service := "Unknown"
	version := "Unknown"

	if err == nil && banner != "" {
		service, version = identifyService(port, banner)
	}

	return ScanResult{
		Port:    port,
		State:   true,
		Service: service,
		Banner:  banner,
		Version: version,
	}
}

func scanPorts(host string, start, end int, timeout time.Duration) []ScanResult {
	var results []ScanResult
	var wg sync.WaitGroup

	// Create a buffered channel to collect results
	resultChan := make(chan ScanResult, end-start+1)

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
	timeout := time.Millisecond * 800
	fmt.Printf("Scanning host: %s from port %d to %d\n", host, startport, endport)

	startTime := time.Now()
	results := scanPorts(host, startport, endport, timeout)
	elapsed := time.Since((startTime))

	fmt.Printf("\nScan completed in %s\n", elapsed)
	fmt.Printf("Found %d open ports:\n", len(results))

	fmt.Println("Port\tService\tVersion\tBanner")
	fmt.Println("----\t-------\t-------\t------")
	for _, result := range results {
		bannerPreview := ""
		if len(result.Banner) > 30 {
			bannerPreview = result.Banner[:30] + "..."
		} else {
			bannerPreview = result.Banner
		}
		fmt.Printf("%d\t%s\t%s\t%s\n", result.Port, result.Service, result.Version, bannerPreview)
		// fmt.Printf("Port %d is open\n", result.Port)
	}
}
