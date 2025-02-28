package elevatorControl

import (
	"Config/config"
	"Network-go/network/bcast"
	"Driver-go/elevio"
	"fmt"
	"sync"
)

var active [config.N_ELEVATORS] bool
var activeMu sync.Mutex
var wv config.WorldView

func RunMaster(updateChan chan config.ElevatorUpdate, worldViewChan chan config.WorldView, quitChan chan bool){
	receiveChan := make(chan config.ButtonMessage)
	sendChan := make(chan config.ButtonMessage)
	go bcast.Receiver(config.Port, receiveChan)
	go bcast.Transmitter(config.Port, sendChan)
	go detectElevators(updateChan, quitChan)
	go wvUpdater(worldViewChan, quitChan)

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
					btnMsg.ElevatorID = assign()
					btnMsg.MessageType = config.SENT
					sendChan <- btnMsg
					fmt.Println("Assigned hall call to", btnMsg.ElevatorID)
				}
			}
		case <- quitChan:
			return
		}
	}
}

func assign() int {
	/* honestly I forgot how this works but I think it calculates time for each elevator 
	by adding the difference between the top and bottom destinations of the elevator and 
	the distance between the elevator position and the top or bottom destination (we pick the biggest distance)
	and the assigned elevator is the one with the lowest time
	It's not the best but should work decently
	*/
	minTime := -1
	minEl := -1
	for el := 0; el < config.N_ELEVATORS; el++ {
		activeMu.Lock()
		if active[el] {
			activeMu.Unlock()
			if minEl == -1 {
				minEl = el
				minTime = calcTime(el)
			} else {
				time := calcTime(el)
				if time < minTime {
					minEl = el
					minTime = time
				}
			} 
		} else {
			activeMu.Unlock()
		}
	}
	return minEl
}

func calcTime(elevatorID int) int {
	topDest := getTopDestination(elevatorID)
	if topDest == -1 {
		return 0
	}
	bottomDest := getBottomDestination(elevatorID)
	return (topDest-bottomDest)+max(abs(topDest-worldView.Elevators[elevatorID].Floor),abs(bottomDest-worldView.Elevators[elevatorID].Floor))
}

// go functions for abs and max only work for float for some reason
// or maybe I'm just stupid
func abs(n int) int {
	if n < 0 {return -n}
	return n
}

func max(a int, b int) int{
	if a >= b {return a}
	return b
}

func getTopDestination(elevatorID int) int {
	for floor := config.N_FLOORS-1; floor>=0; floor--{
		for btn := 0; btn<config.N_BUTTONS; btn++{
			if worldView.Elevators[elevatorID].Requests[floor][btn]{
				return floor
			}
		}
	}
	return -1
}

func getBottomDestination(elevatorID int) int {
	for floor := 0; floor< config.N_FLOORS; floor++{
		for btn := 0; btn< config.N_BUTTONS; btn++{
			if worldView.Elevators[elevatorID].Requests[floor][btn]{
				return floor
			}
		}
	}
	return -1
}

// this updates which elevators are alive and which arent
func detectElevators(updateChan chan config.ElevatorUpdate, quitChan chan bool){
	for {
		select {
		case update := <- updateChan:
			activeMu.Lock()
			active[update.ElevatorID] = update.Alive
			activeMu.Unlock()
		case <- quitChan:
			return
		}
	}
}

// this updates the worldview
func wvUpdater(worldViewChan chan config.WorldView, quitChan chan bool){
	for {
		select {
		case worldview := <- worldViewChan:
			wv = worldview
		case <- quitChan:
			return
		}
	}
}