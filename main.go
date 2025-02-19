package main

import (
	"Network-go/network/bcast"
	"flag"
	"time"
	"Config/config"
	"Elevator/elevator"
	"Driver-go/elevio"
	"fmt"
)



var currentRole = config.SLAVE
var currentDirection config.Direction = config.STAND_STILL
var lastFloor = -1
var elevatorID int

var upLists = [config.N_ELEVATORS][config.N_FLOORS]int{}
var downLists = [config.N_ELEVATORS][config.N_FLOORS]int{}
var cabLists = [config.N_ELEVATORS][config.N_FLOORS]int{}

func main() {
	elevatorIDPtr := flag.Int("id", 0, "ID of the elevator")
	flag.Parse()

	elevatorID = *elevatorIDPtr

	port := 9000

    elevio.Init("localhost:15657", config.N_FLOORS)

	sendChan := make(chan config.Message)
	receiveChan := make(chan config.Message)
	roleChan := make(chan config.Role)


	btnChan := make(chan elevio.ButtonEvent)

	go elevio.PollButtons(btnChan)

	go bcast.Transmitter(port, sendChan)
	go bcast.Receiver(port, receiveChan)

	go elevator.DetermineRole(receiveChan, elevatorID, roleChan)

	// Send every 200ms
	go func() {
		for {
			sendChan <- config.Message{
				ID:        elevatorID,
				Role:      currentRole,
				Direction: currentDirection,
				Floor:     lastFloor,
				UpLists:    upLists,
				DownLists:  downLists,
				CabLists:  cabLists,
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()

	go func(){
		for {
			select {
			case role := <- roleChan:
				currentRole = role;
			}
		}
	}()

	go func(){
		for {
			select {
			case btnEvent := <- btnChan:
				fmt.Println(btnEvent)
				switch btnEvent.Button{
				case elevio.BT_HallUp:
					upLists[elevatorID][btnEvent.Floor] = 1
				case elevio.BT_HallDown:
					downLists[elevatorID][btnEvent.Floor] = 1
				case elevio.BT_Cab:
					cabLists[elevatorID][btnEvent.Floor] = 1
				}
			}
		}
	}()
	

	select {}
}
