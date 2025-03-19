// requests.go
package elevatorLogic

import (
	"Config/config"
	"Driver-go/elevio"
)

type DirnBehaviourPair struct {
	Dirn      elevio.MotorDirection
	Behaviour config.ElevatorBehaviour
}

// This function checks if there are any requests above
func requestsAbove(e config.Elevator) bool {
	for f := e.Floor + 1; f < config.N_FLOORS; f++ {
		for btn := 0; btn < config.N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

// This function checks if there are any requests below
func requestsBelow(e config.Elevator) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := 0; btn < config.N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

// This function checks if there are any requests here
func requestsHere(e config.Elevator) bool {
	for btn := 0; btn < config.N_BUTTONS; btn++ {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

// This function choses the direction based on the remaining requests
func requestsChooseDirection(e config.Elevator) DirnBehaviourPair {
	switch e.Dirn {
	case elevio.MD_Up:
		if requestsAbove(e) {
			return DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: config.EB_Moving}
		} else if requestsHere(e) {
			return DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: config.EB_DoorOpen}
		} else if requestsBelow(e) {
			return DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: config.EB_Moving}
		} else {
			return DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: config.EB_Idle}
		}
	case elevio.MD_Down:
		if requestsBelow(e) {
			return DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: config.EB_Moving}
		} else if requestsHere(e) {
			return DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: config.EB_DoorOpen}
		} else if requestsAbove(e) {
			return DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: config.EB_Moving}
		} else {
			return DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: config.EB_Idle}
		}
	case elevio.MD_Stop:
		if requestsHere(e) {
			return DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: config.EB_DoorOpen}
		} else if requestsAbove(e) {
			return DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: config.EB_Moving}
		} else if requestsBelow(e) {
			return DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: config.EB_Moving}
		} else {
			return DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: config.EB_Idle}
		}
	default:
		return DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: config.EB_Idle}
	}
}

// This function check whether the elevator should stop for a request
func requestsShouldStop(e config.Elevator) bool {
	switch e.Dirn {
	case elevio.MD_Down:
		return e.Requests[e.Floor][int(elevio.BT_HallDown)] ||
			e.Requests[e.Floor][int(elevio.BT_Cab)] ||
			!requestsBelow(e)
	case elevio.MD_Up:
		return e.Requests[e.Floor][int(elevio.BT_HallUp)] ||
			e.Requests[e.Floor][int(elevio.BT_Cab)] ||
			!requestsAbove(e)
	case elevio.MD_Stop:
		fallthrough
	default:
		return true
	}
}

// This function checks if a request should be cleared immediately (call on the same floor we are in)
func requestsShouldClearImmediately(e config.Elevator, btnFloor int, btnType int) bool {
	return e.Floor == btnFloor &&
		((e.Dirn == elevio.MD_Up && btnType == int(elevio.BT_HallUp)) ||
			(e.Dirn == elevio.MD_Down && btnType == int(elevio.BT_HallDown)) ||
			e.Dirn == elevio.MD_Stop ||
			btnType == int(elevio.BT_Cab))
}

// This function clear all the clearable requests at the current floor
func requestsClearAtCurrentFloor(e config.Elevator) config.Elevator {
	e.Requests[e.Floor][int(elevio.BT_Cab)] = false

	switch e.Dirn {
	case elevio.MD_Up:
		if !requestsAbove(e) && !e.Requests[e.Floor][int(elevio.BT_HallUp)] {
			e.Requests[e.Floor][int(elevio.BT_HallDown)] = false
		}
		e.Requests[e.Floor][int(elevio.BT_HallUp)] = false
	case elevio.MD_Down:
		if !requestsBelow(e) && !e.Requests[e.Floor][int(elevio.BT_HallDown)] {
			e.Requests[e.Floor][int(elevio.BT_HallUp)] = false
		}
		e.Requests[e.Floor][int(elevio.BT_HallDown)] = false
	case elevio.MD_Stop:
		fallthrough
	default:
		e.Requests[e.Floor][int(elevio.BT_HallUp)] = false
		e.Requests[e.Floor][int(elevio.BT_HallDown)] = false
	}
	return e
}
