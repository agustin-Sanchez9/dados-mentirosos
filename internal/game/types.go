package game

import (
	"math/rand"
	"sync"
	"time"
)

// Representara un dado (1-6)
type Dice int

// Representacion de un jugador
type Player struct {
	ID string
	Name string
	Dice []Dice
	IsHost bool
}

// Configuraciones de la sala
type GameConfig struct {
	DicesAmount int
	MaxPlayers int
	TurnDuration int
	MinBetIncrement int
	WildAces bool
}

// Estado actual de la ronda
type RoundState struct {
	CurrentPlayerID string // de quien es el turno
	LastBetPlayerID string // ultimo turno
	CurrentBetQuantity int // cantidad de la apuesta
	CurrentBetFace int // cada de la apuesta
}

type UpdateCallback func(roomID string)

// estructura de la sala
type Room struct {
	ID string
	Mutex sync.RWMutex
	Players map[string]*Player // lista de jugadores
	PlayerOrder []string // lista para saber el orden de la mesa
	Config GameConfig
	State RoundState
	Status string // "WAITING", "PLAYING", "FINISHED"
	rng *rand.Rand
	LastResult *GameResult
	TurnTimer *time.Timer // reloj interno
	TurnDeadline time.Time // hora exacta
	OnUpdate UpdateCallback // funcion para actualizar pantallas
}

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