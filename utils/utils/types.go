package utils

import (
	"Driver-go/elevio"
)

// Elevator roles
type Role int

// Elevator behaviours
type ElevatorBehaviour int


// Elevator structure
type Elevator struct {
	Floor     int
	Dirn      elevio.MotorDirection
	Behaviour ElevatorBehaviour
	Requests  [][]bool 
	Obstructed bool
	MotorStopped bool
}

// World view structure
type WorldView struct {
	Elevators [N_ELEVATORS]Elevator
	ElevatorID int
	Role Role
	Alive [N_ELEVATORS]bool
}

// Button message structure
type ButtonMessage struct {
	ButtonEvent elevio.ButtonEvent
	ElevatorID int
}