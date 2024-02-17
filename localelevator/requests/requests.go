package requests

import (
	"Driver-go/elevio"
    "localelevator/elevator"
)

func UpdateRequests(event elevio.ButtonEvent, requests *elevator.Requests) {
    switch event.Button {
	case elevio.BT_Cab:
		requests.ToFloor[event.Floor] = true
	case elevio.BT_HallUp:
		requests.Up[event.Floor] = true
	case elevio.BT_HallDown:
		requests.Down[event.Floor] = true
	}
}

func HasRequestBelow(elevator elevator.Elevator) (bool) {
    for f:= elevator.CurrentFloor - 1; f >= elevator.MinFloor; f-- {
        if elevator.Requests.Down[f] || 
        elevator.Requests.ToFloor[f] ||
        elevator.Requests.Up[f] {
            return true
        }
    }
    return false
}

func HasRequestAbove(elevator elevator.Elevator) (bool) {
    for f := elevator.CurrentFloor + 1; f <= elevator.MaxFloor; f++ {
        if elevator.Requests.Up[f] || 
        elevator.Requests.ToFloor[f] ||
        elevator.Requests.Down[f]{
            return true
        }
    }
    return false
}

func HasRequestHere(floor int, elevator elevator.Elevator) (bool) {
    switch elevator.Dir {
    case elevio.MD_Up:
        return (elevator.Requests.Up[floor] || elevator.Requests.ToFloor[floor])
    case elevio.MD_Down:
        return (elevator.Requests.Down[floor] || elevator.Requests.ToFloor[floor])
    default:
        return (elevator.Requests.Down[floor] || 
        elevator.Requests.ToFloor[floor] || 
        elevator.Requests.Up[floor])
    }
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

func GetNewDirectionAndState(e elevator.Elevator) (elevio.MotorDirection, elevator.ElevatorState) {
    switch e.Dir {
    case elevio.MD_Up:
        if HasRequestHere(e.CurrentFloor, e) {
            return elevio.MD_Up, elevator.DOOR_OPEN
        } else if HasRequestAbove(e) {
            return elevio.MD_Up, elevator.MOVING
        } else if HasRequestBelow(e) {
             return elevio.MD_Down, elevator.MOVING
        } else {
            return elevio.MD_Stop, elevator.IDLE
        }
    case elevio.MD_Down:
         if HasRequestHere(e.CurrentFloor, e) {
            return elevio.MD_Down, elevator.DOOR_OPEN
        } else if HasRequestBelow(e) {
            return elevio.MD_Down, elevator.MOVING
        } else if HasRequestAbove(e) {
            return elevio.MD_Up, elevator.MOVING
        } else {
            return elevio.MD_Stop, elevator.IDLE
        }
    case elevio.MD_Stop:
        if HasRequestHere(e.CurrentFloor, e) {
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
