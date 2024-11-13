package main

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"sort"
	"github.com/gorilla/websocket"
)

// User struct to hold user data
type User struct {
	Name    string
	Balance int
}

var (
	users           = make(map[string]*User) // Store users
	clients         = make(map[*websocket.Conn]bool)
	broadcast       = make(chan []User)
	upgrader        = websocket.Upgrader{}
	mu              sync.Mutex
	tcpAddress      = "localhost:9000" // TCP server address
	udpAddress      = "localhost:9001" // UDP server address
	fakeTransactionTicker *time.Ticker
)

func main() {
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/donate", handleDonate)
	http.HandleFunc("/ws", handleConnections)

	go handleMessages()
	go startTCPServer() // Start TCP server
	go startUDPServer() // Start UDP server

	// Initialize ticker to trigger every 1 seconds
	fakeTransactionTicker = time.NewTicker(1 * time.Second)
	go fakeTransactionHandler()

	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
	updateClients()
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		name := r.FormValue("name")

		mu.Lock()
		if _, exists := users[name]; !exists {
			users[name] = &User{Name: name, Balance: 0}
			fmt.Println("User registered:", name)
		}
		mu.Unlock()

		// Immediately update clients after registration
		updateClients()
	}
}

func handleDonate(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		from := r.FormValue("from")
		to := r.FormValue("to")
		amount, err := strconv.Atoi(r.FormValue("amount"))
		if err != nil {
			displayError(w, "Invalid amount format")
			return
		}

		mu.Lock()
		defer mu.Unlock()

		// Check if both users exist
		userFrom, okFrom := users[from]
		userTo, okTo := users[to]
		if !okFrom {
			displayError(w, "Sender not found")
			return
		}
		if !okTo {
			displayError(w, "Recipient not found")
			return
		}

		// Check if the sender has enough balance
		if userFrom.Balance < amount {
			displayError(w, "Insufficient balance")
			return
		}

		// Process the donation
		userFrom.Balance -= amount
		userTo.Balance += amount
		fmt.Printf("%s donated %d to %s\n", from, amount, to)
		updateClients() // Notify WebSocket clients of the updated balance

		// Redirect to home page on successful donation
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// displayError sends an error message to the client with a 5-second redirect script
func displayError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
		<html>
			<head>
				<title>Error</title>
				<meta http-equiv="refresh" content="5;url=/" />
			</head>
			<body>
				<h2>%s</h2>
				<p>You will be redirected to the home page in 5 seconds...</p>
				<script>
					setTimeout(function() {
						window.location.href = "/";
					}, 5000);
				</script>
			</body>
		</html>
	`, message)
}



func handleConnections(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WebSocket Upgrade Error:", err)
		return
	}

	// Add client to the clients map
	mu.Lock()
	clients[ws] = true
	mu.Unlock()
	fmt.Println("New WebSocket client connected")

	// Keep connection open to receive updates
	for {
		_, _, err := ws.ReadMessage()
		updateClients()
		if err != nil {
			fmt.Println("WebSocket Read Error:", err)
			// Remove client if disconnected
			mu.Lock()
			delete(clients, ws)
			mu.Unlock()
			ws.Close()
			break
		}
	}
}

func handleMessages() {
	for {
		// Wait for new users list in the broadcast channel
		usersList := <-broadcast

		// Send users list to all connected WebSocket clients
		mu.Lock()
		for client := range clients {
			err := client.WriteJSON(usersList)
			if err != nil {
				fmt.Println("WebSocket Write Error:", err)
				client.Close()
				delete(clients, client)
			}
		}
		mu.Unlock()
	}
}

// Collect all users and send the update to broadcast channel
func updateClients() {
	usersList := make([]User, 0, len(users))
	for _, user := range users {
		usersList = append(usersList, *user)
	}
	    // Sort the usersList based on a specific criterion
		sort.Slice(usersList, func(i, j int) bool {
			// Assuming User has a 'Name' field that you want to sort by alphabetically
			return usersList[i].Name < usersList[j].Name
		})
	broadcast <- usersList

}

// Start TCP server to handle balance updates
func startTCPServer() {
	listen, err := net.Listen("tcp", tcpAddress)
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		return
	}
	defer listen.Close()

	fmt.Println("TCP Server started at", tcpAddress)

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handleTCPConnection(conn)
	}
}

// Handle TCP client connection
func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	// Request username from the client
	usernameBuffer := make([]byte, 1024)
	n, err := conn.Read(usernameBuffer)
	if err != nil {
		fmt.Println("Error reading username:", err)
		conn.Write([]byte("Error reading username\n"))
		return
	}
	username := strings.TrimSpace(string(usernameBuffer[:n]))

	// Request amount from the client
	amountBuffer := make([]byte, 1024)
	n, err = conn.Read(amountBuffer)
	if err != nil {
		fmt.Println("Error reading amount:", err)
		conn.Write([]byte("Error reading amount\n"))
		return
	}
	amountStr := strings.TrimSpace(string(amountBuffer[:n]))

	// Convert amount to integer and validate
	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		fmt.Println("Invalid amount:", amountStr)
		conn.Write([]byte("Invalid amount. Please enter a valid integer.\n"))
		return
	}

	// Lock the user map for thread safety
	mu.Lock()
	defer mu.Unlock()

	// Update the user's balance
	if user, exists := users[username]; exists {
		user.Balance += amount
		fmt.Printf("%s's balance updated by %d\n", username, amount)
		// Send updated balance to the client
		conn.Write([]byte(fmt.Sprintf("%s's new balance: $%d\n", username, user.Balance)))
		// Notify WebSocket clients of the updated balance
		updateClients()
	} else {
		conn.Write([]byte("User not found\n"))
	}
}

// Start UDP server to handle balance withdrawal requests
func startUDPServer() {
	addr, err := net.ResolveUDPAddr("udp", udpAddress)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error starting UDP server:", err)
		return
	}
	defer conn.Close()

	fmt.Println("UDP Server started at", udpAddress)

	for {
		// Read data from UDP client
		buffer := make([]byte, 1024)
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading from UDP client:", err)
			continue
		}

		// Parse username and amount
		data := strings.SplitN(strings.TrimSpace(string(buffer[:n])), "\n", 2)
		if len(data) < 2 {
			conn.WriteToUDP([]byte("Invalid request format\n"), clientAddr)
			continue
		}
		username := data[0]
		amountStr := data[1]

		// Convert amount to integer and validate
		amount, err := strconv.Atoi(amountStr)
		if err != nil {
			fmt.Println("Invalid amount:", amountStr)
			conn.WriteToUDP([]byte("Invalid amount. Please enter a valid integer.\n"), clientAddr)
			continue
		}

		// Lock the user map for thread safety
		mu.Lock()
		user, exists := users[username]
		if exists && user.Balance >= amount {
			user.Balance -= amount
			fmt.Printf("%s's balance deducted by %d\n", username, amount)
			conn.WriteToUDP([]byte(fmt.Sprintf("%s's new balance: $%d\n", username, user.Balance)), clientAddr)
			// Notify WebSocket clients of the updated balance
			updateClients()
		} else if !exists {
			conn.WriteToUDP([]byte("User not found\n"), clientAddr)
		} else {
			conn.WriteToUDP([]byte("Insufficient balance\n"), clientAddr)
		}
		mu.Unlock()
	}
}

// Handle fake transaction every 5 seconds
func fakeTransactionHandler() {
	for {
		select {
		case <-fakeTransactionTicker.C:
			mu.Lock()
			// Convert the map of users to a slice and pick the first user
			var firstUser *User
			for _, user := range users {
				firstUser = user
				break // Break after the first user is found
			}

			if firstUser != nil {
				// Add a fake amount to the first user (you can change the amount as needed)
				firstUser.Balance += 0 // You can replace 0 with any fake amount, e.g., rand.Intn(100)
				updateClients() // Notify WebSocket clients
			}
			mu.Unlock()
		}
	}
}

