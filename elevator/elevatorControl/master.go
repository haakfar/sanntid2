package elevatorControl

import (
	"Config/config"
	"Network-go/network/bcast"
	"Driver-go/elevio"
	"fmt"
)

var exit bool

func RunMaster(quitChan chan bool){
	receiveChan := make(chan config.ButtonMessage)
	sendChan := make(chan config.ButtonMessage)
	go bcast.Receiver(config.Port, receiveChan)
	go bcast.Transmitter(config.Port, sendChan)

	for {
		// every time a button is pressed its sent to the master
		select {
		case btnMsg := <- receiveChan:
			if btnMsg.MessageType == config.RECEIVED {
				// if its a cab call its assigned to the elevator
				if btnMsg.ButtonEvent.Button == elevio.BT_Cab {
					btnMsg.MessageType = config.SENT
					sendChan <- btnMsg
					fmt.Println("Assigned cab call to", btnMsg.ElevatorID)
				} else {
					// if its a hall call is assiged to a suitable elevator
					btnMsg.ElevatorID = assign(btnMsg.ButtonEvent)
					btnMsg.MessageType = config.SENT
					sendChan <- btnMsg
					fmt.Println("Assigned hall call to", btnMsg.ElevatorID)
				}
			}
		case <- quitChan:
			exit = true
			return
		}
	}
}

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
	//fmt.Println("Assigning to ",minEl)
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
		// we get currentTop and bottom Destination
		topDest := getTopDestination(elevSim)
		bottomDest := getBottomDestination(elevSim)

		// if we are at a floor and one of those conditions is true we can stop
		if elevSim.Floor == btnEvent.Floor {
			if btnEvent.Button == elevio.BT_Cab || (elevSim.Dirn == elevio.MD_Up && btnEvent.Button == elevio.BT_HallUp) || (elevSim.Dirn == elevio.MD_Down && btnEvent.Button == elevio.BT_HallDown) || btnEvent.Floor == topDest || btnEvent.Floor == bottomDest || topDest == bottomDest{
				
				return time
			} 
		}

		// if we are stoppoing at this floor we have served the request and must wait 3 seconds
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

		// if we reached the top or the bottom we have to change direction
		if elevSim.Floor >= topDest || elevSim.Floor == 3 {
			elevSim.Dirn = elevio.MD_Down
		} else if elevSim.Floor <= bottomDest || elevSim.Floor == 0 {
			elevSim.Dirn = elevio.MD_Up
		}

		// we move floors according to the direction (it takes 2.5 seconds)
		if elevSim.Dirn == elevio.MD_Up {
			time+=2.5
			elevSim.Floor++
		} else if elevSim.Dirn == elevio.MD_Down {
			time+=2.5
			elevSim.Floor--
		}
	}
}

func abs(n int) int {
	if n < 0 {return -n}
	return n
}

func getTopDestination(elevator config.Elevator) int {
	for floor := config.N_FLOORS-1; floor>=0; floor--{
		for btn := 0; btn<config.N_BUTTONS; btn++{
			if elevator.Requests[floor][btn]{
				return floor
			}
		}
	}
	return 3
}

func getBottomDestination(elevator config.Elevator) int {
	for floor := 0; floor< config.N_FLOORS; floor++{
		for btn := 0; btn< config.N_BUTTONS; btn++{
			if elevator.Requests[floor][btn]{
				return floor
			}
		}
	}
	return 0
}

func hasCalls(elevator config.Elevator) bool {
	for floor := 0; floor< config.N_FLOORS; floor++{
		for btn := 0; btn< config.N_BUTTONS; btn++{
			if elevator.Requests[floor][btn]{
				return true
			}
		}
	}
	return false
}