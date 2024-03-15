package request_assigner

import (
	"Driver-go/elevio"
	"sanntid/localelevator/elevator"
	"sanntid/localelevator/elevatorcontroller"
	"sanntid/localelevator/requests"
	"sort"
)

const (
	TravelTime   = 1.7 // Time it takes for an elevator to travel from one floor to another
	DoorOpenTime = 3
)

var thisId string

func DistributeRequests(thisElevator elevator.Elevator,
	elevatorToNetwork chan<- elevator.Elevator,
	elevatorFromNetwork <-chan elevator.Elevator,
	lostElevatorsCh <-chan []string,
	peerTxEnable chan bool,
) {
	elevatorStuckCh := make(chan bool)
	stopedAtFloorCh := make(chan int)
	newOrderCh := make(chan elevator.Order, 100)
	requestUpdateCh := make(chan elevator.Requests, 100)
	elevatorStateUpdateCh := make(chan elevator.Elevator)

	elevators := make(map[string]elevator.Elevator)

	thisId = thisElevator.Id
	elevators[thisId] = thisElevator

	elevatorToNetwork <- thisElevator

	printElevator := true
	go elevatorcontroller.ControlSingleElevator(elevator.CopyElevator(thisElevator),
		requestUpdateCh,
		elevatorStuckCh,
		stopedAtFloorCh,
		newOrderCh,
		elevatorStateUpdateCh,
		printElevator)

	for {
		select {
		case floor := <-stopedAtFloorCh:
			thisElevator := elevators[thisId]
			thisElevator.CurrentFloor = floor
			thisElevator.Requests = requests.ClearAtCurrentFloor(floor, thisElevator)
			elevators[thisId] = thisElevator
			elevatorToNetwork <- elevators[thisId]

			if len(elevators) == 1 {
				requestUpdateCh <- thisElevator.Requests
				elevator.SetHallLights(thisElevator)
			}

		case order := <-newOrderCh:
			assignedElevator := AssignRequest(elevators, order)
			elevatorToNetwork <- assignedElevator

			if len(elevators) == 1 {

				requestUpdateCh <- requests.MergeHallRequests(elevators)
				elevator.SetHallLights(assignedElevator)
			}

		case isStuck := <-elevatorStuckCh:
			if isStuck {
				// Reassign my own requests when stuck
				lostElevator := elevators[thisId]
				delete(elevators, lostElevator.Id)
				if len(elevators) >= 1 {
					reassignOrders(lostElevator, elevators, elevatorToNetwork)
				}
				elevators[thisId] = lostElevator
				// Disable peerTx such that the other elev wont assign new requests to me
				peerTxEnable <- false
			} else {
				peerTxEnable <- true
			}

		case thisElevator := <-elevatorStateUpdateCh:
			elevators[thisElevator.Id] = thisElevator
			elevatorToNetwork <- thisElevator

		case recievedElevator := <-elevatorFromNetwork:
			if recievedElevator.Id == thisId {
				requestUpdateCh <- recievedElevator.Requests
			}
			elevators[recievedElevator.Id] = recievedElevator
			recievedElevator.Requests = requests.MergeHallRequests(elevators)
			elevator.SetHallLights(recievedElevator)
			if len(elevators) == 1 {
				requestUpdateCh <- recievedElevator.Requests
			}

		case lostElevators := <-lostElevatorsCh:
			// Reassign the lost elevators requests
			for _, lostId := range lostElevators {
				if lostId == thisId {
					continue
				}
				lostElevator := elevators[lostId]
				delete(elevators, lostId)
				if len(elevators) >= 1 {
					reassignOrders(lostElevator, elevators, elevatorToNetwork)
				}
			}
		}
	}
}

// Reassigns the requests of the lostElevator
func reassignOrders(lostElevator elevator.Elevator,
	elevators map[string]elevator.Elevator,
	elevatorToNetwork chan<- elevator.Elevator) {
	for floor, req := range lostElevator.Requests.Up {
		if req {
			elev := AssignRequest(elevators, elevator.Order{
				Type:    elevio.BT_HallUp,
				AtFloor: floor,
			})
			elevatorToNetwork <- elev
			lostElevator.Requests.Up[floor] = false
			elevatorToNetwork <- lostElevator
			elevators[elev.Id] = elev
		}
	}
	for floor, req := range lostElevator.Requests.Down {
		if req {
			elev := AssignRequest(elevators, elevator.Order{
				Type:    elevio.BT_HallDown,
				AtFloor: floor,
			})
			elevatorToNetwork <- elev
			lostElevator.Requests.Down[floor] = false
			elevatorToNetwork <- lostElevator
			elevators[elev.Id] = elev
		}
	}
}

// Simulates execution of elevator and returns a cost
// elevator with lowest cost should serve the request
func TimeToIdle(eSim elevator.Elevator) float32 {
	var duration float32 = 0

	switch eSim.State {
	case elevator.IDLE:
		eSim.Dir, _ = requests.GetNewDirectionAndState(eSim)
		if eSim.Dir == elevio.MD_Stop {
			duration = 0
		}
	case elevator.MOVING:
		duration += TravelTime / 2
		eSim.CurrentFloor += int(eSim.Dir)
	case elevator.DOOR_OPEN:
		duration -= DoorOpenTime / 2
	}

	for {
		if requests.ShouldStop(eSim) {
			eSim.Requests = requests.ClearAtCurrentFloor(eSim.CurrentFloor, eSim)
			duration += DoorOpenTime
			eSim.Dir, _ = requests.GetNewDirectionAndState(eSim)
			if eSim.Dir == elevio.MD_Stop {
				return duration
			}
		}
		eSim.CurrentFloor += int(eSim.Dir)
		duration += TravelTime
	}
}

// Asigns the request to the best available elevator
func AssignRequest(elevators map[string]elevator.Elevator,
	order elevator.Order) elevator.Elevator {
	if order.Type == elevio.BT_Cab {
		e := elevators[thisId]
		e.Requests = requests.UpdateRequests(order, e.Requests)
		return e
	}

	if len(elevators) == 1 {
		e, ok := elevators[thisId]
		if ok {
			e.Requests = requests.UpdateRequests(order, e.Requests)
			return e
		}
	}
	var currentDuration float32 = 0.0
	var bestDuration float32 = 1000.0
	var bestElevator string

	var keys []string
	for key := range elevators {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, id := range keys {
		elev := elevator.CopyElevator(elevators[id])
		elev.Requests = requests.UpdateRequests(order, elev.Requests)
		currentDuration = TimeToIdle(elev)
		if currentDuration < bestDuration {
			bestDuration = currentDuration
			bestElevator = id
		}
	}
	elev := elevators[bestElevator]
	elev.Requests = requests.UpdateRequests(order, elev.Requests)
	return elev
}
