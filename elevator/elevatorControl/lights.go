package elevatorControl

import (
	"Driver-go/elevio"
	"Utils/utils"
)

func UpdateLights(){
	for floor := 0; floor < utils.N_FLOORS; floor++ {

		// If theres a hall call on the floor light up
		elevio.SetButtonLamp(elevio.BT_HallUp, floor, callOnFloor(floor, elevio.BT_HallUp))
		elevio.SetButtonLamp(elevio.BT_HallDown, floor, callOnFloor(floor, elevio.BT_HallDown))

		// Light up cab calls only if relative to the elevator
		WorldViewMutex.Lock()
		elevio.SetButtonLamp(elevio.BT_Cab, floor, WorldView.Elevators[WorldView.ElevatorID].Requests[floor][elevio.BT_Cab])
		WorldViewMutex.Unlock()
	}
}

// For every elevator alive check if theres a call on floor
func callOnFloor(floor int, call elevio.ButtonType) bool {

	for el := 0; el < utils.N_ELEVATORS; el++{
		WorldViewMutex.Lock()
		if !WorldView.Alive[el] {
			WorldViewMutex.Unlock()
			continue
		}
		WorldViewMutex.Unlock()
		WorldViewMutex.Lock()
		if WorldView.Elevators[el].Requests[floor][call] {
			WorldViewMutex.Unlock()
			return true
		}
		WorldViewMutex.Unlock()
	}
	return false
}
