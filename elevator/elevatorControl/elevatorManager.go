package elevatorControl

import (
	"Config/config"
	"Driver-go/elevio"
	"Elevator/elevatorLogic"
	"Network-go/network/bcast"
	"fmt"
	"strconv"
	"sync"
	"time"
)

var WorldViewMutex sync.Mutex
var WorldView config.WorldView

// Function to initialize the worldview (gets executed when the program starts)
func init() {
	WorldViewMutex.Lock()
	WorldView.Role = config.SLAVE
	defer WorldViewMutex.Unlock()

	WorldView = config.WorldView{}
	for i := 0; i < config.N_ELEVATORS; i++ {
		requests := make([][]bool, config.N_FLOORS)
		for j := range requests {
			requests[j] = make([]bool, config.N_BUTTONS)
		}

		WorldView.Elevators[i] = config.Elevator{
			Floor:      0,
			Dirn:       0,
			Behaviour:  0,
			Requests:   requests,
			Obstructed: false,
		}
	}
}

func StartManager(elevatorID int, portNumber int) {

	// Start elevator I/O with port number
	var portNumberString string = strconv.Itoa(portNumber)
	elevio.Init("localhost:"+portNumberString, config.N_FLOORS)

	// Initializing WorldView
	WorldView.ElevatorID = elevatorID
	WorldView.Alive[elevatorID] = true

	// The elevator sends updates through this channel to update the world view
	elevatorCh := make(chan config.Elevator)

	// The eleavtor sends button presses through this channel
	btnCh := make(chan elevio.ButtonEvent)

	// Starting the elevator
	go elevatorLogic.StartElevator(btnCh, elevatorCh)

	// WorldView update function
	go elevatorListener(elevatorCh)

	// Button listen function (listens from the master and sends to elevator)
	go ButtonListener(btnCh)

	// When an elevator dies, its calls are reassigned through this channel
	btnReassignChan := make(chan config.ButtonMessage)

	// Button send function (listens from the elevator (and the reassigned calls) and sends to master)
	go ButtonSender(btnReassignChan)

	// World view listener
	go bcastListener(btnReassignChan)

	// Starting the periodic world view sender
	go bcastSender()

	// Wait
	select {}

}

// This function receives updates relative to the elevator from the elevator itself and updates the worldview
func elevatorListener(elevatorCh chan config.Elevator) {

	for {
		select {
		// When we receive an update we update the world view
		case e := <-elevatorCh:
			WorldViewMutex.Lock()
			WorldView.Elevators[WorldView.ElevatorID] = e
			WorldViewMutex.Unlock()

			// We update the lights when our elevator updates
			UpdateLights()
		}
	}
}

