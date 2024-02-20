package elevatorcontroller

import (
	"Driver-go/elevio"
	"localelevator/elevator"
	"localelevator/requests"
	"log"
	"os"
	"os/exec"
	"sanntid/watchdog"
	"time"
)

func ListenAndServe(
	e elevator.Elevator,
	requestUpdateCh chan elevator.Requests,
	elevatorStuckCh chan struct{},
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

	floorWatchdog := watchdog.New(time.Second*15,
		make(chan bool),
		elevatorStuckCh,
		func() {
			elevator.Stop()
			log.Fatal("floor watchdog")
		})

	doorWatchdog := watchdog.New(time.Second*30,
		make(chan bool),
		elevatorStuckCh,
		func() {
			elevator.Stop()
			log.Fatal("Door watchdog")
		})

	go watchdog.Start(floorWatchdog)
	go watchdog.Start(doorWatchdog)

	ticker := time.NewTicker(time.Millisecond * 500)
	elevator.PollElevatorIO(buttonCh, floorSensCh, stopButtonCh, obstructionCh)
	for {
		select {
		case req := <-requestUpdateCh:
			e.Requests = req
            elevator.SetLights(e)
			handleRequestUpdate(&e, onDoorsClosingCh, obstructionCh)
			elevatorUpdateCh <- e

		case event := <-buttonCh:
			orderChan <- elevator.Order{
				Type:    event.Button,
				AtFloor: event.Floor,
			}

		case event := <-floorSensCh:
			watchdog.Feed(floorWatchdog)
			hasStopped := handleFloorArrival(event, &e, onDoorsClosingCh, obstructionCh)
			if hasStopped {
				stopedAtFloor <- event
			}

		case <-onDoorsClosingCh:
			stopedAtFloor <- e.CurrentFloor
			e.Requests = <-requestUpdateCh
            elevator.SetLights(e)
			handleDoorsClosing(&e, onDoorsClosingCh, obstructionCh)
			elevatorUpdateCh <- e

		case <-ticker.C:
			if printEnabled {
				cmd := exec.Command("clear")
				cmd.Stdout = os.Stdout
				cmd.Run()
				elevator.PrintElevator(e)
			}
			if e.State != elevator.MOVING {
				watchdog.Feed(floorWatchdog)
			}
			if e.State != elevator.DOOR_OPEN {
				watchdog.Feed(doorWatchdog)
			}
		}
	}
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
		// Probably no need to do anything on the other states:
		// MOVING => next moves are handled by handleFloorArrival
		// Doors open => next moves are handled by the handleDoorsClosing
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
	obstructionCh chan bool) bool {
	e.CurrentFloor = floor
    elevio.SetFloorIndicator(floor)
	if e.State == elevator.IDLE || e.State == elevator.DOOR_OPEN {
		return false
	}
    if requests.ShouldStop(*e) {
        elevator.Stop()
        e.State = elevator.DOOR_OPEN
        go elevator.OpenDoors(onDoorsClosingCh, obstructionCh)
    }
	// e.Dir, e.State = requests.GetNewDirectionAndState(*e)
	// if e.State == elevator.DOOR_OPEN {
	// 	elevator.Stop()
	// 	go elevator.OpenDoors(onDoorsClosingCh, obstructionCh)
	// 	return true
	// } else {
	// 	elevio.SetMotorDirection(e.Dir)
	// }
	return false
}
