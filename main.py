from flask import Flask, render_template, request, redirect, session
import requests

app = Flask(__name__)
app.secret_key = "super-secret-key"

GO_BACKEND = "http://localhost:8080"

@app.route("/login", methods=["GET", "POST"])
def login():
    if request.method == "POST":
        data = {
            "username": request.form["username"],
            "password": request.form["password"]
        }

        r = requests.post(f"{GO_BACKEND}/login", data=data)

        if r.status_code == 200:
            session["auth"] = True
            return redirect("/")
        else:
            return "Login failed", 401

    return render_template("login.html")

@app.route("/register", methods=["GET", "POST"])
def register():
    data = {
        isdone: request.form["isdone"],
    }
    if request.method == "POST":
        r = requests.post(f"{GO_BACKEND}/register", data=data)
        if r.status_code == 200:
            auth = r.json()
            if auth == True:
                return redirect("/login")
        else:
            return "Registration failed", 400
    return render_template("register.html")

@app.route("/")
def index():
    if not session.get("auth"):
        return redirect("/login")

    return "<h1>Welcome to collaborative board</h1>"

if __name__ == "__main__":
    app.run(port=5000, debug=True)
