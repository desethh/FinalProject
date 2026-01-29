from flask import Flask, request, session, Response, redirect, render_template
import requests

app = Flask(__name__)
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
        # Просто проксируем страницу логина от Go
        r = requests.get(f"{GO_BACKEND}/login")
        return Response(
            r.content,
            status=r.status_code,
            content_type=r.headers.get("Content-Type")
        )

    if request.method == "POST":
        # Получаем данные из формы
        username = request.form.get("username")

        # Сохраняем сессию в Flask
        session["auth"] = True
        session["username"] = username

        # После успешного логина редирект на страницу Go
        return redirect("/")

@app.route("/logout")
def logout():
    session.clear()
    return redirect("/login")

@app.route("/create-room", methods=["POST"])
def create_room():
    if not session.get("auth"):
        return redirect("/login")

    headers = {
        "X-Username": session["username"]
    }

    r = requests.post(f"{GO_BACKEND}/create-room", headers=headers)
    room_id = r.text

    return redirect(f"/room/{room_id}")

@app.route("/room/<room_id>")
def room(room_id):
    if not session.get("auth"):
        return redirect("/login")

    return render_template(
        "room.html",
        username=session["username"],
        room_id=room_id
    )


app.run(port=5000, debug=True)
