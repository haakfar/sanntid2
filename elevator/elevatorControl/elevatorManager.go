package elevatorControl

import (
	"Utils/utils"
	"Driver-go/elevio"
	"Elevator/elevatorLogic"
	"Network-go/network/bcast"
	"fmt"
	"strconv"
	"sync"
	"time"
)

var WorldViewMutex sync.Mutex
var WorldView utils.WorldView

// This module is the core of the program and manages the broadcast of the world view
// It sends the world view via broadcast and listens to the world views sent by the others and acts
// based on what is received

// Function to initialize the worldview (gets executed when the program starts)
func init() {
	WorldViewMutex.Lock()
	WorldView.Role = utils.SLAVE
	defer WorldViewMutex.Unlock()

	WorldView = utils.WorldView{}
	for i := 0; i < utils.N_ELEVATORS; i++ {
		requests := make([][]bool, utils.N_FLOORS)
		for j := range requests {
			requests[j] = make([]bool, utils.N_BUTTONS)
		}

		WorldView.Elevators[i] = utils.Elevator{
			Floor:      0,
			Dirn:       0,
			Behaviour:  0,
			Requests:   requests,
			Obstructed: false,
			MotorStopped: false,
		}
	}
}

func StartManager(elevatorID int, portNumber int) {

	// Start elevator I/O with port number
	var portNumberString string = strconv.Itoa(portNumber)
	elevio.Init("localhost:"+portNumberString, utils.N_FLOORS)

	// Initializing WorldView
	WorldView.ElevatorID = elevatorID
	WorldView.Alive[elevatorID] = true

	// The elevator sends updates through this channel to update the world view
	elevatorCh := make(chan utils.Elevator)

	// The eleavtor sends button presses through this channel
	btnCh := make(chan elevio.ButtonEvent)

	// Starting the elevator
	go elevatorLogic.StartElevator(btnCh, elevatorCh)

	// Channel to send cab calls from the button receiver to the button sender
	btnCabChan := make(chan elevio.ButtonEvent)

	// Channel the elevator uses to send orders to the master.go file (when the elevator is MASTER)
	masterSendChan := make(chan utils.ButtonMessage)

	// Channel the elevator uses to receive orders from the master.go file (when the elevator is MASTER)
	masterReceiveChan := make(chan utils.ButtonMessage)

	// When a button press is assigned to this elevator its sent to this function
	go SendButtonsToElevator(btnCh, btnCabChan, masterSendChan)

	// When an elevator dies, its calls are reassigned through this channel
	btnReassignChan := make(chan utils.ButtonMessage)

	// WorldView update function
	go elevatorListener(elevatorCh, btnReassignChan)

	// When a button is pressed on this elevator its sent to this function
	go ReceiveButtonsFromElevator(btnReassignChan, btnCabChan, masterReceiveChan)

	// World view listener
	go bcastListener(btnReassignChan, masterReceiveChan, masterSendChan)

	// Starting the periodic world view sender
	go bcastSender()

	// Wait
	select {}

}

// This function receives updates relative to the elevator from the elevator itself and updates the worldview
// If the elevator is obstructed or motor blocked, the calls are reassigned
func elevatorListener(elevatorCh chan utils.Elevator, btnReassignChan chan utils.ButtonMessage) {

	for {
		select {
		// When we receive an update we update the world view
		case e := <-elevatorCh:
			WorldViewMutex.Lock()
			if (e.MotorStopped != WorldView.Elevators[WorldView.ElevatorID].MotorStopped){
				fmt.Println("Motor stopped?", e.MotorStopped)
			}
			WorldView.Elevators[WorldView.ElevatorID] = e
			WorldViewMutex.Unlock()
			if e.Obstructed {

				// If the elevator is obstructed we wait for 500 ms so that the master can receive
				// the update and not assign the calls to this elevator

				time.Sleep(500 * time.Millisecond)

				// After waiting we reassign all the hall requests
				for floor := 0; floor < utils.N_FLOORS; floor++ {
					for btn := 0; btn < utils.N_BUTTONS-1; btn++ {
						if e.Requests[floor][btn] {
							btnReassignChan <- utils.ButtonMessage{
								ButtonEvent: elevio.ButtonEvent{
									Floor: floor,
									Button: elevio.ButtonType(btn),
								},
								ElevatorID: WorldView.ElevatorID,
							}
						}
					}
				}
			} else if e.MotorStopped {

				// Same as the obstruction, we wait for the master to receive the update and
				// then reassign the hall calls
				time.Sleep(500 * time.Millisecond)
				for floor := 0; floor < utils.N_FLOORS; floor++ {
					for btn := 0; btn < utils.N_BUTTONS-1; btn++ {
						if e.Requests[floor][btn] {
							btnReassignChan <- utils.ButtonMessage{
								ButtonEvent: elevio.ButtonEvent{
									Floor: floor,
									Button: elevio.ButtonType(btn),
								},
								ElevatorID: WorldView.ElevatorID,
							}
						}
					}
				}
			}

			// We update the lights when our elevator updates
			UpdateLights()
		}
	}
}

