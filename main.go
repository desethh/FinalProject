package main

import (
	"html/template"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type PageData struct {
	Username string
	Password string
}

type DB struct {
	username string
	password string
}

type Client struct {
	Username string
	Conn     *websocket.Conn
}

type Room struct {
	ID       string
	Owner    string
	Clients  map[*Client]bool
	Messages []Message
}

type Message struct {
	User string `json:"user"`
	Text string `json:"text"`
}

var rooms = make(map[string]*Room)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	tmpl := template.Must(template.ParseFiles("templates/main.html"))

	http.HandleFunc("/page", func(w http.ResponseWriter, r *http.Request) {
		username := r.Header.Get("X-Username")

		data := PageData{
			Username: username,
		}

		w.Header().Set("Content-Type", "text/html")
		tmpl.Execute(w, data)
	})
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl := template.Must(template.ParseFiles("templates/login.html"))
			w.Header().Set("Content-Type", "text/html")
			if err := tmpl.Execute(w, nil); err != nil {
				http.Error(w, "Template error", http.StatusInternalServerError)
			}
			return
		}

		if r.Method == http.MethodPost {
			username := r.FormValue("username")
			password := r.FormValue("password")

			details := DB{
				username: "Tayir",
				password: "test",
			}

			if username != details.username || password != details.password {
				http.Error(w, "Wrong Username or Password", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/page", http.StatusSeeOther)
		}
	})

	http.HandleFunc("/create-room", func(w http.ResponseWriter, r *http.Request) {
		username := r.Header.Get("X-Username")
		if username == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		roomID := uuid.New().String()

		rooms[roomID] = &Room{
			ID:      roomID,
			Owner:   username,
			Clients: make(map[*Client]bool),
		}

		w.Write([]byte(roomID))
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		username := r.Header.Get("X-Username")
		roomID := r.URL.Query().Get("room")

		room, ok := rooms[roomID]
		if !ok {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := &Client{
			Username: username,
			Conn:     conn,
		}

		room.Clients[client] = true
		for _, msg := range room.Messages {
			client.Conn.WriteJSON(msg)
		}

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				delete(room.Clients, client)
				conn.Close()
				break
			}

			message := Message{
				User: username,
				Text: string(msg),
			}
			room.Messages = append(room.Messages, message)

			for c := range room.Clients {
				c.Conn.WriteJSON(message)
			}

		}
	})

	http.ListenAndServe(":8080", nil)
}
