package game

import (
	"errors"
	"sync"
)

// GameManager gestionara todas las salas activas del servidor
type GameManager struct {
	mutex sync.RWMutex
	rooms map[string]*Room
}

// NewGameManager inicializa un GameManager
func NewGameManager() *GameManager {
	return &GameManager{
		rooms: make(map[string]*Room),
	}
}

// CreateRoom crea una sala y la agrega al manager
func (gm *GameManager) CreateRoom(id string, config GameConfig) *Room {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	newRoom := NewRoom(id, config)
	gm.rooms[id]=newRoom
	return newRoom
}

// GetRoom busca una sala por ID
func (gm *GameManager) GetRoom(id string) (*Room, error) {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	room, exists := gm.rooms[id]
	if !exists {
		return nil, errors.New("sala no encontrada")
	}
	return room, nil
}