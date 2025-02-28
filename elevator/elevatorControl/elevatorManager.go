package elevatorControl

import (
	"fmt"
	"Elevator/elevatorLogic"
	"Network-go/network/bcast"
	"sync"
	"Config/config"
	"time"
	"Driver-go/elevio"
)

var	wvMutex sync.Mutex
var worldView config.WorldView
var alive [config.N_ELEVATORS] bool
var aliveMutex sync.Mutex

func StartManager(elevatorID int){

	// start elevator
    elevio.Init("localhost:15657", config.N_FLOORS)

	
	elevatorCh := make(chan config.Elevator)

	btnCh := make(chan elevio.ButtonEvent)

	go elevatorLogic.StartElevator(btnCh, elevatorCh)

	worldView.ElevatorID = elevatorID
	worldView.Role = config.SLAVE

	go elevatorListener(elevatorCh)
	go bcastSender()
	go bcastListener()
	go ButtonSender()
	go ButtonListener(btnCh)

	select {}

}

func elevatorListener(elevatorCh chan config.Elevator){

	// updates the worldview for the elevator
	for {
		select {
		case e := <- elevatorCh:
			wvMutex.Lock()
			worldView.Elevators[worldView.ElevatorID] = e
			wvMutex.Unlock()
		}
	}
}

func bcastListener(){

	// receives worldviews from other elevators
	receiveChan := make(chan config.WorldView)
	elUpdateChan := make(chan config.ElevatorUpdate)
	go bcast.Receiver(config.Port, receiveChan)

	var received [config.N_ELEVATORS] bool

	worldViewChan := make (chan config.WorldView)
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
					if i!=worldView.ElevatorID {
						wvMutex.Lock()
						worldView.Elevators[i]=wv.Elevators[i]
						wvMutex.Unlock()
					}

				}

				// if we are master or backup and theres another one with lower id, we became slaves

				if wv.Role == worldView.Role && wv.ElevatorID < worldView.ElevatorID{

					// if we are master we stop the master.go file
					if worldView.Role == config.MASTER {
						quitChan <- true
					}
					worldView.Role = config.SLAVE
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
			aliveMutex.Lock()
			if alive[i] != received[i] {
				alive[i] = received[i] 
				if alive[i] == true {
					fmt.Printf("Elevator %d now alive\n", i)
				} else {
					fmt.Printf("Elevator %d now dead\n", i)
				}
				wvMutex.Lock()
				if (worldView.Role == config.MASTER){
					elUpdateChan <- config.ElevatorUpdate{i,alive[i]}
				}
				wvMutex.Unlock()
			}
			aliveMutex.Unlock()
		}
		wvMutex.Lock()

		// if I'm backup and theres no master become master
		if worldView.Role == config.BACKUP && !masterFound {
			fmt.Println("No MASTER found, BACKUP becoming MASTER")
			worldView.Role = config.MASTER

			// strt master file 
			go RunMaster(elUpdateChan, worldViewChan, quitChan)

			// update elevators status for master
			for i := 0 ; i<config.N_ELEVATORS;i++{
				aliveMutex.Lock()
				elUpdateChan <- config.ElevatorUpdate{i,alive[i]}
				aliveMutex.Unlock()
			}

			// if I'm slave and theres no backup become backup
		} else if worldView.Role == config.SLAVE && !backupFound {
			fmt.Println("No BACKUP found, SLAVE becoming BACKUP")
			worldView.Role = config.BACKUP
		}

		// if I'm master send worldView to master.go
		if worldView.Role == config.MASTER {
			worldViewChan <- worldView
		}

		wvMutex.Unlock()

		// update lights every second
		updateLights()

	}
}

func updateLights(){
	for floor := 0; floor < config.N_FLOORS; floor++ {

		// if theres a hall call on floor light up
		elevio.SetButtonLamp(elevio.BT_HallUp, floor, callOnFloor(floor, elevio.BT_HallUp))
		elevio.SetButtonLamp(elevio.BT_HallDown, floor, callOnFloor(floor, elevio.BT_HallDown))

		// light cab call 
		wvMutex.Lock()
		elevio.SetButtonLamp(elevio.BT_Cab, floor, worldView.Elevators[worldView.ElevatorID].Requests[floor][elevio.BT_Cab])
		wvMutex.Unlock()
	}
}

func callOnFloor(floor int, call elevio.ButtonType) bool {
	// for every elevator alive check if theres a call on floor

	for el := 0; el < config.N_ELEVATORS; el++{
		aliveMutex.Lock()
		if !alive[el] {
			aliveMutex.Unlock()
			continue
		}
		aliveMutex.Unlock()
		wvMutex.Lock()
		if worldView.Elevators[el].Requests[floor][call] {
			wvMutex.Unlock()
			return true
		}
		wvMutex.Unlock()
	}
	return false
}


func bcastSender(){

	// broadcast worldView every 200ms 
	sendChan := make(chan config.WorldView)
	go bcast.Transmitter(config.Port, sendChan)
	for {
		wvMutex.Lock()
		sendChan <- worldView
		wvMutex.Unlock()
		time.Sleep(200 * time.Millisecond)
	}
}

