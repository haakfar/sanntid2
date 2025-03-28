package elevatorControl

import (
	"Utils/utils"
	"Driver-go/elevio"
	"Network-go/network/bcast"
	"fmt"
	"time"
)

// This module manages the master behaviour of the elevator

// Variable to exit when the elevator is not master anymore
var exit bool

// This function manages the "master stuff", that is:
// Recives the button presses via masterReceiveChan if pressed on this elevator or via broadcast (sends confirmation back if via broadcast)
// Decides to which elevator to assign the call and sends them to the elevator via masterSendChan if assigned to this elevator or via broadcast if not and keeps sending until confirmation
func RunMaster(quitChan chan bool, masterReceiveChan chan utils.ButtonMessage, masterSendChan chan utils.ButtonMessage) {

	// Channels and broadcasts to receive button presses and send confirmation
	receiveChan := make(chan utils.ButtonMessage)
	go bcast.Receiver(utils.ElevatorToMasterPort, receiveChan)
	sendConfChan := make(chan utils.ButtonMessage)
	go bcast.Transmitter(utils.MasterConfPort, sendConfChan)

	for {
		// Every time a button is pressed its sent to the master
		select {

		// If we receive a button press via broadcast
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
				// NOTE: this is for reassigning calls, when an elevator that had cab calls comes back
				// Normally cab calls should be managed by the elevator that received them 
				fmt.Println("Assigned cab call to", btnMsg.ElevatorID)
				go masterSenderUntilConfirmation(btnMsg)

			} else {

				// If its a hall call its assiged to an elevator based on what the assigner says
				btnMsg.ElevatorID = FindBestElevator(btnMsg.ButtonEvent)

				// If we don't find any good elevator, or the call is assigned to us (the master), we send it back via channel 
				if btnMsg.ElevatorID == -1 || btnMsg.ElevatorID == WorldView.ElevatorID{
					btnMsg.ElevatorID = WorldView.ElevatorID
					masterSendChan <- btnMsg
				} else {
					// Otherwise the call will be sent via broadcast
					go masterSenderUntilConfirmation(btnMsg)
				}
				fmt.Println("Assigned hall call to", btnMsg.ElevatorID)
			}

		case btnMsg := <-masterReceiveChan:

			// This is for when the calls are pressed on the master's keypad, its very similar to the other
			fmt.Println("Received call as master")

			if callAlreadyAssigned(btnMsg) {
				fmt.Println("Call already assigned")
				break
			}

			if btnMsg.ButtonEvent.Button == elevio.BT_Cab {

				// If its a cab call its assigned to the elevator that sent it
				fmt.Println("Assigned cab call to", btnMsg.ElevatorID)

				if btnMsg.ElevatorID == WorldView.ElevatorID {
					// If its assigned to us its sent via channel
					masterSendChan <- btnMsg
				} else {
					// Otherwise it will be broadcasted
					go masterSenderUntilConfirmation(btnMsg)
				}

			} else {

				// Same as before (i think)
				btnMsg.ElevatorID = FindBestElevator(btnMsg.ButtonEvent)				
				if btnMsg.ElevatorID == -1 || btnMsg.ElevatorID == WorldView.ElevatorID{
					btnMsg.ElevatorID = WorldView.ElevatorID
					masterSendChan <- btnMsg
				} else {
					go masterSenderUntilConfirmation(btnMsg)
				}
				fmt.Println("Assigned hall call to", btnMsg.ElevatorID)
			}

		// When the elevator demotes it terminates
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

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	// Every 50ms we send the button press until we receive the confirmation from the elevator
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

			// If the assigned elevator is not alive anymore we must terminate the sending (it should be assigned again)
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
			if WorldView.Alive[el] && !WorldView.Elevators[el].Obstructed && !WorldView.Elevators[el].MotorStopped && WorldView.Elevators[el].Requests[btnMsg.ButtonEvent.Floor][btnMsg.ButtonEvent.Button] {
				fmt.Println("Call already assigned to", el)
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
