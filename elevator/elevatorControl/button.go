package elevatorControl

import (
	"Config/config"
	"Network-go/network/bcast"
	"Driver-go/elevio"
	"fmt"
	"time"
)

// This function receives button broadcasts from the master and sends them to the elevator
// It also sends a confirmation to the master
func ButtonListener(btnCh chan elevio.ButtonEvent){

	// Channel and broadcasts to receive the button press from the master and send the confirmation
	receiveChan := make(chan config.ButtonMessage)
	go bcast.Receiver(config.MasterToElevatorPort,receiveChan)
	sendConfChan := make(chan config.ButtonMessage)
	go bcast.Transmitter(config.ElevatorConfPort, sendConfChan)

	for {
		select {
		case btnMsg := <- receiveChan:

			// If it's been assigned to us we send it to the elevator and send the confirmation to the master
			if btnMsg.ElevatorID == WorldView.ElevatorID {

				sendConfChan <- btnMsg
				btnCh <- btnMsg.ButtonEvent

				// When we simulate the button press we update the lights
				UpdateLights()
			}
		}
	}
}

// This function receives button events from the elevator and broadcasts them
// It also listens to reassigned hall calls when an elevator dies (and cab calls when an elevator comes back) and broadcasts them
// When a button is pressed we send it to the master untile we receive a confirmation
func ButtonSender(btnReassignChan chan config.ButtonMessage){

	btnChan := make(chan elevio.ButtonEvent)
	go elevio.PollButtons(btnChan)
	for {
		select {
			// This is for calls sent by the elevator
		case btnEvent := <- btnChan:
			go elevatorSenderUntilConfirmation(config.ButtonMessage{
				ButtonEvent: btnEvent,
				ElevatorID: WorldView.ElevatorID,
			})

		case btnMsg := <- btnReassignChan:
			// This is for reassigned calls
			go elevatorSenderUntilConfirmation(config.ButtonMessage{
				ButtonEvent: btnMsg.ButtonEvent,
				ElevatorID: btnMsg.ElevatorID,
			})

		}
	}
}

// This function sends the button press to the master until a confirmation is received 
func elevatorSenderUntilConfirmation(btnMsg config.ButtonMessage) {

	// Channels and broadcasts to send button presses to the master and receive the confirmation
	sendChan := make(chan config.ButtonMessage)
	go bcast.Transmitter(config.ElevatorToMasterPort, sendChan)
	receiveConfChan := make(chan config.ButtonMessage)
	go bcast.Receiver(config.MasterConfPort, receiveConfChan)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Every second we send the button press until we receive the confirmation from the master
	for {
		sendChan <- btnMsg
		select {
		case confMsg := <-receiveConfChan:
			if confMsg == btnMsg {
				return
			}
		case <-ticker.C:
			fmt.Println("Confirmation from MASTER not received")
		}
	}
}