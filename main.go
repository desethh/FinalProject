package main

import (
	"encoding/json"
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

func LoginPage(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	var uid int
	err := DB.QueryRow("SELECT id FROM users WHERE username=$1 AND password=$2", username, password).Scan(&uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return
		}
		log.Println("DB error:", err)
		return
	}
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
		Auth:     true,
		Username: username,
		Password: "",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "user-session")
	auth, ok := session.Values["authenticated"].(bool)
	if !ok || !auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	w.Write([]byte("Main Page Accessed"))

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
	http.HandleFunc("/", mainPage)
	http.HandleFunc("/login", LoginPage)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
