package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"database/sql"

	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
)

var err error
var DB *sql.DB
var store = sessions.NewCookieStore([]byte("pm2zlsz1PdlU8ymTwD4T2UIXpFy6qqzo"))

type User struct {
	Auth     bool   `json:"auth"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Status struct {
	status bool
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	session, ok := store.Get(r, "user-session")
	if ok != nil {
		http.Error(w, "No sessesion found", http.StatusBadRequest)
	}
	username := session.Values["username"].(string)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"username": username,
	})
}

func LoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	} else {
		http.SetCookie(w, &http.Cookie{
			Name:     "auth",
			Value:    "true",
			Path:     "/",
			HttpOnly: true,
		})
		username := r.FormValue("username")
		password := r.FormValue("password")

		var isEx int
		row := DB.QueryRow("SELECT COUNT(*) FROM Users WHERE username = $1 AND password = $2", username, password)
		row.Scan(&isEx)

		if isEx == 0 {
			http.Error(w, "Wrong Username or Password", http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(User{Auth: false})
			return
		}
		userid := DB.QueryRow("SELECT uid from Users WHERE username = ?", username)
		var uid int
		userid.Scan(&uid)
		session, _ := store.Get(r, "user-session")
		session.Values["authenticated"] = true
		session.Values["user-id"] = uid
		session.Values["username"] = username

		err = session.Save(r, w)
		if err != nil {
			http.Error(w, "Cannot save session", http.StatusInternalServerError)
			return
		}

		user := User{
			Auth: true,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

func RegPage(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	var IsEx int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = $1", username).Scan(&IsEx)

	if err != nil {
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if IsEx > 0 {
		http.Error(w, "This username already exists", http.StatusBadRequest)
		return
	}

	_, err = DB.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", username, password)
	if err != nil {
		log.Println("DB error:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	regdone := map[string]bool{"success": true}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(regdone)
}

func main() {
	dsn := "postgresql://postgres.dwbaizkhsefnvtxfjtlw:xExWqy5wQTcH4tX8@aws-1-ap-south-1.pooler.supabase.com:6543/postgres"
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Ошибка sql.Open:", err)
	}

	if err := DB.Ping(); err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	http.HandleFunc("/login", LoginPage)
	http.HandleFunc("/register", RegPage)
	http.HandleFunc("/user", GetUser)
	log.Fatal(http.ListenAndServe(":8080", nil))
	fmt.Println("Server Listen and Server on port :8080")
}
