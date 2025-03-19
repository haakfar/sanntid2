// fsm.go
package elevatorLogic

import (
	//"fmt"
	"Config/config"
	"Driver-go/elevio"
)

var elevator config.Elevator

// Initialize the elevator
func init() {
	elevator = ElevatorUninitialized()
}

// Returns the elevator
func GetElevator() config.Elevator {
	return elevator
}

// When the elevator starts this function is called to fix its position
func FsmOnInitBetweenFloors() {
	elevio.SetMotorDirection(elevio.MD_Down)
	elevator.Dirn = elevio.MD_Down
	elevator.Behaviour = config.EB_Moving
}

// This function is called when a button is pressed and updates the requests
func FsmOnRequestButtonPress(btnFloor int, btnType int) {

	switch elevator.Behaviour {
	// If the door is open check if the call can be cleared immediately and resets the timer
	case config.EB_DoorOpen:
		if requestsShouldClearImmediately(elevator, btnFloor, btnType) {
			TimerStart(config.DOOR_OPEN_DURATION)
		} else {
			elevator.Requests[btnFloor][btnType] = true
		}

	case config.EB_Moving:
		elevator.Requests[btnFloor][btnType] = true
	case config.EB_Idle:

		// If the elevator is still it choses a direction to move towards the request
		elevator.Requests[btnFloor][btnType] = true
		pair := requestsChooseDirection(elevator)
		elevator.Dirn = pair.Dirn
		elevator.Behaviour = pair.Behaviour
		switch pair.Behaviour {
		case config.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			TimerStart(config.DOOR_OPEN_DURATION)
			elevator = requestsClearAtCurrentFloor(elevator)
		case config.EB_Moving:
			elevio.SetMotorDirection(elevator.Dirn)
		case config.EB_Idle:
		}
	}
}

// This function is called when the elevator gets to a floor
func FsmOnFloorArrival(newFloor int) {

	elevator.Floor = newFloor
	elevio.SetFloorIndicator(elevator.Floor)

	// It checks if the elevator should stop to serve a call
	if elevator.Behaviour == config.EB_Moving {
		if requestsShouldStop(elevator) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			elevator = requestsClearAtCurrentFloor(elevator)
			TimerStart(config.DOOR_OPEN_DURATION)
			//setAllLights(elevator)
			elevator.Behaviour = config.EB_DoorOpen
		}
	}

}

// This function is called when the timer times out
func FsmOnDoorTimeout() {

	// If theres an obstruction close the door
	if elevio.GetObstruction() {
		TimerStart(config.DOOR_OPEN_DURATION)
		elevator.Obstructed = true
		return
	}
	elevator.Obstructed = false

	if elevator.Behaviour == config.EB_DoorOpen {
		pair := requestsChooseDirection(elevator)
		elevator.Dirn = pair.Dirn
		elevator.Behaviour = pair.Behaviour

		switch elevator.Behaviour {
		case config.EB_DoorOpen:
			TimerStart(config.DOOR_OPEN_DURATION)
			elevator = requestsClearAtCurrentFloor(elevator)
		case config.EB_Moving, config.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevator.Dirn)
		}
	}

}
