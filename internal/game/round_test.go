package game

import (
	"testing"
)

func TestGameFlow_SuddenDeath(t *testing.T) {
	cfg := GameConfig{MaxPlayers: 2, DicesAmount: 2, WildAces: true}
	room := NewRoom("GAME-SD", cfg)

	p1 := &Player{ID: "p1", Name: "Alice"}
	p2 := &Player{ID: "p2", Name: "Bob"}
	room.AddPlayer(p1)
	room.AddPlayer(p2)
	room.StartGame("p1")

	// Mock de dados: Alice[5,5], Bob[1,4] (1 es comodín) -> Total cincos = 3
	p1.Dice = []Dice{5, 5}
	p2.Dice = []Dice{1, 4}

	// Alice: "Un 5"
	room.PlaceBet("p1", 1, 5)
	// Bob: "Tres 5s"
	room.PlaceBet("p2", 3, 5)
	
	// Alice: "Mentiroso!"
	result, err := room.CallLiar("p1")
	if err != nil {
		t.Fatalf("Error llamando mentiroso: %v", err)
	}

	// Verificaciones
	if result.IsLiar == true {
		t.Errorf("Bob decía la verdad (había 3 cincos), IsLiar debería ser false")
	}
	if result.RealCount != 3 {
		t.Errorf("El conteo real fue %d, se esperaba 3", result.RealCount)
	}
	if result.LoserID != "p1" {
		t.Errorf("Alice desconfió de una verdad, ella debió perder. Perdió: %s", result.LoserID)
	}
	if room.Status != "FINISHED" {
		t.Error("El juego debería haber terminado")
	}
}