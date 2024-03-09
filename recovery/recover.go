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
		case e := <-elevatorFromNetworkCh:
			if e.Id == id {
				return e, true
			}
		}
	}
}
