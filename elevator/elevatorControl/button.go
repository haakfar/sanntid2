package elevatorControl

import (
	"Config/config"
	"Network-go/network/bcast"
	"Driver-go/elevio"
	//"fmt"
)

// This function receives button broadcasts from the master and sends them to the elevator
func ButtonListener(btnCh chan elevio.ButtonEvent){

	receiveChan := make(chan config.ButtonMessage)
	go bcast.Receiver(config.MasterToElevatorPort,receiveChan)

	for {
		select {
		case btnMsg := <- receiveChan:

			// If it's been assigned to us we send it to the elevator
			if btnMsg.ElevatorID == WorldView.ElevatorID {
				btnCh <- btnMsg.ButtonEvent
			}
		}
	}
}

// This function receives button events from the elevator and broadcasts them
// It also listens to reassigned hall calls when an elevator dies (and cab calls when an elevator comes back) and broadcasts them
func ButtonSender(btnReassignChan chan config.ButtonMessage){

	sendChan := make(chan config.ButtonMessage)
	btnChan := make(chan elevio.ButtonEvent)
	go bcast.Transmitter(config.ElevatorToMasterPort,sendChan)
	go elevio.PollButtons(btnChan)
	for {
		select {
			// This is for calls sent by the elevator
		case btnEvent := <- btnChan:
			sendChan <- config.ButtonMessage{
				ButtonEvent: btnEvent,
				ElevatorID: WorldView.ElevatorID,
			}
		case btnMsg := <- btnReassignChan:
			// This is for reassigned calls
			sendChan <- config.ButtonMessage{
				ButtonEvent: btnMsg.ButtonEvent,
				ElevatorID: btnMsg.ElevatorID,
			}
		}
	}
}
