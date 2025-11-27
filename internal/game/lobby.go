package game

import "errors"

var (
	ErrRoomFull = errors.New("la sala esta llena")
	ErrGameStarted = errors.New("el juego ya ha comenzado")
	ErrPlayerExist = errors.New("el jugador ya esta en la sala")
)


// NewRoom crea una instancia de una sala vacia
func NewRoom(id string, config GameConfig) *Room {
	return &Room{
		ID: id,
		Config: config,
		Players: make(map[string]*Player),
		Status: "WAITING",
	}
}

// AddPlayer maneja el aniadir un jugador a una sala
func (r *Room) AddPlayer(p *Player) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock() // se ejecuta al salir de la funcion

	// Se chequea que la partida no haya comenzado
	if r.Status != "WAITING" {
		return ErrGameStarted
	}

	// Se chequea que la sala no este llena
	if len(r.Players) >= r.Config.MaxPlayers {
		return ErrRoomFull
	}

	// Se chequea que el jugador no este en la sala (quizas innecesario)
	if _, exists := r.Players[p.ID]; exists {
		return ErrPlayerExist
	}

	// El primero en unirse sera el admin y si la sala esta vacia es el primero
	if len(r.Players) == 0 {
		p.IsHost = true
	} else {
		p.IsHost = false
	}

	// Se inicializan los dados segun la configuracion de la sala
	p.Dice = make([]Dice, 0, r.Config.DicesAmount)

	r.Players[p.ID] = p
	return nil
}

// RemovePlayer maneja el eliminar a un jugador y reasigna el host si es necesario.
func (r *Room) RemovePlayer(playerID string) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	player, exists := r.Players[playerID]
	if !exists {
		return
	}

	wasHost := player.IsHost
	delete(r.Players, playerID)

	// Si se fue el host y quedan personas en la sala se asigna como host a uno al azar
	if wasHost && len(r.Players) > 0 {
		for _, p := range r.Players {
			p.IsHost = true
			break // solo un host
		}
	}
}


// StartGame cambia el Status de la partida y prepara la primera ronda
func (r *Room) StartGame(playerID string) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	p, exists := r.Players[playerID]
	if !exists || !p.IsHost {
		return errors.New("solo el host puede iniciar la partida")
	}

	if len(r.Players) < 2 {
		return errors.New("no hay suficientes jugadores para comenzar")
	}

	r.Status = "PLAYING"

	r.State = RoundState{
		CurrentBetQuantity: 0,
		CurrentBetFace: 0,
		CurrentPlayerID: playerID, // por ahora el primero sera el host
	}

	// Falta logica de tirar dados de round.go
	r.rollAllDice()

	return nil
}


// rollAllDice es un helper interno (privado) para reiniciar los dados de todos.
func (r *Room) rollAllDice() {
	for _, p := range r.Players {
		// Creamos nuevos dados
		p.Dice = make([]Dice, r.Config.DicesAmount)
		for i := 0; i < r.Config.DicesAmount; i++ {
			// Asignamos valor dummy por ahora, en round.go pondremos el random real
			p.Dice[i] = Dice(1) 
		}
	}
}