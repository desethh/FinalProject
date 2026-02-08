from flask import Flask, request, session, Response, redirect, render_template
import requests

app = Flask(__name__, static_folder="static")
app.secret_key = "secret"

GO_BACKEND = "http://localhost:8080"

@app.route("/static/<path:path>")
def static_proxy(path):
    r = requests.get(f"{GO_BACKEND}/static/{path}")
    return Response(
        r.content,
        status=r.status_code,
        content_type=r.headers.get("Content-Type")
    )

@app.route("/")
def index():
    if not session.get("auth"):
        return redirect("/login")

    headers = {
        "X-Username": session["username"]
    }

    r = requests.get(f"{GO_BACKEND}/page", headers=headers)

    return Response(
        r.content,
        status=r.status_code,
        content_type=r.headers.get("Content-Type")
    )

@app.route("/login", methods=["GET", "POST"])
def login():
    if request.method == "GET":
        r = requests.get(f"{GO_BACKEND}/login")
        return Response(
            r.content,
            status=r.status_code,
            content_type=r.headers.get("Content-Type")
        )

    if request.method == "POST":
        username = request.form.get("username")
        password = request.form.get("password")

        data = {
            "username": username,
            "password": password
        }
        r = requests.post(f"{GO_BACKEND}/login", data=data)

        if r.status_code != 303 and r.status_code != 200:
            return Response(
                r.content,
                status=r.status_code,
                content_type=r.headers.get("Content-Type")
            )

        session["auth"] = True
        session["username"] = username

        return redirect("/")

@app.route("/logout")
def logout():
    session.clear()
    return redirect("/login")

@app.route("/messages/<room_id>", methods=["GET"])
def messages(room_id):
    r = requests.get(f"{GO_BACKEND}/messages", params={"room": room_id})
    return Response(r.content, status=r.status_code, content_type="application/json")


@app.route("/rooms", methods=["GET"])
def rooms():
    r = requests.get(f"{GO_BACKEND}/rooms")
    return Response(
        r.content,
        status=r.status_code,
        content_type="text/html"
    )

@app.route("/create-room", methods=["POST"])
def create_room():
    if not session.get("auth"):
        return redirect("/login")

    headers = {
        "X-Username": session["username"]
    }

    r = requests.post(f"{GO_BACKEND}/create-room", headers=headers)
    room_id = r.text.strip()
    return redirect(f"/room/{room_id}")

@app.route("/delete-room/<room_id>", methods=["POST"])
def delete_room(room_id):
    if not session.get("auth"):
        return redirect("/login")

    headers = {
        "X-Username": session["username"]
    }

    r = requests.post(
        f"{GO_BACKEND}/delete-room",
        json={"room_id": room_id},
        headers=headers
    )

    if r.status_code != 200:
        return Response(r.text, status=r.status_code)

    return redirect("/rooms")
@app.route("/room/<room_id>")
def room(room_id):
    if not session.get("auth"):
        return redirect("/login")

    return render_template(
        "room.html",
        username=session["username"],
        room_id=room_id
    )

@app.route("/register", methods=["GET", "POST"])
def register():
    if request.method == "GET":
        r = requests.get(f"{GO_BACKEND}/login")
        return Response(
            r.content,
            status=r.status_code,
            content_type=r.headers.get("Content-Type")
        )

    if request.method == "POST":
        username = request.form.get("username")
        password = request.form.get("password")

        data = {
            "username": username,
            "password": password
        }
        r = requests.post(f"{GO_BACKEND}/register", data=data)

        if r.status_code == 200:
            return redirect("/login")
        else:
            return Response(
                r.content,
                status=r.status_code,
                content_type=r.headers.get("Content-Type")
            )
        
@app.route("/profile")
def profile():
    if not session.get("auth"):
        return redirect("/login")

    return render_template("profile.html", username=session["username"])

@app.route("/edit-profile", methods=["POST"])
def edit_profile():
    if not session.get("auth"):
        return redirect("/login")
    newusername = request.form.get("username")
    password = request.form.get("password")

    headers = {"X-Username": session["username"]}
    data = {
            "newusername": newusername,
            "password": password
        }
    r = requests.post(f"{GO_BACKEND}/edit-profile", data=data, headers=headers)

    if r.status_code == 200:
        if newusername != "":
            session["username"] = newusername
        return redirect("/profile")
    else:
        return Response(
            r.content,
            status=r.status_code,
            content_type=r.headers.get("Content-Type")
        )

@app.route("/rooms-stats", methods=["GET"])
def rooms_stats():
    r = requests.get(f"{GO_BACKEND}/rooms-stats")
    return Response(r.content, status=r.status_code, content_type="application/json")


app.run(port=5000, debug=True)
