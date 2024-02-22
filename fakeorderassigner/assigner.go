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
	elevatorToNetwork chan elevator.Elevator,
	elevatorFromNetwork chan elevator.Elevator) {

	elevatorStuckCh := make(chan struct{})
	stopedAtFloor := make(chan int)
	orderCh := make(chan elevator.Order)
	requestUpdateCh := make(chan elevator.Requests)
	elevators := make(map[string]elevator.Elevator)

	thisId = thisElevator.Id
    elev := elevator.New(thisElevator.Id)
    elev.CurrentFloor = thisElevator.CurrentFloor
	elevators[thisElevator.Id] = elev
	elevatorUpdateCh := make(chan elevator.Elevator)

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
			// requestUpdateCh <- updateRequests(order, elev.Requests)
			elevatorToNetwork <- e
		case <-elevatorStuckCh:
		case e := <-elevatorUpdateCh:
			elevators[e.Id] = e
			elevatorToNetwork <- e
		case e := <-elevatorFromNetwork:
            // fmt.Printf("Address of elevators[thisId]: %p\n", &elevators[thisId])
            // fmt.Printf("Address of e: %p\n", &e)
			if e.Id == thisId {
                requestUpdateCh <- e.Requests
			}
			elevators[e.Id] = e
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

func Dummy() {
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

