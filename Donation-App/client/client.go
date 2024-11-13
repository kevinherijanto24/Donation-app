package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// Check for command-line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run client.go -deposit or go run client.go -withdraw")
		return
	}

	operation := os.Args[1]
	reader := bufio.NewReader(os.Stdin)

	// Get username
	fmt.Print("Enter username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username) // Remove newline characters

	if operation == "-deposit" {
		// Deposit using TCP connection
		tcpConn, err := net.Dial("tcp", "localhost:9000")
		if err != nil {
			fmt.Println("Error connecting to TCP server:", err)
			return
		}
		defer tcpConn.Close()

		// Get amount to deposit
		fmt.Print("Enter amount to add: ")
		amountStr, _ := reader.ReadString('\n')
		amountStr = strings.TrimSpace(amountStr) // Remove newline characters

		// Send username and amount to the TCP server
		tcpConn.Write([]byte(username + "\n"))
		tcpConn.Write([]byte(amountStr + "\n"))

		// Read response from the TCP server
		response := make([]byte, 1024)
		n, _ := tcpConn.Read(response)
		serverResponse := string(response[:n])
		fmt.Println("Server Response:", serverResponse)

		// Check if the response indicates a successful deposit
		if strings.Contains(serverResponse, "new balance") {
			fmt.Println("Deposit Success")
		} else {
			fmt.Println("Deposit Failed")
		}

	} else if operation == "-withdraw" {
		// Withdraw using UDP connection
		udpAddr, err := net.ResolveUDPAddr("udp", "localhost:9001")
		if err != nil {
			fmt.Println("Error resolving UDP address:", err)
			return
		}
		udpConn, err := net.DialUDP("udp", nil, udpAddr)
		if err != nil {
			fmt.Println("Error connecting to UDP server:", err)
			return
		}
		defer udpConn.Close()

		// Get amount to withdraw
		fmt.Print("Enter amount to withdraw: ")
		withdrawAmountStr, _ := reader.ReadString('\n')
		withdrawAmountStr = strings.TrimSpace(withdrawAmountStr) // Remove newline characters

		// Send username and amount to the UDP server
		udpConn.Write([]byte(username + "\n" + withdrawAmountStr + "\n"))


		// Receive response from the UDP server
		udpResponse := make([]byte, 1024)
		n, _, _ := udpConn.ReadFrom(udpResponse)
		udpServerResponse := string(udpResponse[:n])
		fmt.Println("Server Response:", udpServerResponse)

		// Check if the response indicates a successful withdrawal
		if strings.Contains(udpServerResponse, "new balance") {
			fmt.Println("Withdrawal Success")
		} else {
			fmt.Println("Withdrawal Failed")
		}

	} else {
		fmt.Println("Invalid option. Use -deposit or -withdraw")
	}
}
