package elevatorControl

import (
	"Utils/utils"
	"Driver-go/elevio"
	"Network-go/network/bcast"
	"fmt"
	"time"
)

// This function receives button broadcasts from the master and sends them to the elevator
// It also sends a confirmation to the master
// It also receives cab calls from ButtonSender and sends them to the elevator
func ButtonListener(btnCh chan elevio.ButtonEvent, btnCabChan chan elevio.ButtonEvent, btnMasterChan chan utils.ButtonMessage) {

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

			btnCh <- btnEvent

			UpdateLights()

		case btnMsg := <- btnMasterChan:
			btnCh <- btnMsg.ButtonEvent

			UpdateLights()
		}
	}
}

// This function receives button events from the elevator and broadcasts them
// It also listens to reassigned hall calls when an elevator dies (and cab calls when an elevator comes back) and broadcasts them
// When a button is pressed we send it to the master untile we receive a confirmation
// If its a cab call its sent to the listener that sends to the elevator
func ButtonSender(btnReassignChan chan utils.ButtonMessage, btnCabChan chan elevio.ButtonEvent, btnMasterChan chan utils.ButtonMessage) {

	btnChan := make(chan elevio.ButtonEvent)
	go elevio.PollButtons(btnChan)
	for {
		select {
		// This is for calls sent by the elevator
		case btnEvent := <-btnChan:

			if WorldView.Role == utils.MASTER {
				btnMasterChan <- utils.ButtonMessage{
					ButtonEvent: btnEvent,
					ElevatorID:  WorldView.ElevatorID,
				}
			} else {			
				if btnEvent.Button == elevio.BT_Cab {
					btnCabChan <- btnEvent
				} else {
					go elevatorSenderUntilConfirmation(utils.ButtonMessage{
						ButtonEvent: btnEvent,
						ElevatorID:  WorldView.ElevatorID,
					})
				}
			}



		case btnMsg := <-btnReassignChan:
			// This is for reassigned calls
			go elevatorSenderUntilConfirmation(utils.ButtonMessage{
				ButtonEvent: btnMsg.ButtonEvent,
				ElevatorID:  btnMsg.ElevatorID,
			})

		}
	}
}

// This function sends the button press to the master until a confirmation is received
func elevatorSenderUntilConfirmation(btnMsg utils.ButtonMessage) {

	// Channels and broadcasts to send button presses to the master and receive the confirmation
	sendChan := make(chan utils.ButtonMessage)
	go bcast.Transmitter(utils.ElevatorToMasterPort, sendChan)

	receiveConfChan := make(chan utils.ButtonMessage)
	go bcast.Receiver(utils.MasterConfPort, receiveConfChan)

	ticker := time.NewTicker(50 * time.Millisecond)
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
