package requests

import (
	"Driver-go/elevio"
	"sanntid/localelevator/elevator"
    "fmt"
)

func UpdateRequests(order elevator.Order, requests elevator.Requests) elevator.Requests {
    fmt.Println(requests)
	switch order.Type {
	case elevio.BT_HallUp:
		requests.Up[order.AtFloor] = true
	case elevio.BT_HallDown:
		requests.Down[order.AtFloor] = true
	case elevio.BT_Cab:
		requests.ToFloor[order.AtFloor] = true
	}
    fmt.Println(requests)
	return requests
}

func HasRequestBelow(elevator elevator.Elevator) bool {
	for f := elevator.CurrentFloor - 1; f >= elevator.MinFloor; f-- {
		if elevator.Requests.Down[f] ||
			elevator.Requests.ToFloor[f] ||
			elevator.Requests.Up[f] {
			return true
		}
	}
	return false
}

func HasRequestAbove(elevator elevator.Elevator) bool {
	for f := elevator.CurrentFloor + 1; f <= elevator.MaxFloor; f++ {
		if elevator.Requests.Up[f] ||
			elevator.Requests.ToFloor[f] ||
			elevator.Requests.Down[f] {
			return true
		}
	}
	return false
}

func HasRequestHere(elevator elevator.Elevator) bool {
	return (elevator.Requests.Up[elevator.CurrentFloor] ||
		elevator.Requests.Down[elevator.CurrentFloor] ||
		elevator.Requests.ToFloor[elevator.CurrentFloor])
}

func SimClearRequest(e elevator.Elevator) elevator.Elevator {
    floor := e.CurrentFloor
	switch e.Dir {
	case elevio.MD_Up:
		e.Requests.Up[floor] = false
		e.Requests.ToFloor[floor] = false
		if floor == e.MaxFloor {
			e.Requests.Down[floor] = false
		}
	case elevio.MD_Down:
		e.Requests.Down[floor] = false
		e.Requests.ToFloor[floor] = false
		if floor == e.MinFloor {
			e.Requests.Up[floor] = false
		}
	default:
		e.Requests.Down[floor] = false
		e.Requests.ToFloor[floor] = false
		e.Requests.Up[floor] = false
	}
	return e
}


func ClearRequest(floor int, e *elevator.Elevator, reqType elevio.ButtonType) {
	switch reqType {
	case elevio.BT_HallUp:
		e.Requests.Up[floor] = false
		e.Requests.ToFloor[floor] = false
	case elevio.BT_HallDown:
		e.Requests.Down[floor] = false
		e.Requests.ToFloor[floor] = false
	case elevio.BT_Cab:
		e.Requests.ToFloor[floor] = false
	}
}

func ShouldStop(e elevator.Elevator) (bool) {
    switch(e.Dir){
    case elevio.MD_Down: 
        return e.Requests.Down[e.CurrentFloor] ||
        e.Requests.ToFloor[e.CurrentFloor]      ||
        !HasRequestBelow(e);
    case elevio.MD_Up:
            return e.Requests.Up[e.CurrentFloor]   ||
            e.Requests.ToFloor[e.CurrentFloor] ||
            !HasRequestAbove(e);
    case elevio.MD_Stop:
        return true
    default:
        return true;
    }
}


func GetNewDirectionAndState(e elevator.Elevator) (elevio.MotorDirection, elevator.ElevatorState) {
	switch e.Dir {
	case elevio.MD_Up:
		if HasRequestAbove(e) {
			return elevio.MD_Up, elevator.MOVING
        } else if HasRequestHere(e) {
			return elevio.MD_Stop, elevator.DOOR_OPEN
		} else if HasRequestBelow(e) {
			return elevio.MD_Down, elevator.MOVING
		} else {
			return elevio.MD_Stop, elevator.IDLE
		}
	case elevio.MD_Down:
		 if HasRequestBelow(e) {
			return elevio.MD_Down, elevator.MOVING
        } else if HasRequestHere(e) {
			return elevio.MD_Stop, elevator.DOOR_OPEN
		} else if HasRequestAbove(e) {
			return elevio.MD_Up, elevator.MOVING
		} else {
			return elevio.MD_Stop, elevator.IDLE
		}
	case elevio.MD_Stop:
		if HasRequestHere(e) {
			return elevio.MD_Stop, elevator.DOOR_OPEN
		} else if HasRequestAbove(e) {
			return elevio.MD_Up, elevator.MOVING
		} else if HasRequestBelow(e) {
			return elevio.MD_Down, elevator.MOVING
		} else {
			return elevio.MD_Stop, elevator.IDLE
		}
	default:
		return elevio.MD_Stop, elevator.IDLE
	}
}


func ClearAtCurrentFloor(floor int, e elevator.Elevator) elevator.Requests {
    e.Requests.ToFloor[floor] = false
    switch e.Dir {
    case elevio.MD_Up:
        if !HasRequestAbove(e) && !e.Requests.Up[floor] {
            e.Requests.Down[floor] = false
        }
        e.Requests.Up[floor] = false
    case elevio.MD_Down:
        if !HasRequestBelow(e) && !e.Requests.Down[floor] {

            e.Requests.Up[floor] = false
        }
        e.Requests.Down[floor] = false
    default:
        e.Requests.Up[floor] = false
        e.Requests.Down[floor] = false
    }
    return e.Requests
}
