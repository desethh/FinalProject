package main

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"database/sql"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

var roomsWatchers = struct {
	mu    sync.Mutex
	conns map[*websocket.Conn]bool
}{conns: make(map[*websocket.Conn]bool)}

const MaxClientsPerRoom = 5

type WSIn struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`

	X0    float64 `json:"x0,omitempty"`
	Y0    float64 `json:"y0,omitempty"`
	X1    float64 `json:"x1,omitempty"`
	Y1    float64 `json:"y1,omitempty"`
	Color string  `json:"color,omitempty"`
	Size  float64 `json:"size,omitempty"`
}

type PageData struct {
	Username string
	Password string
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

	Mu sync.Mutex
}

type Message struct {
	Type string `json:"type"`
	User string `json:"user,omitempty"`
	Text string `json:"text,omitempty"`
	Time string `json:"time,omitempty"`

	Current int            `json:"current,omitempty"`
	Max     int            `json:"max,omitempty"`
	Rooms   map[string]int `json:"rooms,omitempty"`
	X0      float64        `json:"x0,omitempty"`
	Y0      float64        `json:"y0,omitempty"`
	X1      float64        `json:"x1,omitempty"`
	Y1      float64        `json:"y1,omitempty"`
	Color   string         `json:"color,omitempty"`
	Size    float64        `json:"size,omitempty"`
}

var rooms = make(map[string]*Room)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func broadcastRoomsUsers() {
	snapshot := map[string]int{}

	for id, room := range rooms {
		room.Mu.Lock()
		snapshot[id] = len(room.Clients)
		room.Mu.Unlock()
	}

	msg := Message{
		Type:  "rooms_users",
		Max:   MaxClientsPerRoom,
		Rooms: snapshot,
	}

	roomsWatchers.mu.Lock()
	defer roomsWatchers.mu.Unlock()

	for c := range roomsWatchers.conns {
		if err := c.WriteJSON(msg); err != nil {
			delete(roomsWatchers.conns, c)
			_ = c.Close()
		}
	}
}

func broadcastUsersCount(room *Room) {
	room.Mu.Lock()
	count := len(room.Clients)
	room.Mu.Unlock()

	msg := Message{
		Type:    "users",
		Current: count,
		Max:     MaxClientsPerRoom,
	}

	for c := range room.Clients {
		_ = c.Conn.WriteJSON(msg)
	}
}

func main() {
	DBopen()
	ctx := context.Background()
	if err := InitGemini(ctx); err != nil {
		log.Fatal("Gemini init error:", err)
	}
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
	http.HandleFunc("/rooms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			rows, err := db.Query("SELECT room_id, owner FROM Rooms")
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			var rooms []Room
			for rows.Next() {
				var rm Room
				if err := rows.Scan(&rm.ID, &rm.Owner); err != nil {
					http.Error(w, "Scan error", http.StatusInternalServerError)
					return
				}
				rooms = append(rooms, rm)
			}
			tmpl := template.Must(template.ParseFiles("templates/room_connection.html"))
			w.Header().Set("Content-Type", "text/html")
			tmpl.Execute(w, rooms)
		}
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

			var isEx int
			row := db.QueryRow("SELECT COUNT(*) FROM Users WHERE username = $1 AND password = $2", username, password)
			row.Scan(&isEx)
			if isEx == 0 {
				http.Error(w, "Wrong Username or Password", http.StatusUnauthorized)
				return
			}

			http.Redirect(w, r, "/page", http.StatusSeeOther)
		}
	})
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
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

			var isEx int
			db.QueryRow("SELECT COUNT(username) FROM Users WHERE username = $1", username).Scan(&isEx)
			if isEx != 0 {
				http.Error(w, "Username already exists", http.StatusConflict)
				return
			}
			_, err := db.Exec("INSERT INTO Users (username, password) VALUES ($1, $2)", username, password)
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}

	})

	// MESSAGES
	http.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		roomID := r.URL.Query().Get("room")
		if roomID == "" {
			http.Error(w, "room is required", http.StatusBadRequest)
			return
		}
		rows, err := db.Query(`
			SELECT username, text, created_at
			FROM messages
			WHERE room_id = $1
			ORDER BY created_at ASC
			LIMIT 200
		`, roomID)
		if err != nil {
			http.Error(w, "db error", 500)
			return
		}
		defer rows.Close()

		var msgs []Message
		for rows.Next() {
			var m Message
			if err := rows.Scan(&m.User, &m.Text, &m.Time); err != nil {
				http.Error(w, "scan error", 500)
				return
			}
			msgs = append(msgs, m)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(msgs)
	})

	// ROOM CREATE
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
		_, err := db.Exec("INSERT INTO Rooms (room_id, owner) VALUES ($1, $2)", roomID, username)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(roomID))
	})
	http.HandleFunc("/ws-rooms", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		roomsWatchers.mu.Lock()
		roomsWatchers.conns[conn] = true
		roomsWatchers.mu.Unlock()

		// сразу отправим snapshot при подключении
		broadcastRoomsUsers()

		// держим соединение открытым
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				roomsWatchers.mu.Lock()
				delete(roomsWatchers.conns, conn)
				roomsWatchers.mu.Unlock()
				_ = conn.Close()
				break
			}
		}
	})

	http.HandleFunc("/rooms-stats", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT room_id FROM Rooms")
		if err != nil {
			http.Error(w, "db error", 500)
			return
		}
		defer rows.Close()

		stats := map[string]int{}

		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				http.Error(w, "scan error", 500)
				return
			}
			if rm, ok := rooms[id]; ok && rm != nil {
				stats[id] = len(rm.Clients)
			} else {
				stats[id] = 0
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"max":   MaxClientsPerRoom,
			"rooms": stats,
		})
	})

	// WEB SOCKET
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		roomID := r.URL.Query().Get("room")
		username := r.URL.Query().Get("username")

		room, ok := rooms[roomID]
		if !ok {
			var owner string
			err := db.QueryRow(`SELECT owner FROM rooms WHERE room_id=$1`, roomID).Scan(&owner)
			if err == sql.ErrNoRows {
				http.Error(w, "Room not found", http.StatusNotFound)
				return
			}
			if err != nil {
				http.Error(w, "DB error", 500)
				return
			}
			room = &Room{ID: roomID, Owner: owner, Clients: make(map[*Client]bool)}
			rooms[roomID] = room
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := &Client{
			Username: username,
			Conn:     conn,
		}

		room.Mu.Lock()
		room.Clients[client] = true
		if len(room.Clients) >= MaxClientsPerRoom {
			room.Mu.Unlock()
			_ = conn.WriteMessage(websocket.TextMessage, []byte("Room is full (max 5)"))
			_ = conn.Close()
			return
		}
		room.Mu.Unlock()

		broadcastUsersCount(room)
		broadcastRoomsUsers()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				room.Mu.Lock()
				delete(room.Clients, client)
				room.Mu.Unlock()

				broadcastUsersCount(room)
				broadcastRoomsUsers()
				break
			}

			var in WSIn
			if err := json.Unmarshal(msg, &in); err != nil {
				in = WSIn{Type: "chat", Text: string(msg)}
			}

			switch in.Type {
			case "chat":
				text := strings.TrimSpace(in.Text)

				if strings.HasPrefix(text, "/gpt ") {
					prompt := strings.TrimSpace(strings.TrimPrefix(text, "/gpt "))

					answer, err := CallGPT(ctx, prompt)
					if err != nil {
						answer = "Ошибка GPT: " + err.Error()
					}

					gptMsg := Message{
						Type: "chat",
						User: "GPT",
						Text: answer,
						Time: time.Now().Format("15:04:05"),
					}

					_, _ = db.Exec(
						`INSERT INTO messages (room_id, username, text) VALUES ($1, $2, $3)`,
						roomID, gptMsg.User, gptMsg.Text,
					)

					for c := range room.Clients {
						_ = c.Conn.WriteJSON(gptMsg)
					}

					continue
				}

				chatMsg := Message{
					Type: "chat",
					User: username,
					Text: text,
					Time: time.Now().Format("15:04:05"),
				}

				_, _ = db.Exec(
					`INSERT INTO messages (room_id, username, text) VALUES ($1, $2, $3)`,
					roomID, chatMsg.User, chatMsg.Text,
				)

				for c := range room.Clients {
					_ = c.Conn.WriteJSON(chatMsg)
				}

			case "draw":
				drawMsg := Message{
					Type: "draw",
					X0:   in.X0, Y0: in.Y0,
					X1: in.X1, Y1: in.Y1,
					Color: in.Color,
					Size:  in.Size,
				}

				for c := range room.Clients {
					_ = c.Conn.WriteJSON(drawMsg)
				}
			}

		}
	})

	http.ListenAndServe(":8080", nil)
}

var db *sql.DB

func DBopen() {
	dsn := "host=localhost port=5432 user=postgres password=gotban7d dbname=postgres sslmode=disable"

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Ошибка sql.Open:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("База недоступна:", err)
	}

	log.Println("База подключена")
}
