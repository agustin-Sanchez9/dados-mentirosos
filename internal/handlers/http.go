package handlers

import (
	"dados-mentirosos/internal/game"
	"html/template"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type GameHandler struct {
	Manager   *game.GameManager
}

func NewGameHandler(manager *game.GameManager) *GameHandler {
	return &GameHandler{
		Manager:   manager,
	}
}

// Home sirve la pantalla principal
func (h *GameHandler) Home(w http.ResponseWriter, r *http.Request) {
	h.render(w, "home.html", nil)
}


// render es una función auxiliar que combina el layout base con la página específica
func (h *GameHandler) render(w http.ResponseWriter, page string, data any) {
	// Definimos los archivos necesarios: Base + Página específica
	files := []string{
		"ui/html/base.html",        // El esqueleto
		"ui/html/pages/" + page,    // El contenido (ej: home.html)
	}

	// Parseamos los archivos
	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		http.Error(w, "Error cargando template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// (Opcional) Parsear partials si los tuviéramos
	// tmpl.ParseGlob("ui/html/partials/**/*.html")

	err = tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		http.Error(w, "Error renderizando: "+err.Error(), http.StatusInternalServerError)
	}
}

// CreateRoom procesa el formulario y redirige
func (h *GameHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	// Se leen datos del formulario
	r.ParseForm()
	playerName := r.FormValue("player_name")
	
	// Por ahora usamos la default para probar el flujo
	config := game.GameConfig{
		MaxPlayers: 7,
		DicesAmount: 5,
		MinBetIncrement: 1,
		WildAces: true,
	}

	// Se genera ID unico para la sala (o usar uno corto de 4 letras)
	roomID := strings.Split(uuid.New().String(), "-")[0] // Solo la primera parte para que sea corto

	// Se crea la sala en memoria
	h.Manager.CreateRoom(roomID, config)

	// Se crea cookie de secion para saber quien es este usuario (simplificado)
	playerID := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:  "player_id",
		Value: playerID + ":" + playerName, // Hack sucio para guardar nombre, idealmente en servidor
		Path:  "/",
	})

	// Se redirige a la sala
	http.Redirect(w, r, "/room/"+roomID, http.StatusSeeOther)
}

// Room sirve la pantalla de espera (Lobby)
func (h *GameHandler) Room(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")

	// Se verifica si la sala existe
	room, err := h.Manager.GetRoom(roomID)
	if err != nil {
		http.Error(w, "Sala no encontrada", http.StatusNotFound)
		return
	}

	// Se renderiza la vista de juego/lobby, se pasa el RoomID para que el HTML sepa a que websocket conectarse
	data := map[string]interface{}{
		"RoomID": room.ID,
	}
	h.render(w, "lobby.html", data)
}

// JoinRoom procesa la solicitud de unirse a una sala existente
func (h *GameHandler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	// Parsear datos
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error en formulario", http.StatusBadRequest)
		return
	}

	playerName := r.FormValue("player_name")
	roomID := r.FormValue("room_id")

	// Validar que la sala exista Y que se pueda entrar
	room, err := h.Manager.GetRoom(roomID)
	if err != nil {
		// Si no existe volvemos al home con error
		http.Redirect(w, r, "/?error=notfound", http.StatusSeeOther)
		return
	}

	// Validar que la sala no este llena
	if len(room.Players) >= room.Config.MaxPlayers {
		http.Redirect(w, r, "/?error=full", http.StatusSeeOther)
		return
	}

	// Validar que la partida no haya empezado
	if room.Status != "WAITING" {
		http.Redirect(w, r, "/?error=started", http.StatusSeeOther)
		return
	}

	// Crear Cookie de Sesión
	playerID := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:  "player_id",
		Value: playerID + ":" + playerName,
		Path:  "/",
	})

	// Redirigir al Lobby
	http.Redirect(w, r, "/room/"+roomID, http.StatusSeeOther)
}