package handlers

import (
	"dados-mentirosos/internal/game"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/olahol/melody"
)

type WSHandler struct {
	Manager   *game.GameManager
	Melody    *melody.Melody
	GameH     *GameHandler
}

func NewWSHandler(m *melody.Melody, gm *game.GameManager, gh *GameHandler) *WSHandler {
	handler := &WSHandler{
		Manager: gm, 
		Melody:  m,
		GameH:   gh,
	}

	// EVENTOS DEL WEBSOCKET

	// Cuando alguien se conecta
	handler.Melody.HandleConnect(func(s *melody.Session) {
		// Leemos el RoomID y PlayerID que guardamos en la sesion
		roomID := s.MustGet("roomID").(string)
		playerID := s.MustGet("playerID").(string)
		playerName := s.MustGet("playerName").(string)

		fmt.Printf("Jugador %s conectado a sala %s\n", playerName, roomID)

		// Se agrega el jugador a la lógica del juego (Room)
		room, err := handler.Manager.GetRoom(roomID)
		if err == nil {

			newPlayer := &game.Player{
				ID:   playerID,
				Name: playerName,
			}
			room.AddPlayer(newPlayer)
			
			// Se avisa a TODOS en la sala que actualicen la lista visual
			handler.BroadcastPlayerList(roomID)
		}
	})

	// Cuando alguien se desconecta
	handler.Melody.HandleDisconnect(func(s *melody.Session) {
		roomID := s.MustGet("roomID").(string)
		playerID := s.MustGet("playerID").(string)

		room, err := handler.Manager.GetRoom(roomID)
		if err == nil {
			room.RemovePlayer(playerID)
			handler.BroadcastPlayerList(roomID)
		}
	})

	return handler
}

// HandleRequest es el endpoint HTTP que transforma la conexion en WebSocket
func (h *WSHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	
	// Leer la cookie para saber quién es
	cookie, err := r.Cookie("player_id")
	if err != nil {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}
	
	// Parsear "ID:Nombre" (que se guardo en http.go)
	parts := strings.Split(cookie.Value, ":")
	if len(parts) != 2 {
		http.Error(w, "Cookie inválida", http.StatusBadRequest)
		return
	}
	playerID := parts[0]
	playerName := parts[1]

	// Pasamos datos a la sesión de Melody para usarlos en HandleConnect
	keys := map[string]interface{}{
		"roomID":     roomID,
		"playerID":   playerID,
		"playerName": playerName,
	}

	h.Melody.HandleRequestWithKeys(w, r, keys)
}

// BroadcastPlayerList genera el HTML de la lista y lo envía a todos en la sala
func (h *WSHandler) BroadcastPlayerList(roomID string) {
	room, _ := h.Manager.GetRoom(roomID)
	
	// Usamos un buffer para renderizar el HTML a texto
	
	var htmlBuilder strings.Builder

	// Le dice a HTMX: "Busca el elemento con id 'players-list' y reemplazalo"
	htmlBuilder.WriteString(`<ul id="players-list" hx-swap-oob="true" class="space-y-2">`)
	
	for _, p := range room.Players {
		hostBadge := ""
		if p.IsHost {
			hostBadge = "👑"
		}
		htmlBuilder.WriteString(fmt.Sprintf(
			`<li class="bg-slate-700 p-2 rounded flex justify-between">
				<span>%s %s</span>
			</li>`, 
			p.Name, hostBadge))
	}
	htmlBuilder.WriteString("</ul>")

	// Enviar solo a los clientes de esta sala
	h.Melody.BroadcastFilter([]byte(htmlBuilder.String()), func(q *melody.Session) bool {
		return q.MustGet("roomID").(string) == roomID
	})
}

