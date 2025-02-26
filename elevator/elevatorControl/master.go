package elevatorControl

import (
	"Config/config"
	"Network-go/network/bcast"
	"Driver-go/elevio"
	"fmt"
	"math/rand"
	"sync"
)

var active [config.N_ELEVATORS] bool
var activeMu sync.Mutex

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
					activeMu.Lock()
					for active[btnMsg.ElevatorID] == false {
						btnMsg.ElevatorID = rand.Intn(3)
					}
					activeMu.Unlock()
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
			activeMu.Lock()
			active[update.ElevatorID] = update.Alive
			activeMu.Unlock()
		}
	}
}