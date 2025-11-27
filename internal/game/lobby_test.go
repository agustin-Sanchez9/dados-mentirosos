package game

import (
	"testing"
)

func TestAddPlayer(t *testing.T) {
	// 1. Configuración básica
	cfg := GameConfig{MaxPlayers: 2, DicesAmount: 5}
	room := NewRoom("TEST-01", cfg)

	// 2. Añadir Host
	p1 := &Player{ID: "p1", Name: "Host"}
	err := room.AddPlayer(p1)
	if err != nil {
		t.Fatalf("Error añadiendo p1: %v", err)
	}
	if !p1.IsHost {
		t.Error("El primer jugador debería ser Host")
	}

	// 3. Añadir Invitado
	p2 := &Player{ID: "p2", Name: "Guest"}
	err = room.AddPlayer(p2)
	if err != nil {
		t.Fatalf("Error añadiendo p2: %v", err)
	}
	if p2.IsHost {
		t.Error("El segundo jugador NO debería ser Host")
	}

	// 4. Intentar añadir tercero (debería fallar por MaxPlayers = 2)
	p3 := &Player{ID: "p3", Name: "Intruder"}
	err = room.AddPlayer(p3)
	if err != ErrRoomFull {
		t.Errorf("Esperaba error 'Sala Llena', obtuve: %v", err)
	}
}

func TestStartGame(t *testing.T) {
	cfg := GameConfig{MaxPlayers: 4, DicesAmount: 5}
	room := NewRoom("TEST-02", cfg)

	p1 := &Player{ID: "p1", Name: "Host"}
	p2 := &Player{ID: "p2", Name: "Guest"}
	room.AddPlayer(p1)
	room.AddPlayer(p2)

	// Intentar iniciar con un no-host
	err := room.StartGame("p2")
	if err == nil {
		t.Error("El invitado no debería poder iniciar el juego")
	}

	// Iniciar con el host
	err = room.StartGame("p1")
	if err != nil {
		t.Fatalf("Error iniciando juego: %v", err)
	}

	if room.Status != "PLAYING" {
		t.Error("El estado debería ser PLAYING")
	}
	
	// Verificar que se 'repartieron' los dados (según nuestro mock en rollAllDice)
	if len(room.Players["p1"].Dice) != 5 {
		t.Errorf("El jugador debería tener 5 dados, tiene %d", len(room.Players["p1"].Dice))
	}
}