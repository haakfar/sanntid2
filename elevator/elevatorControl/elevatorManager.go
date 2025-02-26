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
	
    elevio.Init("localhost:15657", config.N_FLOORS)

	elevatorCh := make(chan config.Elevator)


	btnCh := make(chan elevio.ButtonEvent)

	go elevatorLogic.StartElevator(btnCh, elevatorCh)

	worldView.ElevatorID = elevatorID
	worldView.Role = config.SLAVE

	go elevatorListener(elevatorCh)
	go bcastSender()
	go bcastListener()
	go buttonSender()
	go buttonListener(btnCh)

	select {}

}

func buttonListener(btnCh chan elevio.ButtonEvent){

	// receives button broadcasts and sends them to the elevator
	receiveChan := make(chan config.ButtonMessage)
	go bcast.Receiver(config.Port,receiveChan)

	for {
		select {
		case btnMsg := <- receiveChan:
			// if its SENT then its sent by the master
			if btnMsg.MessageType == config.SENT && btnMsg.ElevatorID == worldView.ElevatorID {
				btnCh <- btnMsg.ButtonEvent
			}
		}
	}
}

func buttonSender(){

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
				ElevatorID: worldView.ElevatorID,
				MessageType: config.RECEIVED,
			}
		}
	}
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
	// 
	for {
		start := time.Now()
		masterFound := false
		backupFound := false

		for i:=0; i< config.N_ELEVATORS; i++ {
			received[i]= false
		}


		for time.Since(start) < time.Second {
			select {
			case wv := <- receiveChan:

				//fmt.Printf("Received: ")
				//fmt.Println(wv.ElevatorID)

				for i:=0; i<config.N_ELEVATORS; i++ {
					if i!=worldView.ElevatorID {
						wvMutex.Lock()
						worldView.Elevators[i]=wv.Elevators[i]
						wvMutex.Unlock()
					}

				}

				if wv.Role == worldView.Role && wv.ElevatorID < worldView.ElevatorID{
					if worldView.Role == config.MASTER {
						quitChan <- true
					}
					worldView.Role = config.SLAVE
					fmt.Println("Going back to SLAVE")
				}

				if wv.Role == config.MASTER {
					masterFound = true
				}
				if wv.Role == config.BACKUP {
					backupFound = true
				}


				received[wv.ElevatorID] = true

			}
		}
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
		if worldView.Role == config.BACKUP && !masterFound {
			fmt.Println("No MASTER found, BACKUP becoming MASTER")
			worldView.Role = config.MASTER
			go RunMaster(elUpdateChan, worldViewChan, quitChan)
			for i := 0 ; i<config.N_ELEVATORS;i++{
				aliveMutex.Lock()
				elUpdateChan <- config.ElevatorUpdate{i,alive[i]}
				aliveMutex.Unlock()
			}
		} else if worldView.Role == config.SLAVE && !backupFound {
			fmt.Println("No BACKUP found, SLAVE becoming BACKUP")
			worldView.Role = config.BACKUP
		}
		if worldView.Role == config.MASTER {
			worldViewChan <- worldView
		}
		wvMutex.Unlock()


		updateLights()

	}
}

func updateLights(){
	for floor := 0; floor < config.N_FLOORS; floor++ {

		elevio.SetButtonLamp(elevio.BT_HallUp, floor, callOnFloor(floor, elevio.BT_HallUp))
		elevio.SetButtonLamp(elevio.BT_HallDown, floor, callOnFloor(floor, elevio.BT_HallDown))
		wvMutex.Lock()
		elevio.SetButtonLamp(elevio.BT_Cab, floor, worldView.Elevators[worldView.ElevatorID].Requests[floor][elevio.BT_Cab])
		wvMutex.Unlock()
	}
}

func callOnFloor(floor int, call elevio.ButtonType) bool {
	//fmt.Println(len(worldView.Elevators))
	//fmt.Println(worldView.Elevators)
	for el := 0; el < config.N_ELEVATORS; el++{
		aliveMutex.Lock()
		if !alive[el] {
			aliveMutex.Unlock()
			continue
		}
		aliveMutex.Unlock()
		//fmt.Println(el)
		wvMutex.Lock()
		if worldView.Elevators[el].Requests[floor][call] {
			wvMutex.Unlock()
			return true
		}
		wvMutex.Unlock()
		//fmt.Println(el)
	}
	return false
}


func bcastSender(){
	sendChan := make(chan config.WorldView)
	go bcast.Transmitter(config.Port, sendChan)
	for {
		wvMutex.Lock()
		sendChan <- worldView
		wvMutex.Unlock()
		time.Sleep(200 * time.Millisecond)
	}
}

