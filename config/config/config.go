package config

import (
	"Driver-go/elevio"
)

// Constants
const N_FLOORS = 4
const N_ELEVATORS = 3
const N_BUTTONS = 3
const DOOR_OPEN_DURATION = 3.0

const WorldViewPort = 9000
const MasterToElevatorPort = 9001
const ElevatorToMasterPort = 9002

// Elevator roles

type Role int
const (
	SLAVE Role = iota
	BACKUP
	MASTER
)

// Elevator behaviours
type ElevatorBehaviour int
const (
	EB_Idle ElevatorBehaviour = iota
	EB_DoorOpen
	EB_Moving
)

// Elevator structure
type Elevator struct {
	Floor     int
	Dirn      elevio.MotorDirection
	Behaviour ElevatorBehaviour
	Requests  [][]bool 
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