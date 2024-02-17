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
	floorSens_Ch := make(chan int)
	stopButton_Ch := make(chan bool)
	onDoorsClosing_Ch := make(chan bool)

	elev_floor_timeout_ch := make(chan bool)
	// elev_door_timeout_ch := make(chan bool)

	floorWatchdog := watchdog.New(time.Second*10,
		elev_floor_timeout_ch,
        elevatorStuckCh,
		func() {
			elevator.Stop()
            log.Fatal("floor watchdog")
		})

	doorWatchdog := watchdog.New(time.Second*5,
		make(chan bool),
        elevatorStuckCh,
		func() {
			elevator.Stop()
            log.Fatal("Door watchdog")
		})

	go watchdog.Start(floorWatchdog)
	go watchdog.Start(doorWatchdog)


	ticker := time.NewTicker(time.Millisecond * 500)
	elevator.PollElevatorIO(buttonCh, floorSens_Ch, stopButton_Ch)
	for {
		select {
        case req := <- requestUpdateCh:
            e.Requests = req
            handleRequestUpdate(&e, onDoorsClosing_Ch)
            elevatorUpdateCh <- e
		case event := <- buttonCh:
			//requests.UpdateRequests(event, *e.Requests)
			//handle_button_press(event, e, doorClosedCh)
            orderChan <- elevator.Order{
                Type: event.Button,
                AtFloor: event.Floor,
            }
		case event := <-floorSens_Ch:
			watchdog.Feed(floorWatchdog)
            hasStopped := handleFloorArrival(event, &e, onDoorsClosing_Ch)
            if hasStopped {
                stopedAtFloor <- event
            }

		case <-onDoorsClosing_Ch:
            stopedAtFloor <- e.CurrentFloor
            e.Requests = <- requestUpdateCh 
			handleDoorsClosing(&e, onDoorsClosing_Ch)
			elevator.PrintElevator(e)
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


func handleRequestUpdate(e *elevator.Elevator, onDoorClosingCh chan bool) {
    // TODO: The requests recieved from the request handler are assumed to be true,
    // so we should now light the buttons
    switch e.State {
    case elevator.IDLE:
        e.Dir, e.State = requests.GetNewDirectionAndState(*e)
        if e.State == elevator.DOOR_OPEN {
            elevator.Stop()
            go elevator.OpenDoors(onDoorClosingCh)
        } else {
            elevio.SetMotorDirection(e.Dir)
        }
        // Probably no need to do anything on the other states:
        // MOVING => next moves are handled by handleFloorArrival
        // Doors open => next moves are handled by the handleDoorsClosing
    }
}

func handleDoorsClosing(e *elevator.Elevator, onDoorClosingCh chan bool) {
    e.Dir, e.State = requests.GetNewDirectionAndState(*e)
    elevio.SetMotorDirection(e.Dir)
    if e.State == elevator.DOOR_OPEN {
        go elevator.OpenDoors(onDoorClosingCh)
    }
}

func handleFloorArrival(floor int, e *elevator.Elevator, onDoorsClosingCh chan bool) bool {
	e.CurrentFloor = floor
	if e.State == elevator.IDLE || e.State == elevator.DOOR_OPEN {
		return false
	}
    e.Dir, e.State = requests.GetNewDirectionAndState(*e)
    if e.State == elevator.DOOR_OPEN {
        elevator.Stop()
        go elevator.OpenDoors(onDoorsClosingCh) 
        return true
    } else {
        elevio.SetMotorDirection(e.Dir)
    }
    return false
}
