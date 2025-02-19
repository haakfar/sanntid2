package main

import (
	"Network-go/network/bcast"
	"flag"
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
const N_ELEVATORS = 3

type Direction int
type Role int

type Message struct {
	ID        int                         `json:"id"`
	Role      Role                        `json:"role"`
	Direction Direction                   `json:"direction"`
	Floor     int                         `json:"floor"`
	UpList    [N_FLOORS]int               `json:"upList"`
	DownList  [N_FLOORS]int               `json:"downList"`
	CabLists  [N_ELEVATORS][N_FLOORS]int  `json:"cabLists"`
}

var currentRole Role = SLAVE
var currentDirection Direction = STAND_STILL
var lastFloor = -1
var elevatorID int

var upList = [N_FLOORS]int{}
var downList = [N_FLOORS]int{}
var cabLists = [N_ELEVATORS][N_FLOORS]int{}

func determineRole(receiveChan chan Message) {
	for {
		start := time.Now()
		masterFound := false
		backupFound := false

		// Listen for 1 second
		for time.Since(start) < time.Second {
			select {
			case msg := <-receiveChan:
				if msg.Role == MASTER {
					masterFound = true
				} else if msg.Role == BACKUP {
					backupFound = true
				}
			}
		}

		if currentRole == BACKUP && !masterFound {
			fmt.Printf("Elevator %d: No MASTER found, BACKUP becoming MASTER\n", elevatorID)
			currentRole = MASTER
		} else if currentRole == SLAVE && !backupFound {
			fmt.Printf("Elevator %d: No BACKUP found, SLAVE becoming BACKUP\n", elevatorID)
			currentRole = BACKUP
		}

		switch currentRole {
		case MASTER:
			fmt.Printf("Elevator %d: Now MASTER\n", elevatorID)
		case BACKUP:
			fmt.Printf("Elevator %d: Now BACKUP\n", elevatorID)
		case SLAVE:
			fmt.Printf("Elevator %d: Now SLAVE\n", elevatorID)
		}
	}
}

func main() {
	elevatorIDPtr := flag.Int("id", 0, "ID of the elevator")
	flag.Parse()

	elevatorID = *elevatorIDPtr

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
				ID:        elevatorID,
				Role:      currentRole,
				Direction: currentDirection,
				Floor:     lastFloor,
				UpList:    upList,
				DownList:  downList,
				CabLists:  cabLists,
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()

	

	select {}
}
