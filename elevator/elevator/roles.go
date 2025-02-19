package elevator


import (
	"fmt"
	"time"
	"Config/config"
)

var currentRole config.Role = config.SLAVE

func DetermineRole(receiveChan chan config.Message, elevatorID int, roleChan chan config.Role){
	for {
		start := time.Now()
		masterFound := false
		backupFound := false

		// Listen for 1 second
		for time.Since(start) < time.Second {
			select {
			case msg := <-receiveChan:
				if msg.Role == config.MASTER {
					masterFound = true
				} else if msg.Role == config.BACKUP {
					backupFound = true
				}
			}
		}

		if currentRole == config.BACKUP && !masterFound {
			fmt.Printf("Elevator %d: No MASTER found, BACKUP becoming MASTER\n", elevatorID)
			currentRole = config.MASTER
			roleChan <- config.MASTER
		} else if currentRole == config.SLAVE && !backupFound {
			fmt.Printf("Elevator %d: No BACKUP found, SLAVE becoming BACKUP\n", elevatorID)
			currentRole = config.BACKUP
			roleChan <- config.BACKUP
		}

		switch currentRole {
		case config.MASTER:
			fmt.Printf("Elevator %d: Now MASTER\n", elevatorID)
		case config.BACKUP:
			fmt.Printf("Elevator %d: Now BACKUP\n", elevatorID)
		case config.SLAVE:
			fmt.Printf("Elevator %d: Now SLAVE\n", elevatorID)
		}
	}
}