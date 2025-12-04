package game

import (
	"errors"
	"time"
)

var (
	ErrNotYourTurn  = errors.New("no es tu turno")
	ErrInvalidBet   = errors.New("la apuesta debe ser mayor a la actual")
	ErrNoBetMade    = errors.New("no hay apuesta previa para llamar mentiroso")
)

// rollDice genera nuevos numeros para un jugador.
func (r *Room) rollDice(p *Player) {

	count := r.Config.DicesAmount
	p.Dice = make([]Dice, count)
	
	for i := 0; i < count; i++ {
		p.Dice[i] = Dice(r.rng.Intn(6) + 1) // Intn(6) da 0 a 5, por eso el +1
	}
}

// PlaceBet maneja la logica de realizar apuestas
func (r *Room) PlaceBet(playerID string, quantity int, face int) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if r.Status != "PLAYING" {
		return errors.New("la partida no está en curso")
	}

	if r.State.CurrentPlayerID != playerID {
		return ErrNotYourTurn
	}

	if !r.isValidBet(quantity, face) {
		return ErrInvalidBet
	}

	// Actualizar estado de la apuesta
	r.State.CurrentBetQuantity = quantity
	r.State.CurrentBetFace = face
	r.State.LastBetPlayerID = playerID
	
	r.nextTurn()
	r.resetTurnTimer()
	
	return nil
}

// isValidBet realiza el chequeo de si la apuesta es valida
func (r *Room) isValidBet(qty int, face int) bool {
	if r.State.CurrentBetQuantity == 0 {
		return qty > 0 && face >= 1 && face <= 6
	}
	if qty >= (r.State.CurrentBetQuantity + r.Config.MinBetIncrement) {
		return true
	}
	return false
}

// CallLiar termina el juego inmediatamente y retorna el resultado.
func (r *Room) CallLiar(accuserPlayerID string) (*GameResult, error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if r.State.CurrentPlayerID != accuserPlayerID {
		return nil, ErrNotYourTurn
	}
	if r.State.CurrentBetQuantity == 0 {
		return nil, ErrNoBetMade
	}

	r.stopTurnTimer() // el jeugo termina asi que paramos el timer

	// Calcular la realidad
	targetFace := r.State.CurrentBetFace
	realCount := 0
	for _, p := range r.Players {
		for _, d := range p.Dice {
			if int(d) == targetFace {
				realCount++
			} else if r.Config.WildAces && int(d) == 1 && targetFace != 1 {
				realCount++
			}
		}
	}

	// Determinar ganador y perdedor
	betQty := r.State.CurrentBetQuantity
	blufferID := r.State.LastBetPlayerID
	
	isLiar := realCount < betQty 

	var winner, loser string

	if isLiar {
		// El acusado (Bluffer) pierde. El acusador gana.
		loser = blufferID
		winner = accuserPlayerID
	} else {
		// El acusado se salva. El acusador pierde.
		loser = accuserPlayerID
		winner = blufferID
	}

	// 3. Finalizar el juego
	r.Status = "FINISHED"

	result := &GameResult{
		AccuserID:   accuserPlayerID,
		BlufferID:   blufferID,
		BetQuantity: betQty,
		BetFace:     targetFace,
		RealCount:   realCount,
		IsLiar:      isLiar,
		WinnerID:    winner,
		LoserID:     loser,
	}

	r.LastResult = result

	return result, nil
}

// nextTurn pasa al siguiente en la lista de forma circular
func (r *Room) nextTurn() {
	if len(r.PlayerOrder) == 0 {
		return // evitar el panic "dividir por cero", manejar error despues?
	}

	currentIdx := -1
	for i, id := range r.PlayerOrder {
		if id == r.State.CurrentPlayerID {
			currentIdx = i
			break
		}
	}

	// si no encuentra al actual que empiece del principio, para no usar el -1
	if currentIdx == -1 {
		r.State.CurrentPlayerID = r.PlayerOrder[0]
		return
	}

	nextIdx := (currentIdx + 1) % len(r.PlayerOrder)
	r.State.CurrentPlayerID = r.PlayerOrder[nextIdx]
}

// resetTurnTimer inicia el timer para el jugador actual
func (r *Room) resetTurnTimer() {
	r.stopTurnTimer() // detener el timer si existe

	if r.Config.TurnDuration <= 0 { // duracion 0 representa infinito
		return
	}

	duration := time.Duration(r.Config.TurnDuration) * time.Second
	r.TurnDeadline = time.Now().Add(duration)

	r.TurnTimer = time.AfterFunc(duration, func() {
		r.handleTimeout()
	})
}

// stopTurnTimer detiene el reloj
func (r *Room) stopTurnTimer() {
	if r.TurnTimer != nil {
		r.TurnTimer.Stop()
		r.TurnTimer = nil
	}
	r.TurnDeadline = time.Time{} // resetear fecha
}

// handleTimeout se ejecuta cuando se termina el tiempo
func (r *Room) handleTimeout() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if r.Status != "PLAYING" {
		return
	}

	currentPlayer := r.State.CurrentPlayerID

	quantity := r.State.CurrentBetQuantity + r.Config.MinBetIncrement
	face := r.State.CurrentBetFace

	if r.State.CurrentBetQuantity == 0 {
		quantity = r.Config.MinBetIncrement
		face = 2
	}

	r.State.CurrentBetFace = face
	r.State.CurrentBetQuantity = quantity
	r.State.LastBetPlayerID = currentPlayer

	r.nextTurn()
	r.resetTurnTimer()

	if r.OnUpdate != nil {
		go r.OnUpdate(r.ID) // goroutine aparte para no bloquear el mutex
	}
}