package main

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"database/sql"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

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
}

type Message struct {
	User string `json:"user"`
	Text string `json:"text"`
	Time string `json:"time"`
}

var rooms = make(map[string]*Room)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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

		room.Clients[client] = true

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				delete(room.Clients, client)
				conn.Close()
				break
			}
			text := string(msg)
			if strings.HasPrefix(text, "/gpt ") {
				prompt := strings.TrimSpace(strings.TrimPrefix(`
				Answer in Markdown.
				Use headings, bullet points, and code blocks where appropriate.

				Question:
				`+text, "/gpt "))
				ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
				answer, gerr := CallGPT(ctx, prompt)
				cancel()

				if gerr != nil {
					answer = "Ошибка GPT: " + gerr.Error()
				}

				aiMsg := Message{User: "GPT", Text: answer}

				if _, err := db.Exec(
					`INSERT INTO messages (room_id, username, text) VALUES ($1, $2, $3)`,
					roomID, aiMsg.User, aiMsg.Text,
				); err != nil {
					log.Println("insert ai message error:", err)
				}

				for c := range room.Clients {
					_ = c.Conn.WriteJSON(aiMsg)
				}

				continue
			}
			_, err = db.Exec("INSERT INTO messages (room_id, username, text) VALUES ($1, $2, $3)", roomID, username, string(msg))
			if err != nil {
				log.Println("Database error:", err)
			}
			message := Message{
				User: username,
				Text: text,
			}

			for c := range room.Clients {
				c.Conn.WriteJSON(message)
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
