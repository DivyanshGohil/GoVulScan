package main

import (
	"fmt"
	"net"
	"time"
)

func scanPort(host string, port int, timeout time.Duration) bool {
	target := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", target, timeout)

	if err != nil {
		return false
	}

	conn.Close()
	return true
}

func main() {

	host := "localhost"
	timeout := time.Second * 2
	fmt.Printf("Scanning host: %s\n", host)

	//scan the first 1024 ports
	for port := 1; port <= 1024; port++ {
		if scanPort(host, port, timeout) {
			fmt.Printf("Port %d is open\n", port)
		}
	}

	fmt.Println("Scan complete")
}
