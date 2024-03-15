package elevatorcontroller

import (
	"Driver-go/elevio"
	// "fmt"
	"sanntid/localelevator/elevator"
	"sanntid/localelevator/requests"
	"time"
)

const doorTimeout time.Duration = 10 * time.Second
const floorTimeout time.Duration = 5 * time.Second

/*
Listen for orders (button presses)
and serve the currently active requests
*/
func ControlSingleElevator(
	thisElevator elevator.Elevator,
	requestUpdateCh chan elevator.Requests,
	elevatorStuckCh chan bool,
	stoppedAtFloor chan int,
	newOrderCh chan elevator.Order,
	elevatorUpdateCh chan elevator.Elevator,
	printEnabled bool,
) {
	buttonCh := make(chan elevio.ButtonEvent)
	floorSensorCh := make(chan int)
	stopButtonCh := make(chan bool)
	onDoorsClosingCh := make(chan bool)
	obstructionCh := make(chan bool)
	elevator.PollElevatorIO(buttonCh, floorSensorCh, stopButtonCh, obstructionCh)

	setInitialState(thisElevator, onDoorsClosingCh, obstructionCh)

	doorTimer := time.NewTicker(doorTimeout)
	floorTimer := time.NewTicker(floorTimeout)
	watchdogTicker := time.NewTicker(time.Millisecond * 250)

	for {
		select {
		case requests := <-requestUpdateCh:
			thisElevator.Requests = requests
			elevator.SetCabLights(thisElevator)
			lastState := thisElevator.State
			lastDir := thisElevator.Dir
			handleRequestUpdate(&thisElevator, onDoorsClosingCh, obstructionCh)
			if lastDir != thisElevator.Dir || lastState != thisElevator.State {
				elevatorUpdateCh <- thisElevator
			}

		case buttonPress := <-buttonCh:
			newOrderCh <- elevator.Order{
				Type:    buttonPress.Button,
				AtFloor: buttonPress.Floor,
			}

		case newFloor := <-floorSensorCh:
			floorTimer.Reset(floorTimeout)
			elevatorStuckCh <- false
			handleFloorArrival(newFloor, &thisElevator, onDoorsClosingCh, obstructionCh)
			elevatorUpdateCh <- thisElevator

		case <-onDoorsClosingCh:
			elevatorStuckCh <- false
			stoppedAtFloor <- thisElevator.CurrentFloor
			thisElevator.Requests = <-requestUpdateCh
			elevator.SetCabLights(thisElevator)
			handleDoorsClosing(&thisElevator, onDoorsClosingCh, obstructionCh)
			elevatorUpdateCh <- thisElevator

		case <-watchdogTicker.C:
			if printEnabled {
				elevator.PrintElevator(thisElevator)
			}
			if thisElevator.State != elevator.MOVING {
				floorTimer.Reset(floorTimeout)
			}
			if thisElevator.State != elevator.DOOR_OPEN {
				doorTimer.Reset(doorTimeout)
			}
		case <-floorTimer.C:
			elevatorStuckCh <- true
		case <-doorTimer.C:
			elevatorStuckCh <- true

		}
	}
}

func handleRequestUpdate(elev *elevator.Elevator,
	onDoorClosingCh chan bool,
	obstructionCh chan bool) {
	switch elev.State {
	case elevator.IDLE:
		elev.Dir, elev.State = requests.GetNewDirectionAndState(*elev)
		if elev.State == elevator.DOOR_OPEN {
			elevator.Stop()
			go elevator.OpenDoors(onDoorClosingCh, obstructionCh)
		} else {
			elevio.SetMotorDirection(elev.Dir)
		}
	case elevator.DOOR_OPEN:
		elevator.Stop()
	case elevator.MOVING:
		elevio.SetMotorDirection(elev.Dir)
	}
}

func handleDoorsClosing(elev *elevator.Elevator,
	onDoorClosingCh chan bool,
	obstructionCh chan bool) {
	elev.Dir, elev.State = requests.GetNewDirectionAndState(*elev)
	elevio.SetMotorDirection(elev.Dir)
	if elev.State == elevator.DOOR_OPEN {
		go elevator.OpenDoors(onDoorClosingCh, obstructionCh)
	}
}

func handleFloorArrival(floor int,
	elev *elevator.Elevator,
	onDoorsClosingCh chan bool,
	obstructionCh chan bool) {
	elev.CurrentFloor = floor
	elevio.SetFloorIndicator(floor)
	if elev.State == elevator.IDLE || elev.State == elevator.DOOR_OPEN {
		elevator.Stop()
		return
	}
	if requests.ShouldStop(*elev) {
		elevator.Stop()
		elev.State = elevator.DOOR_OPEN
		go elevator.OpenDoors(onDoorsClosingCh, obstructionCh)
	}
}

func setInitialState(elev elevator.Elevator, onDoorClosingCh chan bool, obstructionCh chan bool) {
	elevator.PrintElevator(elev)
	elevio.SetMotorDirection(elev.Dir)
	elevator.SetCabLights(elev)
	elevio.SetDoorOpenLamp(false)
	if elev.State == elevator.DOOR_OPEN {
		go elevator.OpenDoors(onDoorClosingCh, obstructionCh)
	}
}
