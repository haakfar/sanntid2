package elevatorControl

import (
	"Config/config"
	"Network-go/network/bcast"
	"Driver-go/elevio"
	"fmt"
)

// This function manages the "master stuff"
func RunMaster(quitChan chan bool){

	// Channels and functions to receive and send button presses
	receiveChan := make(chan config.ButtonMessage)
	sendChan := make(chan config.ButtonMessage)
	go bcast.Receiver(config.ElevatorToMasterPort, receiveChan)
	go bcast.Transmitter(config.MasterToElevatorPort, sendChan)

	for {
		// Every time a button is pressed its sent to the master
		select {
		case btnMsg := <- receiveChan:
			// If the call is already assigned we ignore it
			if callAlreadyAssigned(btnMsg) {
				break
			}

			if btnMsg.ButtonEvent.Button == elevio.BT_Cab {
				// If its a cab call its assigned to the elevator that sent it
				sendChan <- btnMsg
				fmt.Println("Assigned cab call to", btnMsg.ElevatorID)

			} else {

				// If its a hall call its assiged to an elevator based on that the assigner says
				btnMsg.ElevatorID = assign(btnMsg.ButtonEvent)
				sendChan <- btnMsg					
				fmt.Println("Assigned hall call to", btnMsg.ElevatorID)
			}
		case <- quitChan:
			return
		}
	}
}

// This function checks if the call is already assigned.
func callAlreadyAssigned(btnMsg config.ButtonMessage) bool {

	alreadyAssigned := false

	// For cab calls we check if its assigned to the elevator that sent it
	if btnMsg.ButtonEvent.Button == elevio.BT_Cab {

		WorldViewMutex.Lock()
		if WorldView.Elevators[btnMsg.ElevatorID].Requests[btnMsg.ButtonEvent.Floor][btnMsg.ButtonEvent.Button] {
			alreadyAssigned = true
		}
		WorldViewMutex.Unlock()


	} else {

		// For hall calls we check if its assigned to any elevator
		for el := 0; el<config.N_ELEVATORS; el++{
			WorldViewMutex.Lock()
			if WorldView.Alive[el] && WorldView.Elevators[el].Requests[btnMsg.ButtonEvent.Floor][btnMsg.ButtonEvent.Button] {
				alreadyAssigned = true
			}
			WorldViewMutex.Unlock()
			if alreadyAssigned {
				break
			}
		}
	}
	return alreadyAssigned
}

// This is the assigner, very messy but it seems to work fine
// We try to calculate the time that each elevator would take to serve that call and in the end we assign the call to the elevator with the lowest time
func assign(btnEvent elevio.ButtonEvent) int {
	minTime := -1.0
	minEl := -1
	for el := 0; el < config.N_ELEVATORS; el++ {
		WorldViewMutex.Lock()
		if WorldView.Alive[el] {
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
func calcTime(elevator config.Elevator, btnEvent elevio.ButtonEvent) float64 {

	// if elevator is still its just the time to get to the floor
	if elevator.Behaviour == config.EB_Idle {
		return float64(abs(elevator.Floor-btnEvent.Floor))*2.5
	}

	// we create a copy of the elevator (this is a bad way to do it)
	
	var elevSim config.Elevator 
	elevSim.Floor = elevator.Floor
	elevSim.Dirn = elevator.Dirn
	elevSim.Behaviour = elevator.Behaviour	


	elevSim.Requests = make([][]bool, config.N_FLOORS)
	for i := range elevSim.Requests {
    	elevSim.Requests[i] = make([]bool, config.N_BUTTONS)
	}

	for floor := 0; floor< config.N_FLOORS; floor++{
		for btn := 0; btn< config.N_BUTTONS; btn++{
			elevSim.Requests[floor][btn] = elevator.Requests[floor][btn]
		}
	}
	elevSim.Requests[btnEvent.Floor][btnEvent.Button] = true
	time := 0.0
	for {

		// We get currentTop and bottom Destination
		topDest := getTopDestination(elevSim)
		bottomDest := getBottomDestination(elevSim)

		// If we are at a floor and one of those conditions is true we can stop
		if elevSim.Floor == btnEvent.Floor {
			if btnEvent.Button == elevio.BT_Cab || (elevSim.Dirn == elevio.MD_Up && btnEvent.Button == elevio.BT_HallUp) || (elevSim.Dirn == elevio.MD_Down && btnEvent.Button == elevio.BT_HallDown) || btnEvent.Floor == topDest || btnEvent.Floor == bottomDest || topDest == bottomDest{
				
				return time
			} 
		}

		// If we are stopping at this floor we have served the request and must wait 3 seconds
		if elevSim.Requests[elevSim.Floor][elevio.BT_Cab] {
			time+=3
			elevSim.Requests[elevSim.Floor][elevio.BT_Cab]=false
		}
		if elevSim.Requests[elevSim.Floor][elevio.BT_HallUp] && elevSim.Dirn == elevio.MD_Up {
			time+=3
			elevSim.Requests[elevSim.Floor][elevio.BT_HallUp]=false
		}
		if elevSim.Requests[elevSim.Floor][elevio.BT_HallDown] && elevSim.Dirn == elevio.MD_Down {
			time+=3
			elevSim.Requests[elevSim.Floor][elevio.BT_HallDown]=false
		}

		// If we reached the top or the bottom we have to change direction
		if elevSim.Floor >= topDest || elevSim.Floor == 3 {
			elevSim.Dirn = elevio.MD_Down
		} else if elevSim.Floor <= bottomDest || elevSim.Floor == 0 {
			elevSim.Dirn = elevio.MD_Up
		}

		// We move floors according to the direction (it takes about 2.5 seconds)
		if elevSim.Dirn == elevio.MD_Up {
			time+=2.5
			elevSim.Floor++
		} else if elevSim.Dirn == elevio.MD_Down {
			time+=2.5
			elevSim.Floor--
		}
	}
}

// Abs function since GO wants floats and I don't want to cast type
func abs(n int) int {
	if n < 0 {return -n}
	return n
}

// Function to get the current top destination of the elevator
func getTopDestination(elevator config.Elevator) int {
	for floor := config.N_FLOORS-1; floor>=0; floor--{
		for btn := 0; btn<config.N_BUTTONS; btn++{
			if elevator.Requests[floor][btn]{
				return floor
			}
		}
	}

	// If no calls are found we assume top destination = top floor
	return 3
}

// Function to get the current bottom destination of the elevator
func getBottomDestination(elevator config.Elevator) int {
	for floor := 0; floor< config.N_FLOORS; floor++{
		for btn := 0; btn< config.N_BUTTONS; btn++{
			if elevator.Requests[floor][btn]{
				return floor
			}
		}
	}
	// If no calls are found we assume bottom destination = bottom floor
	return 0
}