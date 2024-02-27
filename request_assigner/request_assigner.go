package request_assigner

import (
	//"fmt"
	"Driver-go/elevio"
	"sanntid/localelevator/elevator"
	"sanntid/localelevator/requests"
    "sanntid/localelevator/elevatorcontroller"
    // "fmt"
)

const (
	TravelTime   = 1.7 // Time it takes for an elevator to travel from one floor to another
	DoorOpenTime = 3
)

var thisId string

func HandleOrders(thisElevator elevator.Elevator,
	elevatorToNetwork chan <-  elevator.Elevator,
	elevatorFromNetwork <- chan elevator.Elevator,
	elevatorLostCh <-chan string,
) {
	elevatorStuckCh := make(chan struct{})
	stopedAtFloor := make(chan int)
	orderCh := make(chan elevator.Order, 1000)
	requestUpdateCh := make(chan elevator.Requests, 1000)
	elevators := make(map[string]elevator.Elevator)

	thisId = thisElevator.Id
	elevators[thisElevator.Id] = thisElevator
	elevatorUpdateCh := make(chan elevator.Elevator)

    elevatorToNetwork <- thisElevator

	go elevatorcontroller.ListenAndServe(thisElevator,
		requestUpdateCh,
		elevatorStuckCh,
		stopedAtFloor,
		orderCh,
		elevatorUpdateCh,
		false)

	for {
		select {
		case floor := <-stopedAtFloor:
			e := elevators[thisId]
			e.CurrentFloor = floor
			e.Requests = requests.ClearAtCurrentFloor(floor, e)
			elevators[thisId] = e
			elevatorToNetwork <- elevators[thisId]

		case order := <-orderCh:
			e := AssignRequest(elevators, order)
			elevatorToNetwork <- e

		case <-elevatorStuckCh:

		case e := <-elevatorUpdateCh:
			elevators[e.Id] = e
			elevatorToNetwork <- e

		case e := <-elevatorFromNetwork:
			if e.Id == thisId {
				requestUpdateCh <- e.Requests
			}
			elevators[e.Id] = e
            e.Requests = mergeAllHallReqs(elevators)
            elevator.SetHallLights(e)

		case lostId := <-elevatorLostCh:
			lostElevator := elevators[lostId]
			delete(elevators, lostId)
			reassignOrders(lostElevator, elevators, elevatorToNetwork)
		}
	}
}

func reassignOrders(lostElevator elevator.Elevator, 
elevators map[string]elevator.Elevator,
elevatorToNetwork chan <- elevator.Elevator) {

}


// Simulates execution of elevator and returns a cost - elevator with lowest cost should serve the request
func TimeToIdle(e_sim elevator.Elevator) float32 {
	e := e_sim
	var duration float32 = 0

	switch e.State {
	case elevator.IDLE:
		e.Dir, _ = requests.GetNewDirectionAndState(e)
		if e.Dir == elevio.MD_Stop {
			duration = 0
		}
	case elevator.MOVING:
		duration += TravelTime / 2
		e.CurrentFloor += int(e.Dir)
	case elevator.DOOR_OPEN:
		duration -= DoorOpenTime / 2
	}

	for {
		if requests.ShouldStop(e) {
			e.Requests = requests.ClearAtCurrentFloor(e.CurrentFloor, e)
            // e = requests.SimClearRequest(e_sim)
			duration += DoorOpenTime
			e.Dir, _ = requests.GetNewDirectionAndState(e)
			if e.Dir == elevio.MD_Stop {
				return duration
			}
		}
		e.CurrentFloor += int(e.Dir)
		duration += TravelTime
	}
}

// TODO: lag funksjon som bruker TimeToIdle på alle heiser og setter den unassigned requesten i den mest optimale heisen
// sin request queue. Må vell her simulere å legge requesten til køen ved å legge den til i den kopierte heisen sin kø
// sånn at HasRequest funksjonene funker?
// "Remember to copy the Elevator data and add the new unassigned request to that copy before calling timeToIdle..."

func AssignRequest(elevators map[string]elevator.Elevator,
    order elevator.Order) elevator.Elevator {
    if order.Type == elevio.BT_Cab {
        e := elevators[thisId]
        e.Requests = requests.UpdateRequests(order, e.Requests)
        return e
    }
	var currentDuration float32 = 0
	var bestDuration float32 = 1000
	var bestElevator string
    for id := range elevators {
        e := elevator.CopyElevator(elevators[id])
		e.Requests = requests.UpdateRequests(order, e.Requests)
		currentDuration = TimeToIdle(e)
		if currentDuration < bestDuration {
			bestDuration = currentDuration
			bestElevator = id
		}
	}
    e := elevators[bestElevator]
    e.Requests = requests.UpdateRequests(order, e.Requests)
	return e
}

func mergeAllHallReqs(elevators map[string]elevator.Elevator) elevator.Requests {

    reqs := elevator.Requests {
        Up: make([]bool, elevators[thisId].MaxFloor + 1 - elevators[thisId].MinFloor),
        Down: make([]bool, elevators[thisId].MaxFloor + 1 - elevators[thisId].MinFloor),
        ToFloor: make([]bool, elevators[thisId].MaxFloor + 1 - elevators[thisId].MinFloor),
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