// StartGameAndBroadcast inicia el juego y notifica a todos con sus tableros únicos
func (h *WSHandler) StartGameAndBroadcast(roomID string) error {
	room, err := h.Manager.GetRoom(roomID)
	if err != nil {
		return err
	}
	// Buscamos el ID del Host para iniciar la partida
	var hostID string
	for _, p := range room.Players {
		if p.IsHost {
			hostID = p.ID
			break
		}
	}
	err = room.StartGame(hostID)
	if err != nil {
		return err
	}

	sessions, _ := h.Melody.Sessions()

	for _, s := range sessions {
		sRoomID, exists := s.Get("roomID")
		if !exists || sRoomID.(string) != roomID {
			continue
		}

		playerID, _ := s.Get("playerID")

		// 4. Generar el estado visual para cada jugador
		htmlState := h.generateGameScreenHTML(room, playerID.(string))

		// 5. Enviar HTML por WS individualmente
		s.Write([]byte(htmlState))
	}

	return nil
}

// Helper para rellenar la plantilla
func (h *WSHandler) generateGameScreenHTML(room *game.Room, myPlayerID string) string {
	// 1. Preparar datos
	me := room.Players[myPlayerID]

	currentPlayerName := "???"
	if p, ok := room.Players[room.State.CurrentPlayerID]; ok {
		currentPlayerName = p.Name
	}

	type OpponentView struct {
		Name      string
		DiceCount int
		IsTurn    bool
	}
	var opponents []OpponentView
	for _, p := range room.Players {
		if p.ID != myPlayerID {
			opponents = append(opponents, OpponentView{
				Name:      p.Name,
				DiceCount: len(p.Dice),
				IsTurn:    (p.ID == room.State.CurrentPlayerID),
			})
		}
	}

	lastBetPlayerName := "Nadie"
	if p, ok := room.Players[room.State.LastBetPlayerID]; ok {
		lastBetPlayerName = p.Name
	}

	data := map[string]interface{}{
		"RoomID":            room.ID,
		"IsMyTurn":          (room.State.CurrentPlayerID == myPlayerID),
		"CurrentPlayerName": currentPlayerName,
		"CurrentBetQty":     room.State.CurrentBetQuantity,
		"CurrentBetFace":    room.State.CurrentBetFace,
		"LastBetPlayer":     lastBetPlayerName,
		"MyDice":            me.Dice,
		"Opponents":         opponents,
	}

	// Cargar los templates necesarios aquí mismo
	files := []string{
		"ui/html/partials/game/screen.html",   // El tablero
		"ui/html/partials/game/controls.html", // Los botones
	}

	// Usamos "html/template"
	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		fmt.Println("Error cargando templates WS:", err)
		return `<div class="text-red-500">Error interno cargando el juego</div>`
	}

	// 3. Renderizar a String
	var out strings.Builder
	err = tmpl.ExecuteTemplate(&out, "game_screen", data)
	if err != nil {
		fmt.Println("Error renderizando game_screen:", err)
		return `<div class="text-red-500">Error renderizando el juego</div>`
	}

	// 4. Envolver en OOB swap
	return fmt.Sprintf(`<div id="content" hx-swap-oob="innerHTML">%s</div>`, out.String())
}

// En internal/handlers/ws.go

func (h *WSHandler) HandleStartGame(w http.ResponseWriter, r *http.Request) {
    // Obtener datos de cookie
    cookie, _ := r.Cookie("player_id")
    parts := strings.Split(cookie.Value, ":")
    playerID := parts[0]
    
    // Obtener RoomID 
    roomID := r.URL.Query().Get("roomID")

    // Validar host
    room, err := h.Manager.GetRoom(roomID)
    if err != nil {
        http.Error(w, "Sala no encontrada", http.StatusNotFound)
        return
    }
    
    // Verificacion rapida de host
    isHost := false
    for _, p := range room.Players {
        if p.ID == playerID && p.IsHost { isHost = true; break }
    }
    if !isHost {
        http.Error(w, "Solo el host puede iniciar", http.StatusForbidden)
        return
    }

    // 4. Iniciar y Broadcast
    err = h.StartGameAndBroadcast(roomID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}