package elevatorcontroller

import (
	"Driver-go/elevio"
	// "fmt"
	"sanntid/localelevator/elevator"
	"sanntid/localelevator/requests"

	// "log"
	"os"
	"os/exec"
	// "sanntid/watchdog"
	"time"
	// "fmt"
)

const doorTimeout time.Duration = 10 * time.Second
const foorTimeout time.Duration = 5 * time.Second

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

	// floorWatchdog := watchdog.New(time.Second*5,
	// 	make(chan bool),
	// 	elevatorStuckCh,
	// 	func() {
	// 		elevator.Stop()
	// 		fmt.Println("floor watchdog")
	// 	})
	//
	// doorWatchdog := watchdog.New(time.Second*30,
	// 	make(chan bool),
	// 	elevatorStuckCh,
	// 	func() {
	// 		elevator.Stop()
	// 		fmt.Println("Door watchdog")
	// 	})
	//
    setInitialState(e, onDoorsClosingCh, obstructionCh)
    
	// go watchdog.Start(floorWatchdog)
	// go watchdog.Start(doorWatchdog)

    doorTimer := time.NewTicker(10 * time.Second)
    floorTimer := time.NewTicker(5 * time.Second)


	ticker := time.NewTicker(time.Millisecond * 250)
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
			// watchdog.Feed(floorWatchdog)
            floorTimer.Reset(doorTimeout)
            elevatorStuckCh <- false
			hasStopped := handleFloorArrival(event, &e, onDoorsClosingCh, obstructionCh)
			if hasStopped {
				stopedAtFloor <- event
			}

		case <-onDoorsClosingCh:
            elevatorStuckCh <- false
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
				// watchdog.Feed(floorWatchdog)
                floorTimer.Reset(foorTimeout)
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
