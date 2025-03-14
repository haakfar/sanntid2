package elevatorControl

import (
	"fmt"
	"Elevator/elevatorLogic"
	"Network-go/network/bcast"
	"sync"
	"Config/config"
	"time"
	"Driver-go/elevio"
	"strconv"
)

var	WorldViewMutex sync.Mutex
var WorldView config.WorldView

func StartManager(elevatorID int, portNumber int) {
	var portNumberString string = strconv.Itoa(portNumber)
	// start elevator
	elevio.Init("localhost:"+portNumberString, config.N_FLOORS)

	
	elevatorCh := make(chan config.Elevator)

	btnCh := make(chan elevio.ButtonEvent)

	go elevatorLogic.StartElevator(btnCh, elevatorCh)

	btnReassignChan := make(chan config.ButtonMessage)

	WorldView.ElevatorID = elevatorID
	WorldView.Role = config.SLAVE
	WorldView.Alive[elevatorID]=true    
	WorldView.Elevators[elevatorID] = elevatorLogic.GetElevator()

	go elevatorListener(elevatorCh)
	go bcastSender()
	go bcastListener(btnReassignChan)
	go ButtonSender(btnReassignChan)
	go ButtonListener(btnCh)

	select {}

}

func elevatorListener(elevatorCh chan config.Elevator){

	// updates the worldview for the elevator
	for {
		select {
		case e := <- elevatorCh:
			WorldViewMutex.Lock()
			WorldView.Elevators[WorldView.ElevatorID] = e
			WorldViewMutex.Unlock()
		}
	}
}

func bcastListener(btnReassignChan chan config.ButtonMessage){

	var deadCabCalls [config.N_ELEVATORS][config.N_FLOORS]bool

	// receives worldviews from other elevators
	receiveChan := make(chan config.WorldView)
	//elUpdateChan := make(chan config.ElevatorUpdate)
	go bcast.Receiver(config.Port, receiveChan)

	var received [config.N_ELEVATORS] bool

	//worldViewChan := make (chan config.WorldView)
	quitChan := make(chan bool)
	for {

		start := time.Now()
		masterFound := false
		backupFound := false

		for i:=0; i< config.N_ELEVATORS; i++ {
			received[i]= false
		}


		for time.Since(start) < time.Second {
			// listen to broadcasts for 1 second
			select {
			case wv := <- receiveChan:

				// updates other elevators
				for i:=0; i<config.N_ELEVATORS; i++ {
					if i!=WorldView.ElevatorID{
						WorldViewMutex.Lock()
						WorldView.Elevators[i]=wv.Elevators[i]
						WorldViewMutex.Unlock()
					}

				}

				// if we are master or backup and theres another one with lower id, we became slaves

				if wv.Role == WorldView.Role && wv.ElevatorID < WorldView.ElevatorID{

					// if we are master we stop the master.go file
					if WorldView.Role == config.MASTER {
						quitChan <- true
					}
					WorldView.Role = config.SLAVE
					fmt.Println("Going back to SLAVE")
				}

				// check if master/backup are alive

				if wv.Role == config.MASTER {
					masterFound = true
				}
				if wv.Role == config.BACKUP {
					backupFound = true
				}


				received[wv.ElevatorID] = true

			}
		}

		// update which elevators are alive
		for i := 0 ; i< config.N_ELEVATORS; i++{
			WorldViewMutex.Lock()
			if WorldView.Alive[i] != received[i] {
				WorldView.Alive[i] = received[i] 
				if WorldView.Alive[i] == true {
					fmt.Printf("Elevator %d now alive\n", i)
					// if an elevator comes back alive, we must reassign its cab calls
					for floor := 0; floor < config.N_FLOORS; floor ++ {
						if deadCabCalls[i][floor] {
							fmt.Println("Reassinging ", floor, elevio.BT_Cab)
							deadCabCalls[i][floor] = false
							btnReassignChan <- config.ButtonMessage{
								ButtonEvent: elevio.ButtonEvent{
									Floor: floor,
									Button: elevio.BT_Cab,
								},
								ElevatorID: i,
								MessageType: config.RECEIVED,
							}
						}
					}	
				} else {
					fmt.Printf("Elevator %d now dead\n", i)
					// if an elevator dies, we must reassign his hall calls 
					for floor := 0; floor < config.N_FLOORS; floor ++ {
						for btn := 0; btn < config.N_BUTTONS-1; btn ++ {
							if WorldView.Elevators[i].Requests[floor][btn] {
								fmt.Println("Reassinging ", floor, btn)
								btnReassignChan <- config.ButtonMessage{
									ButtonEvent: elevio.ButtonEvent{ // Corretto l'inizializzazione di ButtonEvent
										Floor:  floor,
										Button: elevio.ButtonType(btn),
									},
									ElevatorID:  WorldView.ElevatorID, // Corretto l'uso del segno `=`, deve essere `:`
									MessageType: config.RECEIVED,
								}
							}
						}

						// we must also save his cab calls so that they can be reassigned to it when it comes back

						if WorldView.Elevators[i].Requests[floor][elevio.BT_Cab] {
							deadCabCalls[i][floor]=true
						}


					}
				}
			}
			WorldViewMutex.Unlock()
		}
		WorldViewMutex.Lock()

		// if I'm backup and theres no master become master
		if WorldView.Role == config.BACKUP && !masterFound {
			fmt.Println("No MASTER found, BACKUP becoming MASTER")
			WorldView.Role = config.MASTER

			// strt master file 
			go RunMaster(quitChan)

			// if I'm slave and theres no backup become backup
		} else if WorldView.Role == config.SLAVE && !backupFound {
			fmt.Println("No BACKUP found, SLAVE becoming BACKUP")
			WorldView.Role = config.BACKUP
		}

		WorldViewMutex.Unlock()

		// update lights every second
		UpdateLights()

	}
}

func callOnFloor(floor int, call elevio.ButtonType) bool {
	// for every elevator alive check if theres a call on floor

	for el := 0; el < config.N_ELEVATORS; el++{
		//fmt.Println(el)
		WorldViewMutex.Lock()
		//fmt.Println("A")
		if !WorldView.Alive[el] {
			WorldViewMutex.Unlock()
			continue
		}
		//fmt.Println("B")
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


func bcastSender(){

	// broadcast WorldView every 200ms 
	sendChan := make(chan config.WorldView)
	go bcast.Transmitter(config.Port, sendChan)
	for {
		WorldViewMutex.Lock()
		sendChan <- WorldView
		WorldViewMutex.Unlock()
		time.Sleep(200 * time.Millisecond)
	}
}

