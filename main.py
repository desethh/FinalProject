from flask import Flask, request, session, Response, redirect
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

app.run(port=5000, debug=True)
