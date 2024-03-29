package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/novalagung/gubrak/v2"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

const MESSAGE_NEW_USER = "New User"
const MESSAGE_CHAT = "Chat"
const MESSAGE_LEAVE = "Leave"

var connections = make([]*WebSocketConnection, 0)

type SocketPayload struct {
	Message string
}

type SocketResponse struct {
	From    string
	Type    string
	Message string
}

type WebSocketConnection struct {
	*websocket.Conn
	Username string
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		content, err := ioutil.ReadFile("index.html")
		if err != nil {
			http.Error(w, "Error: cannot open file", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%s", content)
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		currentGorillaConn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
		if err != nil {
			http.Error(w, "Cannot establish connection", http.StatusBadRequest)
		}
		username := r.URL.Query().Get("username")
		currentConn := WebSocketConnection{Conn: currentGorillaConn, Username: username}
		connections = append(connections, &currentConn)

		go handleIO(&currentConn)
	})

	fmt.Println("Server running on localhost:8080")
	http.ListenAndServe(":8080", nil)

}

func handleIO(currentConn *WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR", fmt.Sprintf("%v", r))
		}
	}()
	broadcastMessage(currentConn, MESSAGE_NEW_USER, "")
	for {
		payload := SocketPayload{}
		err := currentConn.ReadJSON(&payload)
		if err != nil {
			if strings.Contains(err.Error(), "websocket: close") {
				broadcastMessage(currentConn, MESSAGE_LEAVE, "")
				ejectConnection(currentConn)
				return
			}
			log.Println("ERROR", err.Error())
			continue
		}
		broadcastMessage(currentConn, MESSAGE_CHAT, payload.Message)
	}
}
func ejectConnection(currentCon *WebSocketConnection) {
	filtered := gubrak.From(connections).Reject(func(each *WebSocketConnection) bool {
		return each == currentCon
	}).Result()
	connections = filtered.([]*WebSocketConnection)
}

func broadcastMessage(currentConn *WebSocketConnection, kind, message string) {
	for _, eachConn := range connections {
		if eachConn == currentConn {
			continue
		}

		eachConn.WriteJSON(SocketResponse{
			From:    currentConn.Username,
			Type:    kind,
			Message: message,
		})
	}

}
