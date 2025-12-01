package handlers

import (
	"dados-mentirosos/internal/game"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
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

	// Cuando alguien se conecta
	handler.Melody.HandleConnect(func(s *melody.Session) {
		roomID := s.MustGet("roomID").(string)
		playerID := s.MustGet("playerID").(string)
		playerName := s.MustGet("playerName").(string)

		fmt.Printf("Jugador %s conectado a sala %s\n", playerName, roomID)

		room, err := handler.Manager.GetRoom(roomID)
		if err == nil {
			newPlayer := &game.Player{
				ID:   playerID,
				Name: playerName,
			}
			room.AddPlayer(newPlayer)
			
			handler.BroadcastPlayerList(roomID)
			htmlState := handler.generateLobbyHTML(room, playerID)
			s.Write([]byte(htmlState))
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
	h.broadcastGameState(roomID)
	return nil
}

func (h *WSHandler) broadcastGameState(roomID string) {
	room, err := h.Manager.GetRoom(roomID)
	if err != nil {
		return
	}

	sessions, _ := h.Melody.Sessions()
	for _, s := range sessions {
		sRoomID, exists := s.Get("roomID")
		if !exists || sRoomID.(string) != roomID {
			continue
		}

		playerID, _ := s.Get("playerID")
		var htmlState string
		switch room.Status{
			case "WAITING":
				htmlState = h.generateLobbyHTML(room, playerID.(string))
			case "FINISHED":
				htmlState = h.generateResultsHTML(room, playerID.(string))
			case "PLAYING":
				htmlState = h.generateGameScreenHTML(room, playerID.(string))
		}
		s.Write([]byte(htmlState))
	}
}

// Helper para rellenar la plantilla
func (h *WSHandler) generateGameScreenHTML(room *game.Room, myPlayerID string) string {
	
	me := room.Players[myPlayerID]

	if room.Status == "FINISHED" && room.LastResult != nil {
		return h.generateResultsHTML(room, myPlayerID)
	}


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

func (h *WSHandler) generateResultsHTML(room *game.Room, myPlayerID string) string {
    // Debug: Avisar que intentamos generar resultados
    fmt.Printf("🎨 Generando pantalla de resultados para %s...\n", myPlayerID)

    playersList := make([]*game.Player, 0)
    for _, p := range room.Players {
        playersList = append(playersList, p)
    }

    funcMap := template.FuncMap{
        "toInt": func(i interface{}) int {
            switch v := i.(type) {
            case game.Dice:
                return int(v)
            case int:
                return v
            default:
                return 0
            }
        },
    }
    data := map[string]interface{}{
        "RoomID":  room.ID,
        "MyID":    myPlayerID,
        "Result":  room.LastResult,
        "Players": playersList,
        "IsHost":  room.Players[myPlayerID].IsHost,
    }

    // Asegurarse de que la ruta es correcta
    files := []string{"ui/html/partials/game/results.html"}
    
    // Parsear
    tmpl, err := template.New("results_screen").Funcs(funcMap).ParseFiles(files...)
    if err != nil {
        // ERROR CRÍTICO 1: No se encontró el archivo o falló el parseo
        fmt.Printf("❌ ERROR ParseFiles Results: %v\n", err)
        return fmt.Sprintf(`<div id="content" hx-swap-oob="innerHTML" class="bg-red-900 p-4 text-white">ERROR TEMPLATE: %v</div>`, err)
    }

    var out strings.Builder
    err = tmpl.ExecuteTemplate(&out, "results_screen", data)
    if err != nil {
        // ERROR CRÍTICO 2: Falló al ejecutar (variable faltante, función mal llamada)
        fmt.Printf("❌ ERROR ExecuteTemplate Results: %v\n", err)
        return fmt.Sprintf(`<div id="content" hx-swap-oob="innerHTML" class="bg-red-900 p-4 text-white">ERROR EXEC: %v</div>`, err)
    }

    fmt.Println("✅ HTML Resultados generado correctamente")
    return fmt.Sprintf(`<div id="content" hx-swap-oob="innerHTML">%s</div>`, out.String())
}

func (h *WSHandler) generateLobbyHTML(room *game.Room, playerID string) string {
	// Reutilizamos el archivo lobby.html que ya creamos
	files := []string{"ui/html/pages/lobby.html"}
	
	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		return fmt.Sprintf("Error template lobby: %v", err)
	}

	isHost := false
	if p, ok := room.Players[playerID]; ok {
		isHost = p.IsHost
	}

	data := map[string]interface{}{
		"RoomID": room.ID,
		"Config": room.Config,
		"IsHost": isHost,
	}

	var out strings.Builder
	err = tmpl.ExecuteTemplate(&out, "content", data)
	if err != nil {
		return fmt.Sprintf("Error exec lobby: %v", err)
	}
	return fmt.Sprintf(`<div id="content" hx-swap-oob="innerHTML">%s</div>`, out.String())
}

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

func (h *WSHandler) HandleBet(w http.ResponseWriter, r *http.Request) {
	// Identificar al jugador
	cookie, _ := r.Cookie("player_id")
	parts := strings.Split(cookie.Value, ":")
	plaryerID := parts[0]

	roomID := r.URL.Query().Get("roomID")

	room, err := h.Manager.GetRoom(roomID)
	if err != nil {
		http.Error(w, "sala no encontrada", http.StatusNotFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "datos invalidos", http.StatusBadRequest)
		return
	}

	quantity := atoi(r.FormValue("quantity"))
	face := atoi(r.FormValue("face"))

	err = room.PlaceBet(plaryerID, quantity, face)
	if err != nil {
		w.Header().Set("HX-Retarget", "#bet-error")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.broadcastGameState(roomID)
	w.WriteHeader(http.StatusOK)
}

func (h *WSHandler) HandleRestart(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("player_id")
	parts := strings.Split(cookie.Value, ":")
	playerID := parts[0]
	roomID := r.URL.Query().Get("roomID")

	room, err := h.Manager.GetRoom(roomID)
	if err != nil {
		http.Error(w, "Sala no encontrada", http.StatusNotFound)
		return
	}

	// Validar que sea Host
	isHost := false
	if p, ok := room.Players[playerID]; ok && p.IsHost {
		isHost = true
	}
	if !isHost {
		http.Error(w, "Solo el host puede reiniciar", http.StatusForbidden)
		return
	}
	room.Reset()
	h.broadcastGameState(roomID)
	h.BroadcastPlayerList(roomID)
	w.WriteHeader(http.StatusOK)
}

// simplificar pasar de string a int
func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func (h *WSHandler) HandleLiar(w http.ResponseWriter, r *http.Request) {
	// Identificar jugador y sala
	cookie, _ := r.Cookie("player_id")
	parts := strings.Split(cookie.Value, ":")
	playerID := parts[0]
	roomID := r.URL.Query().Get("roomID")

	room, err := h.Manager.GetRoom(roomID)
	if err != nil {
		http.Error(w, "Sala no encontrada", http.StatusNotFound)
		return
	}

	_, err = room.CallLiar(playerID)
	if err != nil {
		fmt.Printf("Error CallLiar: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.broadcastGameState(roomID)
	w.WriteHeader(http.StatusOK)
}

func (h *WSHandler) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	// Identificar Host y Sala
	cookie, _ := r.Cookie("player_id")
	parts := strings.Split(cookie.Value, ":")
	playerID := parts[0]
	roomID := r.URL.Query().Get("roomID")

	room, err := h.Manager.GetRoom(roomID)
	if err != nil {
		http.Error(w, "Sala no encontrada", http.StatusNotFound)
		return
	}

	// Verificar Permisos
	isHost := false
	if p, ok := room.Players[playerID]; ok && p.IsHost {
		isHost = true
	}
	if !isHost {
		http.Error(w, "Solo el host puede configurar", http.StatusForbidden)
		return
	}

	// Procesar Formulario
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Datos inválidos", http.StatusBadRequest)
		return
	}

	// Actualizar Configuración (Protegido por Mutex)
	room.Mutex.Lock()
	room.Config.DicesAmount = atoi(r.FormValue("initial_dice_count"))
	room.Config.TurnDuration = atoi(r.FormValue("turn_duration"))
	room.Config.MinBetIncrement = atoi(r.FormValue("min_bet_increment"))
	room.Config.WildAces = (r.FormValue("wild_aces") == "on")
	
	// Validaciones de seguridad
	if room.Config.MaxPlayers < 2 { room.Config.MaxPlayers = 2 }
	if room.Config.DicesAmount < 1 { room.Config.DicesAmount = 5 }
	if room.Config.MinBetIncrement < 1 { room.Config.MinBetIncrement = 1 }
	
	room.Mutex.Unlock()
	h.broadcastGameState(roomID)
	w.WriteHeader(http.StatusOK)
}