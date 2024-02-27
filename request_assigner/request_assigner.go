package request_assigner

import (
	//"fmt"
	"Driver-go/elevio"
	"localelevator/elevator"
	"localelevator/requests"
)

const (
	TravelTime   = 1.7 // Time it takes for an elevator to travel from one floor to another
	DoorOpenTime = 3
)

// Simulates execution of elevator and returns a cost - elevator with lowest cost should serve the request
func TimeToIdle(e_sim elevator.Elevator) float32 {
	e := e_sim
	var duration float32 = 0

	switch e.State {
	case elevator.IDLE:
		e.Dir = requests.SimUpdateDirection(e).Dir
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
		if requests.SimShouldStop(e) {
			e = requests.SimClearRequest(e.CurrentFloor, e)
			duration += DoorOpenTime
			e.Dir = requests.SimUpdateDirection(e).Dir
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

func AssignRequest(elevators map[int]elevator.Elevator, request elevator.Order) elevator.Elevator {
	var currentDuration float32 = 0
	var bestDuration float32
	var bestElevator int
	var e_sim elevator.Elevator
	for i := 0; i < len(elevators); i++ {
		e_sim = elevators[i] // copying the elevator
		buttonEvent := elevio.ButtonEvent{
			Floor:  request.AtFloor,
			Button: request.Type,
		}
		requests.UpdateRequests(buttonEvent, &e_sim.Requests)
		currentDuration = TimeToIdle(e_sim)
		if currentDuration < bestDuration {
			bestDuration = currentDuration
			bestElevator = i
		}
	}
	return elevators[bestElevator]
}