// This function listens to the other elevator world views and does a bunch of things
func bcastListener(btnReassignChan chan utils.ButtonMessage, masterReceiveChan chan utils.ButtonMessage, masterSendChan chan utils.ButtonMessage) {

	// When an elevator dies, we save the calls here untile they are reassigned
	var deadCabCalls [utils.N_ELEVATORS][utils.N_FLOORS]bool

	// Channel to receive other elevators world view
	receiveChan := make(chan utils.WorldView)

	// Broadcast world view receive function
	go bcast.Receiver(utils.WorldViewPort, receiveChan)

	// Here we keep which elevator broadcasts we have received so that we can tell which one is alive (or dead)
	var received [utils.N_ELEVATORS]bool

	// This channel is used to stop the "master functions" when an elevator demotes from master
	quitChan := make(chan bool)

	for {

		// Set master and backup as not found
		masterFound := false
		backupFound := false

		// Set received as false
		for i := 0; i < utils.N_ELEVATORS; i++ {
			received[i] = false
		}

		// Listen to broadcasts for 1 second

		timeout := time.After(1 * time.Second)
		done := false
		for !done {

			select {

			// When we receive a world view:
			case wv := <-receiveChan:

				// We update the other elevators
				for el := 0; el < utils.N_ELEVATORS; el++ {
					if el != WorldView.ElevatorID {
						WorldViewMutex.Lock()

						// If the elevator changed we update the lights
						if differentElevator(WorldView.Elevators[el], wv.Elevators[el]) {
							WorldView.Elevators[el] = wv.Elevators[el]
							WorldViewMutex.Unlock()
							UpdateLights()
						} else {
							WorldViewMutex.Unlock()
						}
					}

				}

				// If we are master or backup and theres another one with lower id, we must became slaves

				if wv.Role == WorldView.Role && wv.ElevatorID < WorldView.ElevatorID {

					if WorldView.Role == utils.MASTER {

						WorldView.Role = utils.SLAVE
						fmt.Println("MASTER going back to SLAVE")

						// If we are master we stop doing "master stuff"
						quitChan <- true

					} else if WorldView.Role == utils.BACKUP {

						WorldView.Role = utils.SLAVE
						fmt.Println("BACKUP going back to SLAVE")

					}

				}

				// Check if we received master or backup signals

				if wv.Role == utils.MASTER {
					masterFound = true
				}
				if wv.Role == utils.BACKUP {
					backupFound = true
				}

				// Set the received as true when receiving signal
				received[wv.ElevatorID] = true

			case <-timeout:
				// Exit the select after 1 second
				done = true
			}

			
		}

		// After listening for 1 second, we update which elevators are alive

		for el := 0; el < utils.N_ELEVATORS; el++ {
			WorldViewMutex.Lock()

			// If alive != received, the elevator died or came back
			if WorldView.Alive[el] != received[el] {

				WorldView.Alive[el] = received[el]

				if WorldView.Alive[el] == true {

					fmt.Printf("Elevator %d now alive\n", el)

					// If an elevator comes back alive, we must reassign its cab calls
					for floor := 0; floor < utils.N_FLOORS; floor++ {

						// We basically simulate pressing all the cab calls button the elevator had
						if deadCabCalls[el][floor] {

							fmt.Println("Reassinging ", floor, elevio.BT_Cab)
							deadCabCalls[el][floor] = false

							btnMsg := utils.ButtonMessage{
								ButtonEvent: elevio.ButtonEvent{
									Floor:  floor,
									Button: elevio.BT_Cab,
								},
								ElevatorID: el,
							}

							WorldViewMutex.Unlock()
							btnReassignChan <- btnMsg
							WorldViewMutex.Lock()
						}
					}
				} else {

					fmt.Printf("Elevator %d now dead\n", el)

					// If an elevator dies, we must reassign his hall calls
					for floor := 0; floor < utils.N_FLOORS; floor++ {
						for btn := 0; btn < utils.N_BUTTONS-1; btn++ {

							// We basically simulate pressing all the hall calls button the elevator had and the master will assign to the remaining elevators
							if WorldView.Elevators[el].Requests[floor][btn] {

								fmt.Println("Reassinging ", floor, btn)

								btnMsg := utils.ButtonMessage{
									ButtonEvent: elevio.ButtonEvent{
										Floor:  floor,
										Button: elevio.ButtonType(btn),
									},
									ElevatorID: WorldView.ElevatorID,
								}
								WorldViewMutex.Unlock()
								btnReassignChan <- btnMsg
								WorldViewMutex.Lock()

							}
						}

						// We must also save his cab calls so that they can be reassigned to it when it comes back

						if WorldView.Elevators[el].Requests[floor][elevio.BT_Cab] {
							deadCabCalls[el][floor] = true
						}

					}
				}
			}
			WorldViewMutex.Unlock()
		}
		WorldViewMutex.Lock()

		// If we are a backup and theres no master become master
		if WorldView.Role == utils.BACKUP && !masterFound {
			fmt.Println("No MASTER found, BACKUP becoming MASTER")
			WorldView.Role = utils.MASTER

			// And we start doing "master stuff"
			go RunMaster(quitChan, masterReceiveChan, masterSendChan)

			// If we are a slave and theres no backup become backup
		} else if WorldView.Role == utils.SLAVE && !backupFound {
			fmt.Println("No BACKUP found, SLAVE becoming BACKUP")
			WorldView.Role = utils.BACKUP
		}

		WorldViewMutex.Unlock()

	}
}

// This function is used to check if an elevator changed from what we have in the world view
func differentElevator(el1 utils.Elevator, el2 utils.Elevator) bool {
	if el1.Floor != el2.Floor || el1.Dirn != el2.Dirn || el1.Behaviour != el2.Behaviour || el1.Obstructed != el2.Obstructed || el1.MotorStopped != el2.MotorStopped {
		return true
	}
	for floor := 0; floor < utils.N_FLOORS; floor++ {
		for btn := 0; btn < utils.N_BUTTONS; btn++ {
			if el1.Requests[floor][btn] != el2.Requests[floor][btn] {
				return true
			}
		}
	}
	return false
}

// This function broadcasts the world view every 50 ms
func bcastSender() {

	// This channel is to send the world view
	sendChan := make(chan utils.WorldView)

	go bcast.Transmitter(utils.WorldViewPort, sendChan)
	for {
		WorldViewMutex.Lock()
		sendChan <- WorldView
		WorldViewMutex.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
}
