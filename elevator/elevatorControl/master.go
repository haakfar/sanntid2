package elevatorControl

import (
	"Utils/utils"
	"Driver-go/elevio"
	"Network-go/network/bcast"
	"fmt"
	"time"
)

var exit bool

// This function manages the "master stuff" (better explained in the function)
func RunMaster(quitChan chan bool) {

	// Channels and broadcasts to receive button presses and send confirmation
	receiveChan := make(chan utils.ButtonMessage)
	go bcast.Receiver(utils.ElevatorToMasterPort, receiveChan)
	sendConfChan := make(chan utils.ButtonMessage)
	go bcast.Transmitter(utils.MasterConfPort, sendConfChan)

	for {
		// Every time a button is pressed its sent to the master
		select {
		case btnMsg := <-receiveChan:

			// We send the confirmation back
			sendConfChan <- btnMsg

			// If the call is already assigned we ignore it
			if callAlreadyAssigned(btnMsg) {
				fmt.Println("Call already assigned")
				break
			}

			if btnMsg.ButtonEvent.Button == elevio.BT_Cab {

				// If its a cab call its assigned to the elevator that sent it
				fmt.Println("Assigned cab call to", btnMsg.ElevatorID)
				go masterSenderUntilConfirmation(btnMsg)

			} else {

				// If its a hall call its assiged to an elevator based on that the assigner says
				btnMsg.ElevatorID = FindBestElevator(btnMsg.ButtonEvent)
				fmt.Println("Assigned hall call to", btnMsg.ElevatorID)
				go masterSenderUntilConfirmation(btnMsg)
			}

		// When an elevator demotes it terminates
		case <-quitChan:
			exit = true
			return
		}
	}
}

// This function sends the button press to the assigned elevator until a confirmation is received
func masterSenderUntilConfirmation(btnMsg utils.ButtonMessage) {

	// Channels and broadcasts to send button presses to the elevator and receive the confirmation
	sendChan := make(chan utils.ButtonMessage)
	go bcast.Transmitter(utils.MasterToElevatorPort, sendChan)

	receiveConfChan := make(chan utils.ButtonMessage)
	go bcast.Receiver(utils.ElevatorConfPort, receiveConfChan)

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	// Every second we send the button press until we receive the confirmation from the elevator
	for {
		sendChan <- btnMsg
		select {
		case confMsg := <-receiveConfChan:
			if confMsg == btnMsg {
				return
			}
		case <-ticker.C:

			// If exit is true then the elevator isn't master anymore and must terminate
			if exit {
				return
			}

			// If the assigned elevator is not alive anymore we must terminate (it should be assigned again)
			WorldViewMutex.Lock()
			if !WorldView.Alive[btnMsg.ElevatorID] {
				WorldViewMutex.Unlock()
				return
			}
			WorldViewMutex.Unlock()

			fmt.Println("Confirmation from elevator not received")
		}
	}
}

// This function checks if the call is already assigned.
func callAlreadyAssigned(btnMsg utils.ButtonMessage) bool {

	alreadyAssigned := false

	// For cab calls we check if its assigned to the elevator that sent it
	if btnMsg.ButtonEvent.Button == elevio.BT_Cab {

		WorldViewMutex.Lock()
		if WorldView.Elevators[btnMsg.ElevatorID].Requests[btnMsg.ButtonEvent.Floor][btnMsg.ButtonEvent.Button] {
			alreadyAssigned = true
		}
		WorldViewMutex.Unlock()

	} else {

		// For hall calls we check if its assigned to any elevator
		for el := 0; el < utils.N_ELEVATORS; el++ {
			WorldViewMutex.Lock()
			if WorldView.Alive[el] && WorldView.Elevators[el].Requests[btnMsg.ButtonEvent.Floor][btnMsg.ButtonEvent.Button] {
				alreadyAssigned = true
			}
			WorldViewMutex.Unlock()
			if alreadyAssigned {
				break
			}
		}
	}

	return alreadyAssigned
}
