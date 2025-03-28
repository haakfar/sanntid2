// timer.go
package elevatorLogic

import (
	"time"
)

// This module is basically a timer used to regulate the door opening

var timerEndTime time.Time
var timerActive bool

// Starts timer
func TimerStart(duration float64) {
	timerEndTime = time.Now().Add(time.Duration(duration * float64(time.Second)))
	timerActive = true
}

// Stops timer
func TimerStop() {
	timerActive = false
}

// Returns true when timer times out
func TimerTimedOut() bool {
	return timerActive && time.Now().After(timerEndTime)
}
