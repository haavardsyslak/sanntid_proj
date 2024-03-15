package recovery

import(
    "sanntid/localelevator/elevator"
    "time"
)


func AttemptNetworkRecovery(id string,
	elevatorFromNetworkCh chan elevator.Elevator) (elevator.Elevator, bool) {
	timeout := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-timeout.C:
			return elevator.Elevator{}, false
		case elevator := <-elevatorFromNetworkCh:
			if elevator.Id == id {
				return elevator, true
			}
		}
	}
}
