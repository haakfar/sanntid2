package elevatorControl

import (
	"Utils/utils"
	"Driver-go/elevio"
	"math"
)

// This is the assigner module, very messy but it seems to work fine

// We try to calculate the time that each elevator would take to serve that call and in the end we assign the call to the elevator with the lowest time
func FindBestElevator(btnEvent elevio.ButtonEvent) int {
	minTime := -1.0
	minEl := -1
	for el := 0; el < utils.N_ELEVATORS; el++ {
		WorldViewMutex.Lock()
		if WorldView.Alive[el] && !WorldView.Elevators[el].Obstructed && !WorldView.Elevators[el].MotorStopped {
			WorldViewMutex.Unlock()
			if minEl == -1 {
				minEl = el
				minTime = calcTime(WorldView.Elevators[el], btnEvent)
			} else {
				time := calcTime(WorldView.Elevators[el], btnEvent)
				if time < minTime {
					minEl = el
					minTime = time
				}
			}
		} else {
			WorldViewMutex.Unlock()
		}
	}
	return minEl
}

// here we try to simulate how much time it takes for the elevator to serve that call (2.5 seconds to move between floors, 3 seconds when stopping at the floor)
// I tried it quite a lot and it seems to not crash anymore
func calcTime(elevator utils.Elevator, btnEvent elevio.ButtonEvent) float64 {

	// if elevator is still its just the time to get to the floor
	if elevator.Behaviour == utils.EB_Idle {
		return math.Abs(float64(elevator.Floor-btnEvent.Floor)) * 2.5
	}

	// we create a copy of the elevator 

	elevSim := elevator

	elevSim.Requests = make([][]bool, len(elevator.Requests))
	for i := range elevator.Requests {
		elevSim.Requests[i] = append([]bool{}, elevator.Requests[i]...) 
	}
	elevSim.Requests[btnEvent.Floor][btnEvent.Button] = true

	time:=0.0

	switch(elevSim.Behaviour){
	case utils.EB_Moving:
		time+= 1.25 // It takes about 2.5 seconds to move floors so if the elevator is moving we put half the time
		switch (elevSim.Dirn){
		case elevio.MD_Up:
			elevSim.Floor++
		case elevio.MD_Down:
			elevSim.Floor--
		}
	case utils.EB_DoorOpen:
		time+= 1.5 // Doors stay open for 3 seconds so if the elevator has its doors open we put half the time
	}

	// Sometimes the floor gets outside the possible values so we fix it
	if elevSim.Floor<0 {
		elevSim.Floor = 0
	}
	if elevSim.Floor>3 {
		elevSim.Floor = 3
	}

	// Sometimes the loop gets stuck so we count the loops and when it gets stuck we exit with a high time
	// Of course its not ideal but it gets the job done
	loops:=0 

	for {

		if loops>=50 {
			// If we're here it means the loop probably got stuck so we return 500 
			return 500
		}

		// If we should stop on the floor we clear the clearable requests 
		if shouldStop(elevSim) {
			elevSim = clearRequests(elevSim)

			if !elevSim.Requests[btnEvent.Floor][btnEvent.Button] {
				// If we cleared the request of the button event we return
				return time
			}

			// Otherwise we add the door opening time and decide the next direction
			time += 3
			elevSim.Dirn = getDirection(elevSim)
		}

		// Moving floors
		switch (elevSim.Dirn){
		case elevio.MD_Up:
			elevSim.Floor++
		case elevio.MD_Down:
			elevSim.Floor--
		}

		// Fixing floor position if it gets outside the possible positions
		if elevSim.Floor<0 {
			elevSim.Floor = 0
		}
		if elevSim.Floor>3 {
			elevSim.Floor = 3
		}

		// The elevator takes about 2.5 seconds to move floors
		time+=2.5

		loops++
	}
}

// This function gets the next direction the elevator should take based on the calls
func getDirection(elevator utils.Elevator) elevio.MotorDirection {
	switch elevator.Dirn {
	case elevio.MD_Up:
		if elevator.Floor < getTopDestination(elevator) {
			return elevio.MD_Up
		} else {
			return elevio.MD_Down
		}
	case elevio.MD_Down:
		if elevator.Floor > getBottomDestination(elevator) {
			return elevio.MD_Down
		} else {
			return elevio.MD_Up
		}
	case elevio.MD_Stop:
		if elevator.Floor < getTopDestination(elevator) {
			return elevio.MD_Up
		} else if elevator.Floor > getBottomDestination(elevator) {
			return elevio.MD_Down
		} else {
			return elevio.MD_Stop
		}
	}
	return elevio.MD_Stop
}

// This function decides whether the elevator should stop
func shouldStop(elevator utils.Elevator) bool {
	switch elevator.Dirn {
	case elevio.MD_Down:
		return elevator.Requests[elevator.Floor][elevio.BT_HallDown] || elevator.Requests[elevator.Floor][elevio.BT_Cab] || elevator.Floor == getBottomDestination(elevator)
	case elevio.MD_Up:
		return elevator.Requests[elevator.Floor][elevio.BT_HallUp] || elevator.Requests[elevator.Floor][elevio.BT_Cab] || elevator.Floor == getTopDestination(elevator)
	default:
		return true
	}
}

// This function clears the clearable requests on the floor
func clearRequests(elevator utils.Elevator) utils.Elevator{
	elevator.Requests[elevator.Floor][elevio.BT_Cab] = false
	switch elevator.Dirn {
	case elevio.MD_Up:
		if elevator.Floor == getTopDestination(elevator) && !elevator.Requests[elevator.Floor][int(elevio.BT_HallUp)] {
			elevator.Requests[elevator.Floor][int(elevio.BT_HallDown)] = false
		}
		elevator.Requests[elevator.Floor][int(elevio.BT_HallUp)] = false
	case elevio.MD_Down:
		if elevator.Floor == getBottomDestination(elevator) && !elevator.Requests[elevator.Floor][int(elevio.BT_HallDown)] {
			elevator.Requests[elevator.Floor][int(elevio.BT_HallUp)] = false
		}
		elevator.Requests[elevator.Floor][int(elevio.BT_HallDown)] = false
	default:
		elevator.Requests[elevator.Floor][int(elevio.BT_HallUp)] = false
		elevator.Requests[elevator.Floor][int(elevio.BT_HallDown)] = false
	}
	return elevator
}

// Function to get the current top destination of the elevator
func getTopDestination(elevator utils.Elevator) int {
	for floor := utils.N_FLOORS - 1; floor >= 0; floor-- {
		for btn := 0; btn < utils.N_BUTTONS; btn++ {
			if elevator.Requests[floor][btn] {
				return floor
			}
		}
	}

	// If no calls are found we assume top destination = top floor
	return 3
}

// Function to get the current bottom destination of the elevator
func getBottomDestination(elevator utils.Elevator) int {
	for floor := 0; floor < utils.N_FLOORS; floor++ {
		for btn := 0; btn < utils.N_BUTTONS; btn++ {
			if elevator.Requests[floor][btn] {
				return floor
			}
		}
	}
	
	// If no calls are found we assume bottom destination = bottom floor
	return 0
}
