package main

import (
	"dados-mentirosos/internal/game"
	"dados-mentirosos/internal/handlers"
	"fmt"
	"net/http"

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

	// Rutas WS
	r.Get("/ws/{roomID}", wsHandler.HandleRequest)

	// Iniciar Servidor
	fmt.Println("Servidor corriendo en http://localhost:3000")
	http.ListenAndServe(":3000", r)
}