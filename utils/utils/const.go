package utils

// Constants
const N_FLOORS = 4
const N_ELEVATORS = 3
const N_BUTTONS = 3
const DOOR_OPEN_DURATION = 3.0

// Ports
const WorldViewPort = 9000
const MasterToElevatorPort = 9001
const ElevatorToMasterPort = 9002
const ElevatorConfPort = 9003
const MasterConfPort = 9004

// Roles
const (
	SLAVE Role = iota
	BACKUP
	MASTER
)

// Elevator behaviours
const (
	EB_Idle ElevatorBehaviour = iota
	EB_DoorOpen
	EB_Moving
)