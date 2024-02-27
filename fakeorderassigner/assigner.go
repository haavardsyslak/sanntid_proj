package fakeorderassigner

import (
	"Driver-go/elevio"
	// "fmt"
	"math/rand"
	"sanntid/localelevator/elevator"
	"sanntid/localelevator/elevatorcontroller"
	"sanntid/localelevator/requests"
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
		true)

	for {
		select {
		case floor := <-stopedAtFloor:
			e := elevators[thisId]
			e.CurrentFloor = floor
			e.Requests = requests.ClearAtCurrentFloor(floor, e)
			elevators[thisId] = e
			elevatorToNetwork <- elevators[thisId]

		case order := <-orderCh:
			e := assignOrder(elevators, order)
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

func reassignOrders(e elevator.Elevator,
	elevators map[string]elevator.Elevator,
	elevatorToNetwork chan <- elevator.Elevator,
) {
	for f, req := range e.Requests.Up {
		if req {
			elevatorToNetwork <- assignOrder(elevators, elevator.Order{
				Type:    elevio.BT_HallUp,
				AtFloor: f,
			})
            e.Requests.Up[f] = false
            elevatorToNetwork <- e
		}
	}
	for f, req := range e.Requests.Down {
		if req {
			elevatorToNetwork <- assignOrder(elevators, elevator.Order{
				Type:    elevio.BT_HallDown,
				AtFloor: f,
			})
            e.Requests.Down[f] = false
            elevatorToNetwork <- e
		}
	}
}

func assignOrder(elevators map[string]elevator.Elevator, order elevator.Order) elevator.Elevator {
	switch order.Type {
	case elevio.BT_Cab:
		e := elevators[thisId]
		e.Requests = updateRequests(order, e.Requests)
		return e
	default:
		e := getRandomElev(elevators)
		e.Requests = updateRequests(order, e.Requests)
		return e
	}
}

func getRandomElev(elevators map[string]elevator.Elevator) elevator.Elevator {
	keys := make([]string, 0, len(elevators))
	for key := range elevators {
		keys = append(keys, key)
	}
	randomIndex := rand.Intn(len(keys))
	randomKey := keys[randomIndex]

	// Print the randomly chosen key
	return elevators[randomKey]
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
