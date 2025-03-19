// elevator.go
package elevatorLogic

import (
	"Config/config"
	"Driver-go/elevio"
	"fmt"
	"time"
)

// Function called when an elevator starts
func StartElevator(buttonCh chan elevio.ButtonEvent, elevatorCh chan config.Elevator) {

	//Receives calls from buttonCh, sends updates when reaches a floor to elevator
	fmt.Println("Started!")

	inputPollRate := 25 * time.Millisecond

	elevio.Init("localhost:15657", config.N_FLOORS)

	floorCh := make(chan int)

	go elevio.PollFloorSensor(floorCh)

	currentFloor := elevio.GetFloor()
	if currentFloor == -1 {
		FsmOnInitBetweenFloors()
	}

	ticker := time.NewTicker(inputPollRate)
	defer ticker.Stop()

	// Sends the elevator to the elevatorManager that updates the world view
	elevatorCh <- GetElevator()

	for {
		select {
		// When a button is pressed the elevator processes it and then updates the world view through elevatorCh
		case btnEvent := <-buttonCh:
			FsmOnRequestButtonPress(btnEvent.Floor, int(btnEvent.Button))
			elevatorCh <- elevator
		case f := <-floorCh:
			// When the elevator arrives at a floor it processes it and then updates the world view
			FsmOnFloorArrival(f)
			elevatorCh <- elevator
		case <-ticker.C:
			// When the timer times out, the elevator processes it and then updates the world view
			if TimerTimedOut() {
				TimerStop()
				FsmOnDoorTimeout()
				elevatorCh <- elevator
			}
		}
	}
}

// Function to initialize the elevator
func ElevatorUninitialized() config.Elevator {
	req := make([][]bool, config.N_FLOORS)
	for i := 0; i < config.N_FLOORS; i++ {
		req[i] = make([]bool, config.N_BUTTONS)
	}
	return config.Elevator{
		Floor:     -1,
		Dirn:      elevio.MD_Stop,
		Behaviour: config.EB_Idle,
		Requests:  req,
	}
}
