package elevatorcontroller

import (
	"Driver-go/elevio"
	// "fmt"
	"sanntid/localelevator/elevator"
	"sanntid/localelevator/requests"
	"time"
)

const doorTimeout time.Duration = 5 * time.Second
const floorTimeout time.Duration = 5 * time.Second
/*
 Listen for orders (button presses)
 and serve the currently active requests
*/
func ListenAndServe(
	e elevator.Elevator,
	requestUpdateCh chan elevator.Requests,
	elevatorStuckCh chan bool,
	stoppedAtFloor chan int,
	orderCh chan elevator.Order,
	elevatorUpdateCh chan elevator.Elevator,
	printEnabled bool,
) {
	buttonCh := make(chan elevio.ButtonEvent)
	floorSensorCh := make(chan int)
	stopButtonCh := make(chan bool)
	onDoorsClosingCh := make(chan bool)
	obstructionCh := make(chan bool)

    setInitialState(e, onDoorsClosingCh, obstructionCh)
    
    doorTimer := time.NewTicker(doorTimeout)
    floorTimer := time.NewTicker(floorTimeout)

	watchdogTicker := time.NewTicker(time.Millisecond * 250)
	elevator.PollElevatorIO(buttonCh, floorSensorCh, stopButtonCh, obstructionCh)

	for {
		select {
		case req := <-requestUpdateCh:
            e.Requests = req
            elevator.SetCabLights(e)
            lastState := e.State
            lastDir := e.Dir
            handleRequestUpdate(&e, onDoorsClosingCh, obstructionCh)
            if lastDir != e.Dir || lastState != e.State {
                elevatorUpdateCh <- e
            }

		case event := <-buttonCh:
			orderCh <- elevator.Order{
				Type:    event.Button,
				AtFloor: event.Floor,
			}

		case event := <-floorSensorCh:
            floorTimer.Reset(floorTimeout)
            elevatorStuckCh <- false
			handleFloorArrival(event, &e, onDoorsClosingCh, obstructionCh)
            elevatorUpdateCh <- e

		case <-onDoorsClosingCh:
            elevatorStuckCh <- false
			stoppedAtFloor <- e.CurrentFloor
			e.Requests = <-requestUpdateCh
            elevator.SetCabLights(e)
			handleDoorsClosing(&e, onDoorsClosingCh, obstructionCh)
			elevatorUpdateCh <- e

        case <- watchdogTicker.C:
			if printEnabled {
				elevator.PrintElevator(e)
			}
			if e.State != elevator.MOVING {
                floorTimer.Reset(floorTimeout)
			}
			if e.State != elevator.DOOR_OPEN {
				doorTimer.Reset(doorTimeout)
			}
        case <- floorTimer.C:
            elevatorStuckCh <- true
        case <- doorTimer.C:
            elevatorStuckCh <- true
            
		}
	}
}

func handleRequestUpdate(e *elevator.Elevator,
	onDoorClosingCh chan bool,
	obstructionCh chan bool) {
	switch e.State {
	case elevator.IDLE :
		e.Dir, e.State = requests.GetNewDirectionAndState(*e)
		if e.State == elevator.DOOR_OPEN {
			elevator.Stop()
			go elevator.OpenDoors(onDoorClosingCh, obstructionCh)
		} else {
			elevio.SetMotorDirection(e.Dir)
		}
    case elevator.DOOR_OPEN:
        elevator.Stop()
    case elevator.MOVING:
        elevio.SetMotorDirection(e.Dir)
	}
}

func handleDoorsClosing(e *elevator.Elevator,
	onDoorClosingCh chan bool,
	obstructionCh chan bool) {
	e.Dir, e.State = requests.GetNewDirectionAndState(*e)
	elevio.SetMotorDirection(e.Dir)
	if e.State == elevator.DOOR_OPEN {
		go elevator.OpenDoors(onDoorClosingCh, obstructionCh)
	}
}

func handleFloorArrival(floor int,
	e *elevator.Elevator,
	onDoorsClosingCh chan bool,
	obstructionCh chan bool) {
	e.CurrentFloor = floor
    elevio.SetFloorIndicator(floor)
	if e.State == elevator.IDLE || e.State == elevator.DOOR_OPEN {
        elevator.Stop()
        return
	}
    if requests.ShouldStop(*e) {
        elevator.Stop()
        e.State = elevator.DOOR_OPEN
        go elevator.OpenDoors(onDoorsClosingCh, obstructionCh)
    }
}


func setInitialState(e elevator.Elevator, onDoorClosingCh chan bool, obstructionCh chan bool) {
    elevator.PrintElevator(e)
    elevio.SetMotorDirection(e.Dir) 
    elevator.SetCabLights(e)
    elevio.SetDoorOpenLamp(false)
    if e.State == elevator.DOOR_OPEN {
        go elevator.OpenDoors(onDoorClosingCh, obstructionCh)
    }
}
