package elevatorcontroller

import (
	"Driver-go/elevio"
	"sanntid/localelevator/elevator"
	"sanntid/localelevator/requests"
	// "log"
	"os"
	"os/exec"
	"sanntid/watchdog"
	"time"
    // "fmt"
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
			panic("floor watchdog")
		})

	doorWatchdog := watchdog.New(time.Second*30,
		make(chan bool),
		elevatorStuckCh,
		func() {
			elevator.Stop()
			panic("Door watchdog")
		})

    setInitialState(e, onDoorsClosingCh, obstructionCh)
    
	go watchdog.Start(floorWatchdog)
	go watchdog.Start(doorWatchdog)

	ticker := time.NewTicker(time.Millisecond * 1000)
	elevator.PollElevatorIO(buttonCh, floorSensCh, stopButtonCh, obstructionCh)
	for {
		select {
		case req := <-requestUpdateCh:
            if hasNewRequest(e, req) {
                e.Requests = req
                elevator.SetCabLights(e)
                handleRequestUpdate(&e, onDoorsClosingCh, obstructionCh)
                elevatorUpdateCh <- e
            }


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
            elevator.SetCabLights(e)
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
        elevator.Stop()
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


func setInitialState(e elevator.Elevator, onDoorClosingCh chan bool, obstructionCh chan bool) {
    elevator.PrintElevator(e)
    elevio.SetMotorDirection(e.Dir) 
    elevator.SetCabLights(e)
    if e.State == elevator.DOOR_OPEN {
        go elevator.OpenDoors(onDoorClosingCh, obstructionCh)
    }
}
