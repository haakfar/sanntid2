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

	receiveChan := make(chan config.ButtonMessage)
	go bcast.Receiver(config.Port,receiveChan)

	for {
		select {
		case btnMsg := <- receiveChan:
			if btnMsg.MessageType == config.SENT && btnMsg.ElevatorID == worldView.ElevatorID {
				btnCh <- btnMsg.ButtonEvent
			}
		}
	}
}

func buttonSender(){
	sendChan := make(chan config.ButtonMessage)
	btnChan := make(chan elevio.ButtonEvent)
	go bcast.Transmitter(config.Port,sendChan)
	go elevio.PollButtons(btnChan)
	for {
		select {
		case btnEvent := <- btnChan:
			sendChan <- config.ButtonMessage{
				ButtonEvent: btnEvent,
				ElevatorID: worldView.ElevatorID,
				MessageType: config.RECEIVED,
			}
		}
	}
}

func elevatorListener(elevatorCh chan config.Elevator){


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
	receiveChan := make(chan config.WorldView)
	elUpdateChan := make(chan config.ElevatorUpdate)
	go bcast.Receiver(config.Port, receiveChan)

	var alive [config.N_ELEVATORS] bool

	var received [config.N_ELEVATORS] bool

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
		}
		wvMutex.Lock()
		if worldView.Role == config.BACKUP && !masterFound {
			fmt.Println("No MASTER found, BACKUP becoming MASTER")
			worldView.Role = config.MASTER
			go RunMaster(elUpdateChan)
			for i := 0 ; i<config.N_ELEVATORS;i++{
				elUpdateChan <- config.ElevatorUpdate{i,alive[i]}
			}
		} else if worldView.Role == config.SLAVE && !backupFound {
			fmt.Println("No BACKUP found, SLAVE becoming BACKUP")
			worldView.Role = config.BACKUP
		}
		wvMutex.Unlock()
	}
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

