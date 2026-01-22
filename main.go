package main

import (
	"html/template"
	"net/http"
)

type PageData struct {
	Username string
	Password string
}

type DB struct {
	username string
	password string
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

	http.ListenAndServe(":8080", nil)
}
