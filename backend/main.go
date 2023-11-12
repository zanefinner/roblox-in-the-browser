package main

import (
	"encoding/json"
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

type GameState struct {
	Players map[string]PlayerPosition `json:"players"`
}

type PlayerPosition struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Player string  `json:"player"`
}

// Client struct to store WebSocket connection and ID
type Client struct {
	ID     string          `json:"id"`
	Conn   *websocket.Conn `json:"-"`
	Method string          `json:"method"`
	Game   *GameState      `json:"game"`
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
			ID:     clientID,
			Conn:   conn,
			Method: "some_method", // Set your desired default method
			Game:   &GameState{Players: make(map[string]PlayerPosition)},
		}

		// Lock the mutex before modifying the clients map
		clientsMutex.Lock()
		clients[clientID] = client
		clientsMutex.Unlock()

		// Run forever
		// Inside the WebSocket handler loop
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				clientsMutex.Lock()
				delete(clients, clientID)
				clientsMutex.Unlock()
				return
			}

			var message map[string]interface{}
			if err := json.Unmarshal(msg, &message); err != nil {
				fmt.Println("Error decoding JSON message:", err)
				continue
			}

			switch message["method"] {
			case "update_position":
				// Handle a message to update player position
				fmt.Println("update posi")
				if client.Game == nil {
					client.Game = &GameState{Players: make(map[string]PlayerPosition)}
				}

				client.Game.Players[clientID] = PlayerPosition{
					X:      message["x"].(float64),
					Y:      message["y"].(float64),
					Player: clientID,
				}

				// Broadcast the updated game state to all clients
				broadcastGameState()

			// Add more cases for other game-related messages

			default:
				fmt.Println("Unknown method:", message["method"])
			}
		}

	})

	http.HandleFunc("/getClients", func(resp http.ResponseWriter, req *http.Request) {
		// Get a JSON representation of the clients
		clientsMutex.Lock()
		clientList := make([]*Client, 0, len(clients))
		for _, client := range clients {
			clientList = append(clientList, client)
		}
		clientsMutex.Unlock()

		// Convert the client list to JSON
		jsonData, err := json.Marshal(clientList)
		if err != nil {
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Send the JSON data as the response
		resp.Header().Set("Content-Type", "application/json")
		resp.Write(jsonData)
	})

	http.ListenAndServe(":8000", nil)
}

func broadcastGameState() {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	gameState := &GameState{
		Players: make(map[string]PlayerPosition),
	}

	for _, client := range clients {
		for _, player := range client.Game.Players {
			gameState.Players[player.Player] = player
		}
	}

	jsonData, err := json.Marshal(gameState)
	if err != nil {
		fmt.Println("Error encoding game state:", err)
		return
	}

	for _, client := range clients {
		if err := client.Conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
			fmt.Println("Error broadcasting game state to client:", err)
		}
	}
}
