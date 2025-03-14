package elevatorControl

import (
	"Config/config"
	"Network-go/network/bcast"
	"Driver-go/elevio"
	//"fmt"
)

func ButtonListener(btnCh chan elevio.ButtonEvent){

	// receives button broadcasts and sends them to the elevator
	receiveChan := make(chan config.ButtonMessage)
	go bcast.Receiver(config.Port,receiveChan)

	for {
		select {
		case btnMsg := <- receiveChan:
			// if its SENT then its sent by the master
			if btnMsg.MessageType == config.SENT && btnMsg.ElevatorID == WorldView.ElevatorID {
				btnCh <- btnMsg.ButtonEvent
				//fmt.Println("Elevator ", WorldView.ElevatorID, " received")
			}
		}
	}
}


func ButtonSender(btnReassignChan chan config.ButtonMessage){

	// receives button from the elevator keyboard and sends them to the master
	sendChan := make(chan config.ButtonMessage)
	btnChan := make(chan elevio.ButtonEvent)
	go bcast.Transmitter(config.Port,sendChan)
	go elevio.PollButtons(btnChan)
	for {
		select {
		case btnEvent := <- btnChan:
			// if its RECEIVED then it must be received by the master
			sendChan <- config.ButtonMessage{
				ButtonEvent: btnEvent,
				ElevatorID: WorldView.ElevatorID,
				MessageType: config.RECEIVED,
			}
		case btnMsg := <- btnReassignChan:
			// reassign calls from dead elevators
			sendChan <- config.ButtonMessage{
				ButtonEvent: btnMsg.ButtonEvent,
				ElevatorID: btnMsg.ElevatorID,
				MessageType: config.RECEIVED,
			}
		}
	}
}
