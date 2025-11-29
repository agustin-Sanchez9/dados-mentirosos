package game

import (
	"errors"
	"math/rand"
	"time"
)

var (
	ErrNotYourTurn  = errors.New("no es tu turno")
	ErrInvalidBet   = errors.New("la apuesta debe ser mayor a la actual")
	ErrNoBetMade    = errors.New("no hay apuesta previa para llamar mentiroso")
)

// ---Logica de Dados---

// rollDice genera nuevos numeros para un jugador.
func (r *Room) rollDice(p *Player) {
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	count := r.Config.DicesAmount
	p.Dice = make([]Dice, count)
	
	for i := 0; i < count; i++ {
		p.Dice[i] = Dice(rng.Intn(6) + 1)
	}
}

// ---Logica de Apuestas---

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
	
	return nil
}

func (r *Room) isValidBet(qty int, face int) bool {
	if r.State.CurrentBetQuantity == 0 {
		return qty > 0 && face >= 1 && face <= 6
	}
	if qty >= (r.State.CurrentBetQuantity + r.Config.MinBetIncrement) {
		return true
	}
	return false
}

// ---Logica de "Mentiroso"---

// GameResult contiene los datos finales para mostrar en la pantalla de resultados
type GameResult struct {
	AccuserID    string // El que dijo "Mentiroso"
	BlufferID    string // El que hizo la apuesta (el acusado)
	BetQuantity  int
	BetFace      int
	RealCount    int
	IsLiar       bool   // True = Bluffer pierde, False = Accuser pierde
	WinnerID     string
	LoserID      string
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

	// 1. Calcular la realidad
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

	// 2. Determinar ganador/perdedor
	betQty := r.State.CurrentBetQuantity
	blufferID := r.State.LastBetPlayerID
	
	isLiar := realCount < betQty 

	var winner, loser string

	if isLiar {
		// El acusado (Bluffer) pierde. El acusador gana.
		loser = blufferID
		winner = accuserPlayerID
	} else {
		// El acusado se salva. El acusador (que desconfió mal) pierde.
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