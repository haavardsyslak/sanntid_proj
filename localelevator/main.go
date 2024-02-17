package main

import (
	"Driver-go/elevio"
	"localelevator/elevator"
	"localelevator/elevatorcontroller"
)

func main() {
	floor := elevator.Init()
	nFloors := 4

	elev := elevator.Elevator{
		Dir:   elevio.MD_Stop,
		State: elevator.IDLE,
		Requests: elevator.Requests{
			Up:      make([]bool, nFloors),
			Down:    make([]bool, nFloors),
			ToFloor: make([]bool, nFloors),
		},
		MaxFloor:     3,
		MinFloor:     0,
		CurrentFloor: floor,
	}

	elevatorStuckCh := make(chan struct{})
	stopedAtFloor := make(chan int)
	orderCh := make(chan elevator.Order)
	elevatorUpdateCh := make(chan elevator.Elevator)
	requestUpdateCh := make(chan elevator.Requests)

	go elevatorcontroller.ListenAndServe(elev,
		requestUpdateCh,
		elevatorStuckCh,
		stopedAtFloor,
		orderCh, 
        elevatorUpdateCh,
        false)

	for {
		select {
        case floor := <-stopedAtFloor:
            requestUpdateCh <- clearRequest(floor, elev)
		case order := <-orderCh:
			requestUpdateCh <- updateRequests(order, elev.Requests)
		case <-elevatorStuckCh:
		case elev = <-elevatorUpdateCh:
		}
	}
}

func clearRequest(floor int, e elevator.Elevator) elevator.Requests {
    requests := e.Requests
    switch e.Dir {
    case elevio.MD_Up:
        requests.Up[floor] = false
        requests.ToFloor[floor] = false
    case elevio.MD_Down:
        requests.Down[floor] = false
        requests.ToFloor[floor] = false
    default:
        requests.Down[floor] = false
        requests.ToFloor[floor] = false
        requests.Up[floor] = false
    }
    return requests
}

func updateRequests(order elevator.Order, requests elevator.Requests) elevator.Requests {
	switch order.Type {
	case elevio.BT_HallUp:
		requests.Up[order.AtFloor] = true
	case elevio.BT_HallDown:
		requests.Down[order.AtFloor] = true
	case elevio.BT_Cab:
		requests.ToFloor[order.AtFloor] = true
	}
	return requests
}
