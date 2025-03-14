package elevatorControl

import (
	"Driver-go/elevio"
	"Config/config"
)

func UpdateLights(){
	for floor := 0; floor < config.N_FLOORS; floor++ {

		// if theres a hall call on floor light up
		elevio.SetButtonLamp(elevio.BT_HallUp, floor, callOnFloor(floor, elevio.BT_HallUp))
		elevio.SetButtonLamp(elevio.BT_HallDown, floor, callOnFloor(floor, elevio.BT_HallDown))

		// light cab call 
		WorldViewMutex.Lock()
		elevio.SetButtonLamp(elevio.BT_Cab, floor, WorldView.Elevators[WorldView.ElevatorID].Requests[floor][elevio.BT_Cab])
		WorldViewMutex.Unlock()
	}
}