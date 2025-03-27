// fsm.go
package elevatorLogic

import (
	//"fmt"
	"Utils/utils"
	"Driver-go/elevio"
)

var elevator utils.Elevator

// Initialize the elevator
func init() {
	elevator = ElevatorUninitialized()
}

// Returns the elevator
func GetElevator() utils.Elevator {
	return elevator
}

// When the elevator starts this function is called to fix its position
func FsmOnInitBetweenFloors() {
	elevio.SetMotorDirection(elevio.MD_Down)
	elevator.Dirn = elevio.MD_Down
	elevator.Behaviour = utils.EB_Moving
}

// This function is called when a button is pressed and updates the requests
func FsmOnRequestButtonPress(btnFloor int, btnType int) {

	switch elevator.Behaviour {
	// If the door is open check if the call can be cleared immediately and resets the timer
	case utils.EB_DoorOpen:
		if requestsShouldClearImmediately(elevator, btnFloor, btnType) {
			TimerStart(utils.DOOR_OPEN_DURATION)
		} else {
			elevator.Requests[btnFloor][btnType] = true
		}

	case utils.EB_Moving:
		elevator.Requests[btnFloor][btnType] = true
	case utils.EB_Idle:

		// If the elevator is still it choses a direction to move towards the request
		elevator.Requests[btnFloor][btnType] = true
		pair := requestsChooseDirection(elevator)
		elevator.Dirn = pair.Dirn
		elevator.Behaviour = pair.Behaviour
		switch pair.Behaviour {
		case utils.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			TimerStart(utils.DOOR_OPEN_DURATION)
			elevator = requestsClearAtCurrentFloor(elevator)
		case utils.EB_Moving:
			elevio.SetMotorDirection(elevator.Dirn)
		case utils.EB_Idle:
		}
	}
}

// This function is called when the elevator gets to a floor
func FsmOnFloorArrival(newFloor int) {

	elevator.Floor = newFloor
	elevio.SetFloorIndicator(elevator.Floor)

	// It checks if the elevator should stop to serve a call
	if elevator.Behaviour == utils.EB_Moving {
		// If the elevator is stopped and has gotten to a floor it means its not stopped anymore
		if requestsShouldStop(elevator) || elevator.MotorStopped{
			elevator.MotorStopped = false
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			elevator = requestsClearAtCurrentFloor(elevator)
			TimerStart(utils.DOOR_OPEN_DURATION)
			elevator.Behaviour = utils.EB_DoorOpen
		}
	}

}

// This function is called when the timer times out
func FsmOnDoorTimeout() {
	// If theres an obstruction close the door
	
	if elevio.GetObstruction() {
		TimerStart(utils.DOOR_OPEN_DURATION)
		elevator.Obstructed = true
		return
	}
	elevator.Obstructed = false
	

	if elevator.Behaviour == utils.EB_DoorOpen {
		pair := requestsChooseDirection(elevator)
		elevator.Dirn = pair.Dirn
		elevator.Behaviour = pair.Behaviour

		switch elevator.Behaviour {
		case utils.EB_DoorOpen:
			TimerStart(utils.DOOR_OPEN_DURATION)
			elevator = requestsClearAtCurrentFloor(elevator)
		case utils.EB_Moving, utils.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevator.Dirn)
		}
	}

}
