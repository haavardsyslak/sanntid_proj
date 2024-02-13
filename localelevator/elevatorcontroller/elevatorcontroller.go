package elevatorcontroller

import (
	"Driver-go/elevio"
	"localelevator/elevator"
	"localelevator/requests"
	"os"
	"os/exec"
	"sanntid/watchdog"
	"time"
)

func ListenAndServe(
	e elevator.Elevator,
	requestUpdateCh chan elevator.Requests,
	elevatorStuckCh chan bool,
	stopedAtFloor chan int,
	orderChan chan elevator.Order,
    elevatorUpdateCh chan elevator.Elevator,
) {
	buttonCh := make(chan elevio.ButtonEvent)
	floorSens_Ch := make(chan int)
	stopButton_Ch := make(chan bool)
	onDoorsClosing_Ch := make(chan bool)

	elev_floor_timeout_ch := make(chan bool)
	elev_door_timeout_ch := make(chan bool)

	floorWatchdog := watchdog.New(time.Second*10,
		elev_floor_timeout_ch,
		func(elevatorStuckCh chan interface{}) {
			elevator.Stop()
			elevatorStuckCh <- true
		})

	doorWatchdog := watchdog.New(time.Second*5,
		elev_door_timeout_ch,
		func(elevatorStuckCh chan interface{}) {
			elevator.Stop()
			elevatorStuckCh <- true
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
            stopedAtFloor <- event
			watchdog.Feed(floorWatchdog)
            elevatorUpdateCh <- e
			handleFloorArrival(event, &e, onDoorsClosing_Ch)
		case <-onDoorsClosing_Ch:
            stopedAtFloor <- e.CurrentFloor
			handleDoorsClosing(&e, onDoorsClosing_Ch)
            elevatorUpdateCh <- e
		case <-ticker.C:
			cmd := exec.Command("clear")
			cmd.Stdout = os.Stdout
			cmd.Run()
			elevator.PrintElevator(e)

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
        if requests.HasRequestAbove(*e) {
            e.State = elevator.MOVING
            e.Dir = elevio.MD_Up
            elevator.GoUp()
        } else if requests.HasRequestsBelow(*e) {
            e.State = elevator.MOVING
            e.Dir = elevio.MD_Down
            elevator.GoDown()
        } else if requests.HasRequestHere(e.CurrentFloor, *e){
            go elevator.OpenDoors(onDoorClosingCh)
        }
        // Probably no need to do anything on the other states:
        // MOVING => next moves are handled by handleFloorArrival
        // Doors open => next moves are handled by the handleDoorsClosing
    }
}

func handleDoorsClosing(e *elevator.Elevator, onDoorClosingCh chan bool) {
	switch e.Dir {
	case elevio.MD_Stop:
		if requests.HasRequestAbove(*e) {
			e.State = elevator.MOVING
			e.Dir = elevio.MD_Up
			elevator.GoUp()
		} else if requests.HasRequestsBelow(*e) {
			e.State = elevator.MOVING
			elevator.GoDown()
		} else if requests.HasRequestHere(e.CurrentFloor, *e) {
			e.State = elevator.DOOR_OPEN
			//requests.ClearRequest(e.CurrentFloor, e)
			go elevator.OpenDoors(onDoorClosingCh)
		} else {
			e.State = elevator.IDLE
			e.Dir = elevio.MD_Stop
		}

	case elevio.MD_Down:
		if requests.HasRequestsBelow(*e) {
			e.State = elevator.MOVING
			e.Dir = elevio.MD_Down
			elevator.GoDown()
		} else if requests.HasRequestAbove(*e) {
			e.State = elevator.MOVING
			e.Dir = elevio.MD_Up
			elevator.GoUp()
		} else {
			e.State = elevator.IDLE
			e.Dir = elevio.MD_Stop
		}

	case elevio.MD_Up:
		if requests.HasRequestAbove(*e) {
			e.State = elevator.MOVING
			e.Dir = elevio.MD_Up
			elevator.GoUp()
		} else if requests.HasRequestsBelow(*e) {
			e.State = elevator.MOVING
			e.Dir = elevio.MD_Down
			elevator.GoDown()
		} else {
			e.State = elevator.IDLE
			e.Dir = elevio.MD_Stop
		}
	}
}

func handle_button_press(event elevio.ButtonEvent, e *elevator.Elevator, door_closed_ch chan bool) {
	switch e.State {
	case elevator.IDLE:

		if e.CurrentFloor == event.Floor {
			e.State = elevator.DOOR_OPEN
			go elevator.OpenDoors(door_closed_ch)
			// requests.ClearRequest(event.Floor, e)
			return
		}

		switch event.Button {
		case elevio.BT_Cab:
			e.State = elevator.MOVING
			e.Dir = elevator.ServeOrder(e.CurrentFloor, event.Floor)
		case elevio.BT_HallUp:
			e.Dir = elevio.MD_Up
			e.State = elevator.MOVING
			elevator.ServeOrder(e.CurrentFloor, event.Floor)
		case elevio.BT_HallDown:
			e.Dir = elevio.MD_Down
			e.State = elevator.MOVING
			elevator.ServeOrder(e.CurrentFloor, event.Floor)
		}
	}
}

func handleFloorArrival(floor int, e *elevator.Elevator, door_closed_ch chan bool) {
	e.CurrentFloor = floor
	if e.State == elevator.IDLE || e.State == elevator.DOOR_OPEN {
		return
	}
	if requests.HasRequestHere(floor, *e) {
		elevator.Stop()
		e.State = elevator.DOOR_OPEN
		// requests.ClearRequest(floor, e)
		go elevator.OpenDoors(door_closed_ch)
		return
	}

	switch e.Dir {
	case elevio.MD_Up:
		if requests.HasRequestAbove(*e) {
			e.State = elevator.MOVING
			e.Dir = elevio.MD_Up
			elevator.GoUp()
		} else if requests.HasRequestsBelow(*e) {
			e.State = elevator.MOVING
			elevator.GoDown()
		}
	case elevio.MD_Down:
		if requests.HasRequestsBelow(*e) {
			e.State = elevator.MOVING
			e.Dir = elevio.MD_Down
			elevator.GoDown()
		} else if requests.HasRequestAbove(*e) {
			e.State = elevator.MOVING
			e.Dir = elevio.MD_Up
			elevator.GoUp()
		}
	}
}
