// timer.go
package elevatorLogic

import (
	"time"
)

var timerEndTime time.Time
var timerActive bool

func TimerStart(duration float64) {
	timerEndTime = time.Now().Add(time.Duration(duration * float64(time.Second)))
	timerActive = true
}

func TimerStop() {
	timerActive = false
}

func TimerTimedOut() bool {
	return timerActive && time.Now().After(timerEndTime)
}
