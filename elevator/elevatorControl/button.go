package elevatorControl

import (
	"Utils/utils"
	"Driver-go/elevio"
	"Network-go/network/bcast"
	//"fmt"
	"time"
)

// This module manages the button presses. When a button is pressed its sent 
// to the ReceiveButtonsFromElevator function and when a call is assigned to this elevator
// its sent to the SendButtonToElevator function

// This function receives the button presses that have been assigned to this elevator, in particular:
// If the elevator is master, it receives the button presses via btnMasterChan
// If its a cab call pressed on this elevator, it receives it via btnCabChan
// Otherwise its received via broadcast form the master and when its received, a confirmation is sent back
func SendButtonsToElevator(btnCh chan elevio.ButtonEvent, btnCabChan chan elevio.ButtonEvent, btnMasterChan chan utils.ButtonMessage) {

	// Channel and broadcasts to receive the button press from the master and send the confirmation
	receiveChan := make(chan utils.ButtonMessage)
	go bcast.Receiver(utils.MasterToElevatorPort, receiveChan)
	sendConfChan := make(chan utils.ButtonMessage)
	go bcast.Transmitter(utils.ElevatorConfPort, sendConfChan)

	for {
		select {
		case btnMsg := <-receiveChan:

			// If it's been assigned to us we send it to the elevator and send the confirmation to the master
			if btnMsg.ElevatorID == WorldView.ElevatorID {

				sendConfChan <- btnMsg
				btnCh <- btnMsg.ButtonEvent

				// When we simulate the button press we update the lights
				UpdateLights()
			}
		case btnEvent := <- btnCabChan:

			// If its a cab call we send it to the elevator
			btnCh <- btnEvent

			UpdateLights()

		case btnMsg := <- btnMasterChan:

			// If we are here it means the call is for us 
			btnCh <- btnMsg.ButtonEvent

			UpdateLights()
		}
	}
}

// This function receives the button presses that have been pressed on this elevator, in particular:
// If the elevator is master, they are sent to the master.go file via btnMasterChan
// If its a cab call its sent via btnCabChan
// Otherwise its sent via broadcast to the master until a confirmation is sent back
// Also reassigned calls are sent here (when an elevator dies its hall calls, when it comes back its cab calls) 
func ReceiveButtonsFromElevator(btnReassignChan chan utils.ButtonMessage, btnCabChan chan elevio.ButtonEvent, btnMasterChan chan utils.ButtonMessage) {

	btnChan := make(chan elevio.ButtonEvent)
	go elevio.PollButtons(btnChan)
	for {
		select {
		// This is for calls sent by the elevator
		case btnEvent := <-btnChan:

			// If its a cab call its sent to SendButtonsToElevator
			if btnEvent.Button == elevio.BT_Cab {
				btnCabChan <- btnEvent
			} else if WorldView.Role == utils.MASTER {

				// If we are master we send it via channel
				btnMasterChan <- utils.ButtonMessage{
					ButtonEvent: btnEvent,
					ElevatorID:  WorldView.ElevatorID,
				}
			} else {	

				// Otherwise we broadcast
				go elevatorSenderUntilConfirmation(utils.ButtonMessage{
					ButtonEvent: btnEvent,
					ElevatorID:  WorldView.ElevatorID,
				}, btnChan)
			}

		case btnMsg := <-btnReassignChan:

			// This is for reassigned calls

			// If we are master we send via channel, otherwise via broadcast
			if WorldView.Role == utils.MASTER {
				btnMasterChan <- btnMsg
			} else {
				go elevatorSenderUntilConfirmation(btnMsg, btnChan)
			}

		}
	}
}

// This function sends the button press to the master until a confirmation is received
func elevatorSenderUntilConfirmation(btnMsg utils.ButtonMessage, btnChan chan elevio.ButtonEvent) {

	// Channels and broadcasts to send button presses to the master and receive the confirmation
	sendChan := make(chan utils.ButtonMessage)
	go bcast.Transmitter(utils.ElevatorToMasterPort, sendChan)

	receiveConfChan := make(chan utils.ButtonMessage)
	go bcast.Receiver(utils.MasterConfPort, receiveConfChan)

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	// Every 50ms we send the button press until we receive the confirmation from the master
	for {
		sendChan <- btnMsg
		select {
		case confMsg := <-receiveConfChan:
			if confMsg == btnMsg {
				return
			}
		case <-ticker.C:
			if WorldView.Role == utils.MASTER {
				btnChan <- btnMsg.ButtonEvent
				return
			}
			
		}
	}
}
