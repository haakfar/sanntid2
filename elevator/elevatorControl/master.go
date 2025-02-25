package elevatorControl

import (
	"Config/config"
	"Network-go/network/bcast"
	"Driver-go/elevio"
	"fmt"
	"math/rand"
	"sync"
)

var alive [config.N_ELEVATORS] bool
var aliveMu sync.Mutex

func RunMaster(updateChan chan config.ElevatorUpdate){
	receiveChan := make(chan config.ButtonMessage)
	sendChan := make(chan config.ButtonMessage)
	go bcast.Receiver(config.Port, receiveChan)
	go bcast.Transmitter(config.Port, sendChan)
	go detectElevators(updateChan)

	for {
		select {
		case btnMsg := <- receiveChan:
			if btnMsg.MessageType == config.RECEIVED {
				if btnMsg.ButtonEvent.Button == elevio.BT_Cab {
					btnMsg.MessageType = config.SENT
					sendChan <- btnMsg
					fmt.Println("Assigned cab call to", btnMsg.ElevatorID)
				} else {
					// for now its assigned randomly
					btnMsg.ElevatorID = rand.Intn(3)
					aliveMu.Lock()
					for alive[btnMsg.ElevatorID] == false {
						btnMsg.ElevatorID = rand.Intn(3)
					}
					aliveMu.Unlock()
					btnMsg.MessageType = config.SENT
					sendChan <- btnMsg
					fmt.Println("Assigned hall call to", btnMsg.ElevatorID)
				}
			}
		}
	}
}

func assign(){

}

func detectElevators(updateChan chan config.ElevatorUpdate){
	for {
		select {
		case update := <- updateChan:
			aliveMu.Lock()
			alive[update.ElevatorID] = update.Alive
			aliveMu.Unlock()
		}
	}
}