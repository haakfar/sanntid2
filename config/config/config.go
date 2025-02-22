package config

import (
	"Driver-go/elevio"
)

const (
	STAND_STILL Direction = iota
	GOING_UP
	GOING_DOWN
)

const (
	SLAVE Role = iota
	BACKUP
	MASTER
)

const (
	EB_Idle ElevatorBehaviour = iota
	EB_DoorOpen
	EB_Moving
)

const (
	RECEIVED MessageType = iota
	SENT 
)

const N_FLOORS = 4
const N_ELEVATORS = 3
const N_BUTTONS = 3
const DOOR_OPEN_DURATION = 3.0

const Port = 9000

type Direction int
type Role int
type MessageType int

type ElevatorBehaviour int

type Elevator struct {
	Floor     int
	Dirn      elevio.MotorDirection
	Behaviour ElevatorBehaviour
	Requests  [][]bool 
}

type WorldView struct {
	Elevators [N_ELEVATORS]Elevator
	SentBy int
	Role Role
}

type ButtonMessage struct {
	ButtonEvent elevio.ButtonEvent
	ElevatorID int
	MessageType MessageType
}