// elevator.go
package elevatorLogic

import (
	"fmt"
	"Driver-go/elevio"
	"time"
	"Config/config"
)

func StartElevator(buttonCh chan elevio.ButtonEvent, elevatorCh chan config.Elevator) {

	//receives calls from buttonCh, sends updates when reaches a floor to elevator
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

	elevatorCh <- GetElevator()

	for {
		select {
		case btnEvent := <-buttonCh:
			FsmOnRequestButtonPress(btnEvent.Floor, int(btnEvent.Button))
			elevatorCh <- elevator
		case f := <-floorCh:
			FsmOnFloorArrival(f)
			elevatorCh <- elevator
		case <-ticker.C:
			if TimerTimedOut() {
				TimerStop()
				FsmOnDoorTimeout()
				elevatorCh <- elevator
			}
		}
	}
}
/*

func (eb config.ElevatorBehaviour) String() string {
	switch eb {
	case config.EB_Idle:
		return "EB_Idle"
	case config.EB_DoorOpen:
		return "EB_DoorOpen"
	case config.EB_Moving:
		return "EB_Moving"
	default:
		return "EB_UNDEFINED"
	}
}

*/

/*

func ElevatorPrint(e config.Elevator) {
	fmt.Println("  +--------------------+")
	fmt.Printf("  |floor = %-2d          |\n", e.Floor)
	fmt.Printf("  |dirn  = %-12s|\n", motorDirToString(e.Dirn))
	fmt.Printf("  |behav = %-12s|\n", e.Behaviour.String())
	fmt.Println("  +--------------------+")
	fmt.Println("  |  | up  | dn  | cab |")
	for f := config.N_FLOORS - 1; f >= 0; f-- {
		fmt.Printf("  | %d", f)
		for btn := 0; btn < config.N_BUTTONS; btn++ {
			if (f == config.N_FLOORS-1 && btn == int(elevio.BT_HallUp)) || (f == 0 && btn == int(elevio.BT_HallDown)) {
				fmt.Printf("|     ")
			} else {
				if e.Requests[f][btn] {
					fmt.Printf("|  #  ")
				} else {
					fmt.Printf("|  -  ")
				}
			}
		}
		fmt.Println("|")
	}
	fmt.Println("  +--------------------+")
}

*/

func ElevatorUninitialized() config.Elevator {
	req := make([][]bool, config.N_FLOORS)
	for i := 0; i < config.N_FLOORS; i++ {
		req[i] = make([]bool, config.N_BUTTONS)
	}
	return config.Elevator{
		Floor:     -1,
		Dirn:      elevio.MD_Stop,
		Behaviour: config.EB_Idle,
		Requests: req,
	}
}

func motorDirToString(dir elevio.MotorDirection) string {
	switch dir {
	case elevio.MD_Up:
		return "Up"
	case elevio.MD_Down:
		return "Down"
	case elevio.MD_Stop:
		return "Stop"
	default:
		return "Unknown"
	}
}
