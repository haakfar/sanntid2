// elevator.go
package elevatorLogic

import (
	"Utils/utils"
	"Driver-go/elevio"
	"fmt"
	"time"
)

// Function called when an elevator starts
func StartElevator(buttonCh chan elevio.ButtonEvent, elevatorCh chan utils.Elevator) {

	//Receives calls from buttonCh, sends updates when reaches a floor to elevator
	fmt.Println("Started!")

	inputPollRate := 25 * time.Millisecond

	elevio.Init("localhost:15657", utils.N_FLOORS)

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

	go obstructionManager(elevatorCh)

	go detectMotorStop(elevatorCh)

	for {
		select {
		// When a button is pressed the elevator processes it and then updates the world view through elevatorCh
		case btnEvent := <-buttonCh:
			FsmOnRequestButtonPress(btnEvent.Floor, int(btnEvent.Button))
			elevatorCh <- elevator
		case f := <-floorCh:
			// When the elevator arrives at a floor it processes it and then updates the world view
			/*
			if elevator.MotorStopped {
				fmt.Println("Motor restarted")
				elevator.MotorStopped = false
				elevio.SetMotorDirection(elevio.MD_Stop)
				elevator.Dirn = elevio.MD_Stop
				elevator.Behaviour = utils.EB_Idle
			}
				*/
			FsmOnFloorArrival(f)

			elevatorCh <- elevator
			//fmt.Println("Arrived on floor",f)
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

func removeHallCalls(elevatorCh chan utils.Elevator){
	elevatorCh <- elevator

	time.Sleep(1 * time.Second)

	for floor := 0; floor < utils.N_FLOORS; floor++ {
		for btn := 0; btn < utils.N_BUTTONS-1; btn++ {
			elevator.Requests[floor][btn] = false
		}
	}

	elevatorCh <- elevator
}

func obstructionManager(elevatorCh chan utils.Elevator){
	for {
		if elevio.GetObstruction() != elevator.Obstructed {
			elevator.Obstructed = elevio.GetObstruction()
			if elevator.Obstructed {

				go removeHallCalls(elevatorCh)

			} else {
				elevatorCh <- elevator
			}

		}
	}
}

func detectMotorStop(elevatorCh chan utils.Elevator) {
	const timeout = 4 * time.Second
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	lastFloor := elevator.Floor
	lastChange := time.Now()

	for {

		<-ticker.C

		if elevator.Floor != lastFloor {
			lastFloor = elevator.Floor
			lastChange = time.Now()
		}

		if elevator.Behaviour == utils.EB_Idle || elevator.Behaviour == utils.EB_DoorOpen {
			lastChange = time.Now()
		}


		if elevator.Dirn != elevio.MD_Stop && time.Since(lastChange) > timeout {
			
			if !elevator.MotorStopped {
				fmt.Println("Motor stopped")
				elevator.MotorStopped = true
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
