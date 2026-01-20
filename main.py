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

        if not r.ok:
            return "<h1>Wrong Username or Password</h1>"

        resp = r.json()
        if not resp["auth"]:
            return "<h1>Wrong Username or Password</h1>"

        for cookie in r.cookies:
            session[cookie.name] = cookie.value

        session["Auth"] = True
        return redirect("/")

    return render_template("login.html")

@app.route("/")
def index():
    if not session.get("Auth"):
        return redirect("/login")
    return render_template("main.html")


if __name__ == "__main__":
    app.run(port=5000, debug=True)
