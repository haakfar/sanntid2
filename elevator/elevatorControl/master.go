package elevatorControl

import (
	"Config/config"
	"Network-go/network/bcast"
	"Driver-go/elevio"
	"fmt"
	"math/rand"
)

func RunMaster(){
	receiveChan := make(chan config.ButtonMessage)
	sendChan := make(chan config.ButtonMessage)
	go bcast.Receiver(config.Port, receiveChan)
	go bcast.Transmitter(config.Port, sendChan)

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
					btnMsg.MessageType = config.SENT
					sendChan <- btnMsg
					fmt.Println("Assigned cab call to", btnMsg.ElevatorID)
				}
			}
		}
	}
}

func assign(){

}