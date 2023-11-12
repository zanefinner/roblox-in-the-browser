package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[string]*websocket.Conn)
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

		// Lock the mutex before modifying the clients map
		clientsMutex.Lock()
		clients[conn.RemoteAddr().String()] = conn
		clientsMutex.Unlock()

		// Run forever
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				// Remove the connection from the clients map on error
				clientsMutex.Lock()
				delete(clients, conn.RemoteAddr().String())
				clientsMutex.Unlock()
				return
			}

			fmt.Printf("%s send: %s\n", conn.RemoteAddr(), string(msg))

			// Lock the mutex before iterating over the clients map
			clientsMutex.Lock()
			for _, client := range clients {
				if err = client.WriteMessage(msgType, msg); err != nil {
					// Handle error writing to client if needed
				}
			}
			clientsMutex.Unlock()
		}
	})

	http.ListenAndServe(":8000", nil)
}
