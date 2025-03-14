// fsm.go
package elevatorLogic

import (
	//"fmt"
	"Driver-go/elevio"
	"Config/config"
)

var elevator config.Elevator

func init() {
	elevator = ElevatorUninitialized()
}

func GetElevator() config.Elevator {
	return elevator
}

/*
func setAllLights(e config.Elevator) {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, e.Requests[floor][btn])
			//fmt.Println(elevio.ButtonType(btn), floor, e.Requests[floor][btn])
		}
	}
}

*/

func FsmOnInitBetweenFloors() {
	elevio.SetMotorDirection(elevio.MD_Down)
	elevator.Dirn = elevio.MD_Down
	elevator.Behaviour = config.EB_Moving
}

func FsmOnRequestButtonPress(btnFloor int, btnType int) {
	//fmt.Printf("\n\nFsmOnRequestButtonPress(%d, %s)\n", btnFloor, buttonToString(btnType))
	//ElevatorPrint(elevator)

	switch elevator.Behaviour {
	case config.EB_DoorOpen:
		if requestsShouldClearImmediately(elevator, btnFloor, btnType) {
			TimerStart(config.DOOR_OPEN_DURATION)
		} else {
			elevator.Requests[btnFloor][btnType] = true
		}
	case config.EB_Moving:
		elevator.Requests[btnFloor][btnType] = true
	case config.EB_Idle:
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
	//setAllLights(elevator)
	//fmt.Println("\nNew state:")
	//ElevatorPrint(elevator)
}

func FsmOnFloorArrival(newFloor int) {
	//fmt.Printf("\n\nFsmOnFloorArrival(%d)\n", newFloor)
	//ElevatorPrint(elevator)

	elevator.Floor = newFloor
	elevio.SetFloorIndicator(elevator.Floor)

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

	//fmt.Println("\nNew state:")
	//ElevatorPrint(elevator)
}

func FsmOnDoorTimeout() {
	//fmt.Println("\n\nFsmOnDoorTimeout()")
	//ElevatorPrint(elevator)

	if elevio.GetObstruction() {
		TimerStart(config.DOOR_OPEN_DURATION)
		return
	}
	if elevator.Behaviour == config.EB_DoorOpen {
		pair := requestsChooseDirection(elevator)
		elevator.Dirn = pair.Dirn
		elevator.Behaviour = pair.Behaviour

		switch elevator.Behaviour {
		case config.EB_DoorOpen:
			TimerStart(config.DOOR_OPEN_DURATION)
			elevator = requestsClearAtCurrentFloor(elevator)
			//setAllLights(elevator)
		case config.EB_Moving, config.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevator.Dirn)
		}
	}

	//fmt.Println("\nNew state:")
	//ElevatorPrint(elevator)
}

func buttonToString(btnType int) string {
	switch elevio.ButtonType(btnType) {
	case elevio.BT_HallUp:
		return "HallUp"
	case elevio.BT_HallDown:
		return "HallDown"
	case elevio.BT_Cab:
		return "Cab"
	default:
		return "Unknown"
	}
}
