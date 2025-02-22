package main

import (
	"flag"
	"Elevator/elevatorControl"
	//"fmt"
)


func main() {
	elevatorIDPtr := flag.Int("id", 0, "ID of the elevator")
	flag.Parse()

	elevatorID := *elevatorIDPtr

	go elevatorControl.StartManager(elevatorID)

	select {}

}
