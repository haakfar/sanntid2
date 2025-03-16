package main

import (
	"flag"
	"Elevator/elevatorControl"
	"fmt"
)


func main() {

	// Reading elevator ID and port number
	elevatorIDPtr := flag.Int("id", 0, "ID of the elevator")
	portNumberPtr := flag.Int("port", 15657, "Port Number of the elevator")

	flag.Parse()

	elevatorID := *elevatorIDPtr
	portNumber := *portNumberPtr

	fmt.Println("Elevator", elevatorID)

	// Starting the elvator manager
	go elevatorControl.StartManager(elevatorID, portNumber)

	// Wait
	select {}

}
