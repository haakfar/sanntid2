package main

import (
	"flag"
	"Elevator/elevatorControl"
	"fmt"
)


func main() {
	elevatorIDPtr := flag.Int("id", 0, "ID of the elevator")
	//flag.Parse()


	portNumberPtr := flag.Int("port", 15657, "Port Number of the elevator")
	flag.Parse()

	elevatorID := *elevatorIDPtr
	portNumber := *portNumberPtr
	fmt.Println("Elevator ", elevatorID)

	go elevatorControl.StartManager(elevatorID, portNumber)

	select {}

}
