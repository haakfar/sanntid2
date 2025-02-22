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

var mu sync.Mutex
var worldView config.WorldView

func StartManager(elevatorID int){

    elevio.Init("localhost:15657", config.N_FLOORS)

	//placeholder
	//this must be a receiver from the master
	btnCh := make(chan elevio.ButtonEvent)

	elevatorCh := make(chan config.Elevator)

	go elevatorLogic.StartElevator(btnCh, elevatorCh)

	worldView.SentBy = elevatorID
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
			if btnMsg.MessageType == config.SENT && btnMsg.ElevatorID == worldView.SentBy {
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
				ElevatorID: worldView.SentBy,
				MessageType: config.RECEIVED,
			}
		}
	}
}

func elevatorListener(elevatorCh chan config.Elevator){


	for {
		select {
		case e := <- elevatorCh:
			mu.Lock()
			worldView.Elevators[worldView.SentBy] = e
			mu.Unlock()
		}
	}
}

func bcastListener(){
	receiveChan := make(chan config.WorldView)
	go bcast.Receiver(config.Port, receiveChan)
	for {
		start := time.Now()
		masterFound := false
		backupFound := false

		for time.Since(start) < time.Second {
			select {
			case wv := <- receiveChan:

				for i:=0; i<config.N_ELEVATORS; i++ {
					if i!=worldView.SentBy {
						mu.Lock()
						worldView.Elevators[i]=wv.Elevators[i]
						mu.Unlock()
					}
				}



				if wv.Role == config.MASTER {
					masterFound = true
				}
				if wv.Role == config.BACKUP {
					backupFound = true
				}
			}
		}
		mu.Lock()
		if worldView.Role == config.BACKUP && !masterFound {
			fmt.Println("No MASTER found, BACKUP becoming MASTER")
			worldView.Role = config.MASTER
			go RunMaster()
		} else if worldView.Role == config.SLAVE && !backupFound {
			fmt.Println("No BACKUP found, SLAVE becoming BACKUP")
			worldView.Role = config.BACKUP
		}
		mu.Unlock()
	}
}

func bcastSender(){
	sendChan := make(chan config.WorldView)
	go bcast.Transmitter(config.Port, sendChan)
	for {
		mu.Lock()
		sendChan <- worldView
		mu.Unlock()
		time.Sleep(200 * time.Millisecond)
	}
}

