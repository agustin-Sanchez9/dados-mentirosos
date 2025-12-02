package main

import (
	"dados-mentirosos/internal/game"
	"dados-mentirosos/internal/handlers"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/olahol/melody"
)

func main() {
	// Se inicializan Dependencias
	gm := game.NewGameManager()
	m := melody.New()
	gameHandler := handlers.NewGameHandler(gm)
	wsHandler := handlers.NewWSHandler(m, gm, gameHandler)

	// Se configura el router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Servir archivos estáticos (CSS/JS) si los tuvieras locales
	// r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("ui/static"))))

	// Rutas HTTP
	r.Get("/", gameHandler.Home)
	r.Post("/create-room", gameHandler.CreateRoom)
	r.Post("/join-room", gameHandler.JoinRoom)
	r.Get("/room/{roomID}", gameHandler.Room)
	r.Post("/game/start", wsHandler.HandleStartGame)
	r.Post("/game/bet", wsHandler.HandleBet)
	r.Post("/game/liar", wsHandler.HandleLiar)
	r.Post("/game/restart", wsHandler.HandleRestart)
	r.Post("/game/config", wsHandler.HandleUpdateConfig)

	// Rutas WS
	r.Get("/ws/{roomID}", wsHandler.HandleRequest)

	port := os.Getenv("PORT")
	if port == ""{
		port = "3000"
	}

	// Iniciar Servidor
	fmt.Println("Servidor corriendo en http://localhost:3000")
	http.ListenAndServe(":"+port, r)
}