// This function listens to the other elevator world views and does a bunch of things
func bcastListener(btnReassignChan chan config.ButtonMessage) {

	// When an elevator dies, we save the calls here untile they are reassigned
	var deadCabCalls [config.N_ELEVATORS][config.N_FLOORS]bool

	// Channel to receive other elevators world view
	receiveChan := make(chan config.WorldView)

	// Broadcast world view receive function
	go bcast.Receiver(config.WorldViewPort, receiveChan)

	// Here we keep which elevator broadcasts we have received so that we can tell which one is alive (or dead)
	var received [config.N_ELEVATORS]bool

	// This channel is used to stop the "master functions" when an elevator demotes from master
	quitChan := make(chan bool)

	for {

		//fmt.Println("Listening")

		// Start the timer
		start := time.Now()

		// Set master and backup as not found
		masterFound := false
		backupFound := false

		// Set received as false
		for i := 0; i < config.N_ELEVATORS; i++ {
			received[i] = false
		}

		// Listen to broadcasts for 1 second
		for time.Since(start) < time.Second {

			select {

			// When we receive a world view:
			case wv := <-receiveChan:

				// We update the other elevators
				for i := 0; i < config.N_ELEVATORS; i++ {
					if i != WorldView.ElevatorID {
						WorldViewMutex.Lock()

						// If the elevator changed we update the lights
						if differentElevator(WorldView.Elevators[i], wv.Elevators[i]) {
							WorldView.Elevators[i] = wv.Elevators[i]
							WorldViewMutex.Unlock()
							UpdateLights()
						} else {
							WorldViewMutex.Unlock()
						}
					}

				}

				// If we are master or backup and theres another one with lower id, we must became slaves

				if wv.Role == WorldView.Role && wv.ElevatorID < WorldView.ElevatorID {

					if WorldView.Role == config.MASTER {

						WorldView.Role = config.SLAVE
						fmt.Println("MASTER going back to SLAVE")

						// If we are master we stop doing "master stuff"
						quitChan <- true

					} else if WorldView.Role == config.BACKUP {

						WorldView.Role = config.SLAVE
						fmt.Println("BACKUP going back to SLAVE")

					}

				}

				// Check if we received master or backup signals

				if wv.Role == config.MASTER {
					masterFound = true
				}
				if wv.Role == config.BACKUP {
					backupFound = true
				}

				// Set the received as true when receiving signal
				received[wv.ElevatorID] = true

			}
		}

		// After listening for 1 second, we update which elevators are alive

		for i := 0; i < config.N_ELEVATORS; i++ {
			WorldViewMutex.Lock()

			// If alive != received, the elevator died or came back
			if WorldView.Alive[i] != received[i] {

				WorldView.Alive[i] = received[i]

				if WorldView.Alive[i] == true {

					fmt.Printf("Elevator %d now alive\n", i)

					// If an elevator comes back alive, we must reassign its cab calls
					for floor := 0; floor < config.N_FLOORS; floor++ {

						// We basically simulate pressing all the cab calls button the elevator had
						if deadCabCalls[i][floor] {

							fmt.Println("Reassinging ", floor, elevio.BT_Cab)
							deadCabCalls[i][floor] = false

							btnMsg := config.ButtonMessage{
								ButtonEvent: elevio.ButtonEvent{
									Floor:  floor,
									Button: elevio.BT_Cab,
								},
								ElevatorID: i,
							}

							WorldViewMutex.Unlock()
							btnReassignChan <- btnMsg
							WorldViewMutex.Lock()
						}
					}
				} else {

					fmt.Printf("Elevator %d now dead\n", i)

					// If an elevator dies, we must reassign his hall calls
					for floor := 0; floor < config.N_FLOORS; floor++ {
						for btn := 0; btn < config.N_BUTTONS-1; btn++ {

							// We basically simulate pressing all the hall calls button the elevator had and the master will assign to the remaining elevators
							if WorldView.Elevators[i].Requests[floor][btn] {

								fmt.Println("Reassinging ", floor, btn)

								btnMsg := config.ButtonMessage{
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

						if WorldView.Elevators[i].Requests[floor][elevio.BT_Cab] {
							deadCabCalls[i][floor] = true
						}

					}
				}
			}
			WorldViewMutex.Unlock()
		}
		WorldViewMutex.Lock()

		// If we are a backup and theres no master become master
		if WorldView.Role == config.BACKUP && !masterFound {
			fmt.Println("No MASTER found, BACKUP becoming MASTER")
			WorldView.Role = config.MASTER

			// And we start doing "master stuff"
			go RunMaster(quitChan)

			// If we are a slave and theres no backup become backup
		} else if WorldView.Role == config.SLAVE && !backupFound {
			fmt.Println("No BACKUP found, SLAVE becoming BACKUP")
			WorldView.Role = config.BACKUP
		}

		WorldViewMutex.Unlock()

	}
}

// This function checks if an elevator changed from what we have in the world view
func differentElevator(el1 config.Elevator, el2 config.Elevator) bool {
	if el1.Floor != el2.Floor || el1.Dirn != el2.Dirn || el1.Behaviour != el2.Behaviour {
		return true
	}
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS; btn++ {
			if el1.Requests[floor][btn] != el2.Requests[floor][btn] {
				return true
			}
		}
	}
	return false
}

// This function broadcasts the world view every 200 ms
func bcastSender() {

	// This channel is to send the world view
	sendChan := make(chan config.WorldView)

	go bcast.Transmitter(config.WorldViewPort, sendChan)
	for {
		WorldViewMutex.Lock()
		sendChan <- WorldView
		WorldViewMutex.Unlock()
		time.Sleep(200 * time.Millisecond)
	}
}
