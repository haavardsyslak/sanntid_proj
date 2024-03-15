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

func HasRequestBelow(elev elevator.Elevator) bool {
	for floor := elev.CurrentFloor - 1; floor >= elev.MinFloor; floor-- {
		if elev.Requests.Down[floor] ||
			elev.Requests.ToFloor[floor] ||
			elev.Requests.Up[floor] {
			return true
		}
	}
	return false
}

func HasRequestAbove(elev elevator.Elevator) bool {
	for floor := elev.CurrentFloor + 1; floor <= elev.MaxFloor; floor++ {
		if elev.Requests.Up[floor] ||
			elev.Requests.ToFloor[floor] ||
			elev.Requests.Down[floor] {
			return true
		}
	}
	return false
}

func HasRequestHere(elev elevator.Elevator) bool {
	return (elev.Requests.Up[elev.CurrentFloor] ||
		elev.Requests.Down[elev.CurrentFloor] ||
		elev.Requests.ToFloor[elev.CurrentFloor])
}


func ShouldStop(elev elevator.Elevator) (bool) {
    switch(elev.Dir){
    case elevio.MD_Down: 
        return elev.Requests.Down[elev.CurrentFloor] ||
        elev.Requests.ToFloor[elev.CurrentFloor]      ||
        !HasRequestBelow(elev);
    case elevio.MD_Up:
            return elev.Requests.Up[elev.CurrentFloor]   ||
            elev.Requests.ToFloor[elev.CurrentFloor] ||
            !HasRequestAbove(elev);
    case elevio.MD_Stop:
        return true
    default:
        return true;
    }
}


func GetNewDirectionAndState(elev elevator.Elevator) (elevio.MotorDirection, elevator.ElevatorState) {
	switch elev.Dir {
	case elevio.MD_Up:
		if HasRequestAbove(elev) {
			return elevio.MD_Up, elevator.MOVING
        } else if HasRequestHere(elev) {
			return elevio.MD_Stop, elevator.DOOR_OPEN
		} else if HasRequestBelow(elev) {
			return elevio.MD_Down, elevator.MOVING
		} else {
			return elevio.MD_Stop, elevator.IDLE
		}
	case elevio.MD_Down:
		 if HasRequestBelow(elev) {
			return elevio.MD_Down, elevator.MOVING
        } else if HasRequestHere(elev) {
			return elevio.MD_Stop, elevator.DOOR_OPEN
		} else if HasRequestAbove(elev) {
			return elevio.MD_Up, elevator.MOVING
		} else {
			return elevio.MD_Stop, elevator.IDLE
		}
	case elevio.MD_Stop:
		if HasRequestHere(elev) {
			return elevio.MD_Stop, elevator.DOOR_OPEN
		} else if HasRequestAbove(elev) {
			return elevio.MD_Up, elevator.MOVING
		} else if HasRequestBelow(elev) {
			return elevio.MD_Down, elevator.MOVING
		} else {
			return elevio.MD_Stop, elevator.IDLE
		}
	default:
		return elevio.MD_Stop, elevator.IDLE
	}
}


func ClearAtCurrentFloor(floor int, elev elevator.Elevator) elevator.Requests {
    elev.Requests.ToFloor[floor] = false
    switch elev.Dir {
    case elevio.MD_Up:
        if !HasRequestAbove(elev) && !elev.Requests.Up[floor] {
            elev.Requests.Down[floor] = false
        }
        elev.Requests.Up[floor] = false
    case elevio.MD_Down:
        if !HasRequestBelow(elev) && !elev.Requests.Down[floor] {

            elev.Requests.Up[floor] = false
        }
        elev.Requests.Down[floor] = false
    default:
        elev.Requests.Up[floor] = false
        elev.Requests.Down[floor] = false
    }
    return elev.Requests
}

func MergeHallRequests(elevators map[string]elevator.Elevator) elevator.Requests {

    requests := elevator.Requests {
        Up: make([]bool, config.NumFloors),
        Down: make([]bool, config.NumFloors),
        ToFloor: make([]bool, config.NumFloors),
    }

    for _, elev := range elevators {
        for floor := elev.MinFloor; floor <= elev.MaxFloor; floor++ {
            if elev.Requests.Up[floor] {
                requests.Up[floor] = true
            }
            if elev.Requests.Down[floor] {
                requests.Down[floor] = true
            }
            if elev.Requests.ToFloor[floor] {
                requests.ToFloor[floor] = true
            }
        }
    }
    return requests
}

