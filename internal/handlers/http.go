package handlers

import (
	"dados-mentirosos/internal/game"
	"fmt"
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


// render carga el layout, la pagina solicitada Y los partials necesarios explicitamente
func (h *GameHandler) render(w http.ResponseWriter, page string, data any) {
	files := []string{
		"ui/html/base.html",       // El esqueleto
		"ui/html/pages/" + page,   // La pagina (home.html o lobby.html)
	}

	if page == "lobby.html" {
		files = append(files, "ui/html/partials/lobby/settings.html")
	}

	tmpl := template.New("base")
	
	// Funciones auxiliares
	tmpl.Funcs(template.FuncMap{
		"toInt": func(i any) int { return 0 }, 
	})

	// Parsear la lista exacta de archivos
	var err error
	tmpl, err = tmpl.ParseFiles(files...)
	if err != nil {
		// Logueamos el error en la terminal para que sepas que archivo falta
		fmt.Println("❌ Error ParseFiles:", err) 
		http.Error(w, "Error cargando archivos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Ejecutar
	err = tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		fmt.Println("❌ Error ExecuteTemplate:", err)
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

// Room sirve la pantalla de espera
func (h *GameHandler) Room(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")

	room, err := h.Manager.GetRoom(roomID)
	if err != nil {
		http.Redirect(w, r, "/?error=notfound", http.StatusSeeOther)
		return
	}

	// Detectar host leyendo la cookie
	isHost := false
	cookie, err := r.Cookie("player_id")
	if err == nil {
		parts := strings.Split(cookie.Value, ":")
		playerID := parts[0]

		if p, ok := room.Players[playerID]; ok && p.IsHost {
			isHost = true
		} else if len(room.Players) == 0 {
			// Caso especial: Si la sala esta vacia, el primero que entra sera el host
			isHost = true
		}
	}
	data := map[string]interface{}{
		"RoomID": room.ID,
		"Config": room.Config,
		"IsHost": isHost,
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