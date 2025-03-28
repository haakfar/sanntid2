// elevator.go
package elevatorLogic

import (
	"Utils/utils"
	"Driver-go/elevio"
	"fmt"
	"time"
)

// This module is the core of the single elevator logic, it operates completely ignoring the other elevators

// Function called when an elevator starts
func StartElevator(buttonCh chan elevio.ButtonEvent, elevatorCh chan utils.Elevator) {

	//Receives calls from buttonCh, sends updates when reaches a floor to elevator
	fmt.Println("Started!")

	inputPollRate := 25 * time.Millisecond

	// Start the elevator
	elevio.Init("localhost:15657", utils.N_FLOORS)

	// Channel and function to receive the elevator's floor
	floorCh := make(chan int)
	go elevio.PollFloorSensor(floorCh)

	// If we aren't on any floor (between floors) we run the init function (it goes down until it gets to a floor)
	currentFloor := elevio.GetFloor()
	if currentFloor == -1 {
		FsmOnInitBetweenFloors()
	}

	ticker := time.NewTicker(inputPollRate)
	defer ticker.Stop()

	// Sends the elevator to the elevatorManager that updates the world view
	elevatorCh <- GetElevator()

	// Function that detects obstructions
	go obstructionDetector(elevatorCh)

	// Function that detects motor stops
	go motorStopDetector(elevatorCh)

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

// This function is called when an obstruction or a motor stop is detected
// We update the worldview of the obstruction/motor stop, we wait for 1 second
// to make sure that the worldview is updated for all elevators and that the calls
// have been reassigned and then we remove the hall calls and update the worldview again
func removeHallCalls(elevatorCh chan utils.Elevator){
	// Updating worldview
	elevatorCh <- elevator

	// Waiting 1 sec for everything to update (its a relatively long wait but its just to make sure even with packetloss)
	time.Sleep(1 * time.Second)

	// Removing hall calls
	for floor := 0; floor < utils.N_FLOORS; floor++ {
		for btn := 0; btn < utils.N_BUTTONS-1; btn++ {
			elevator.Requests[floor][btn] = false
		}
	}

	// Updating worldview
	elevatorCh <- elevator
}

// This function manages the obstruction
// When the obstruction is activated the hall calls are removed and everything is updated
// When the obstruction is removed the worldview is updated
func obstructionDetector(elevatorCh chan utils.Elevator){
	for {
		if elevio.GetObstruction() != elevator.Obstructed {
			elevator.Obstructed = elevio.GetObstruction()
			if elevator.Obstructed {

				// If we have an obstruction we have to remove the hall calls
				go removeHallCalls(elevatorCh)

			} else {

				// If we have don't have an obstruction anymore we update the world view
				elevatorCh <- elevator
			}

		}
	}
}

// This function detects the motor stop
// If the elevator is in a moving state but hasn't gotten to any new floor for 4 seconds it means it has stopped
func motorStopDetector(elevatorCh chan utils.Elevator) {
	const timeout = 4 * time.Second
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	lastFloor := elevator.Floor
	lastChange := time.Now()

	for {

		// We check every 100ms
		<-ticker.C

		// If we got to a new floor we are not stopped
		if elevator.Floor != lastFloor {
			lastFloor = elevator.Floor
			lastChange = time.Now()
		}

		// If the eleavtor is idle or has its doors open its not stopped
		if elevator.Behaviour == utils.EB_Idle || elevator.Behaviour == utils.EB_DoorOpen {
			lastChange = time.Now()
		}

		// If we are moving and its been more than 4sec since the last change the motor has stopped
		if elevator.Dirn != elevio.MD_Stop && time.Since(lastChange) > timeout {
			
			if !elevator.MotorStopped {
				elevator.MotorStopped = true

				// We remove the hall calls and update everything
				go removeHallCalls(elevatorCh)
			}
		}
	}
}

// Function to initialize the elevator
func ElevatorUninitialized() utils.Elevator {
	req := make([][]bool, utils.N_FLOORS)
	for i := 0; i < utils.N_FLOORS; i++ {
		req[i] = make([]bool, utils.N_BUTTONS)
	}
	return utils.Elevator{
		Floor:     -1,
		Dirn:      elevio.MD_Stop,
		Behaviour: utils.EB_Idle,
		Requests:  req,
		Obstructed: false,
		MotorStopped: false,
	}
}
