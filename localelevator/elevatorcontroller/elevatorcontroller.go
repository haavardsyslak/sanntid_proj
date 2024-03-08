package elevatorcontroller

import (
	"Driver-go/elevio"
	// "fmt"
	"sanntid/localelevator/elevator"
    "sanntid/localelevator/requests"
	"os"
	"os/exec"
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
	stopedAtFloor chan int,
	orderChan chan elevator.Order,
	elevatorUpdateCh chan elevator.Elevator,
	printEnabled bool,
) {
	buttonCh := make(chan elevio.ButtonEvent)
	floorSensCh := make(chan int)
	stopButtonCh := make(chan bool)
	onDoorsClosingCh := make(chan bool)
	obstructionCh := make(chan bool)

    setInitialState(e, onDoorsClosingCh, obstructionCh)
    
    doorTimer := time.NewTicker(doorTimeout)
    floorTimer := time.NewTicker(floorTimeout)

	watchdogTicker := time.NewTicker(time.Millisecond * 250)
	elevator.PollElevatorIO(buttonCh, floorSensCh, stopButtonCh, obstructionCh)

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
			orderChan <- elevator.Order{
				Type:    event.Button,
				AtFloor: event.Floor,
			}

		case event := <-floorSensCh:
            floorTimer.Reset(doorTimeout)
            elevatorStuckCh <- false
			handleFloorArrival(event, &e, onDoorsClosingCh, obstructionCh)
            elevatorUpdateCh <- e

		case <-onDoorsClosingCh:
            elevatorStuckCh <- false
			stopedAtFloor <- e.CurrentFloor
			e.Requests = <-requestUpdateCh
            elevator.SetCabLights(e)
			handleDoorsClosing(&e, onDoorsClosingCh, obstructionCh)
			elevatorUpdateCh <- e

        case <- watchdogTicker.C:
			if printEnabled {
				cmd := exec.Command("clear")
				cmd.Stdout = os.Stdout
				cmd.Run()
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

func hasNewRequest(e elevator.Elevator, new elevator.Requests) bool {
    newRequest := false
	for f := e.MinFloor; f <= e.MaxFloor; f++ {
		if e.Requests.Up[f] != new.Up[f] ||
			e.Requests.Down[f] != new.Down[f] ||
			e.Requests.ToFloor[f] != new.ToFloor[f] {
                newRequest = true
                break
            }
        }
        return newRequest
}

func handleRequestUpdate(e *elevator.Elevator,
	onDoorClosingCh chan bool,
	obstructionCh chan bool) {
	switch e.State {
	case elevator.IDLE:
		e.Dir, e.State = requests.GetNewDirectionAndState(*e)
		if e.State == elevator.DOOR_OPEN {
			elevator.Stop()
			go elevator.OpenDoors(onDoorClosingCh, obstructionCh)
		} else {
			elevio.SetMotorDirection(e.Dir)
		}
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

// TODO: return value is not used
func handleFloorArrival(floor int,
	e *elevator.Elevator,
	onDoorsClosingCh chan bool,
	obstructionCh chan bool) bool {
	e.CurrentFloor = floor
    elevio.SetFloorIndicator(floor)
	if e.State == elevator.IDLE || e.State == elevator.DOOR_OPEN {
        elevator.Stop()
		return false
	}
    if requests.ShouldStop(*e) {
        elevator.Stop()
        e.State = elevator.DOOR_OPEN
        go elevator.OpenDoors(onDoorsClosingCh, obstructionCh)
    }
	return false
}


func setInitialState(e elevator.Elevator, onDoorClosingCh chan bool, obstructionCh chan bool) {
    elevator.PrintElevator(e)
    elevio.SetMotorDirection(e.Dir) 
    elevator.SetCabLights(e)
    if e.State == elevator.DOOR_OPEN {
        go elevator.OpenDoors(onDoorClosingCh, obstructionCh)
    }
}
