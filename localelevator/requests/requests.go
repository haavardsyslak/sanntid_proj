package requests

import (
    "sanntid/config"
	"Driver-go/elevio"
	"sanntid/localelevator/elevator"
)

func UpdateRequests(order elevator.Order, requests elevator.Requests) elevator.Requests {
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

func MergeHallRequests(elevators map[string]elevator.Elevator) elevator.Requests {

    reqs := elevator.Requests {
        Up: make([]bool, config.NumFloors),
        Down: make([]bool, config.NumFloors),
        ToFloor: make([]bool, config.NumFloors),
    }

    for _, e := range elevators {
        for f := e.MinFloor; f <= e.MaxFloor; f++ {
            if e.Requests.Up[f] {
                reqs.Up[f] = true
            }
            if e.Requests.Down[f] {
                reqs.Down[f] = true
            }
            if e.Requests.ToFloor[f] {
                reqs.ToFloor[f] = true
            }
        }
    }
    return reqs
}

