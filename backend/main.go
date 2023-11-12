package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client struct to store WebSocket connection and ID
type Client struct {
	ID     string
	Conn   *websocket.Conn
	Method string
}

var clients = make(map[string]*Client)
var clientsMutex sync.Mutex

func main() {
	// Socket endpoint
	http.HandleFunc("/echo", func(resp http.ResponseWriter, req *http.Request) {
		// Config init
		conn, err := upgrader.Upgrade(resp, req, nil)
		if err != nil {
			fmt.Println("Error upgrading to WebSocket:", err)
			return
		}

		// Generate a unique ID for the client
		clientID := uuid.New().String()

		// Create a Client struct with ID and WebSocket connection
		client := &Client{
			ID:   clientID,
			Conn: conn,
		}

		// Lock the mutex before modifying the clients map
		clientsMutex.Lock()
		clients[clientID] = client
		clientsMutex.Unlock()

		// Run forever
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				// Remove the connection from the clients map on error
				clientsMutex.Lock()
				delete(clients, clientID)
				clientsMutex.Unlock()
				return
			}

			fmt.Printf("%s send: %s\n", clientID, string(msg))

			// Lock the mutex before iterating over the clients map
			clientsMutex.Lock()
			for _, otherClient := range clients {
				if err = otherClient.Conn.WriteMessage(msgType, msg); err != nil {
					// Handle error writing to client if needed
				}
			}
			clientsMutex.Unlock()
		}
	})

	http.ListenAndServe(":8000", nil)
}
