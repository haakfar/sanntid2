package main

import (
	"Network-go/network/bcast"
	"fmt"
	"time"
)

const (
	STAND_STILL Direction = iota
	GOING_UP
	GOING_DOWN
)

const (
	SLAVE Role = iota
	BACKUP
	MASTER
)

const N_FLOORS = 4

type Direction int
type Role int

type Message struct {
	Role      Role      `json:"role"`
	Direction Direction `json:"direction"`
	Floor     int       `json:"floor"`
	UpList    []int     `json:"upList"`
	DownList  []int     `json:"downList"`
	CabList   []int     `json:"cabList"`
}

var currentRole Role = SLAVE
var currentDirection Direction = STAND_STILL
var lastFloor = -1

var upList = [N_FLOORS]int{}
var downList = [N_FLOORS]int{}
var cabList = [N_FLOORS]int{}

func determineRole(receiveChan chan Message) {
	start := time.Now()
	masterFound := false
	backupFound := false

	// Listen for 1 second
	for time.Since(start) < time.Second {
		select {
		case msg := <-receiveChan:
			// Check if master is alive
			if msg.Role == Role(MASTER) {
				masterFound = true

				// Check if backup is alive
			} else if msg.Role == Role(BACKUP) {
				backupFound = true
			}
		}
	}

	if !masterFound {
		currentRole = MASTER
		fmt.Println("No MASTER found, becoming MASTER")
	} else if !backupFound {
		currentRole = BACKUP
		fmt.Println("No BACKUP found, becoming BACKUP")
	} else {
		currentRole = SLAVE
		fmt.Println("MASTER and BACKUP found, staying SLAVE")

		// If is slave go back to listen
		// Theres probably a better way to do this but im tired
		determineRole(receiveChan)
	}
}

func main() {
	port := 9000

	sendChan := make(chan Message)
	receiveChan := make(chan Message)

	go bcast.Transmitter(port, sendChan)
	go bcast.Receiver(port, receiveChan)

	go determineRole(receiveChan)

	// Send every 200ms
	go func() {
		for {
			sendChan <- Message{
				Role:      currentRole,
				Direction: currentDirection,
				Floor:     lastFloor,
				UpList:    upList[:],
				DownList:  downList[:],
				CabList:   cabList[:],
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()

	for {}
}
