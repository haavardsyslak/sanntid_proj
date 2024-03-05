package request_assigner

import (
	"Driver-go/elevio"
	p "Network-go/network/peers"
	"fmt"
	"sanntid/config"
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
//
func DistributeRequests(thisElevator elevator.Elevator,
	elevatorToNetwork chan <-  elevator.Elevator,
	elevatorFromNetwork <- chan elevator.Elevator,
	peerUpdateCh <-chan p.PeerUpdate,
    peerTxEnable chan bool,
) {
	elevatorStuckCh := make(chan bool)
	stopedAtFloor := make(chan int)
	orderCh := make(chan elevator.Order, 100)
	requestUpdateCh := make(chan elevator.Requests, 100)
	elevators := make(map[string]elevator.Elevator)

	thisId = thisElevator.Id
	elevators[thisId] = thisElevator
	elevatorUpdateCh := make(chan elevator.Elevator)

    var connectedElevators []string
    var isStuck = false

    elevatorToNetwork <- thisElevator

	go elevatorcontroller.ListenAndServe(elevator.CopyElevator(thisElevator),
		requestUpdateCh,
		elevatorStuckCh,
		stopedAtFloor,
		orderCh,
		elevatorUpdateCh,
		true)

	for {
		select {
		case floor := <-stopedAtFloor:
			e, ok := elevators[thisId]
            if !ok {
            }
			e.CurrentFloor = floor
			e.Requests = requests.ClearAtCurrentFloor(floor, e)
			elevators[thisId] = e
                elevatorToNetwork <- elevators[thisId]

            if len(connectedElevators) == 0 {
                requestUpdateCh <- e.Requests
                elevator.SetHallLights(e)
            }

		case order := <-orderCh:
            e := AssignRequest(elevators, order)
                elevatorToNetwork <- e

            if len(connectedElevators) == 0 {
                requestUpdateCh <- e.Requests
                elevator.SetHallLights(e)
            }
            

        case isStuck = <-elevatorStuckCh:
            if isStuck {
                lostElevator := elevators[thisId]
                delete(elevators, lostElevator.Id)
                if len(connectedElevators) >= 0 && len(elevators) >= 1 {
                    reassignOrders(lostElevator, elevators, elevatorToNetwork)
                }
                elevators[thisId] = lostElevator
                peerTxEnable <- false
            } else {
                peerTxEnable <- true
            }

		case e := <-elevatorUpdateCh:
			elevators[e.Id] = e
            elevatorToNetwork <- e

		case e := <-elevatorFromNetwork:
            if isElevatorAlive(connectedElevators, e.Id) {
                if e.Id == thisId {
                    elevators[e.Id] = e
                    requestUpdateCh <- e.Requests
                }
			    elevators[e.Id] = e
                e.Requests = mergeAllHallReqs(elevators)
                elevator.SetHallLights(e)
            }

        case p := <- peerUpdateCh:
            fmt.Println(p.Peers)
            connectedElevators = p.Peers
            for _, lostId := range p.Lost {
                lostElevator := elevators[lostId]
			    delete(elevators, lostId)
                if len(connectedElevators) > 0 && len(elevators) >= 1 {
                    reassignOrders(lostElevator, elevators, elevatorToNetwork)
                }
                if lostId == thisId {
                    elevators[thisId] = lostElevator
                }
            }
		}
	}
}

func isElevatorAlive(elevators []string, elevatorId string) bool {
    for _, id := range elevators {
        if id == elevatorId {
            return true
        }
    }
    return false
}

// Reassigns the orders of the lostElevator
func reassignOrders(lostElevator elevator.Elevator, 
elevators map[string]elevator.Elevator,
elevatorToNetwork chan <- elevator.Elevator) {
	for f, req := range lostElevator.Requests.Up {
		if req {
            e :=  AssignRequest(elevators, elevator.Order{
				Type:    elevio.BT_HallUp,
				AtFloor: f,
			})
            elevatorToNetwork <- e
            lostElevator.Requests.Up[f] = false
            elevatorToNetwork <- lostElevator
            elevators[e.Id] = e
		}
	}
	for f, req := range lostElevator.Requests.Down {
		if req {
            e:= AssignRequest(elevators, elevator.Order{
				Type:    elevio.BT_HallDown,
				AtFloor: f,
			})
            elevatorToNetwork <- e
            lostElevator.Requests.Down[f] = false
            elevatorToNetwork <- lostElevator
            elevators[e.Id] = e
		}
	}
}


// Simulates execution of elevator and returns a cost 
// elevator with lowest cost should serve the request
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

// Asigns the request to the best available elevator
func AssignRequest(elevators map[string]elevator.Elevator,
    order elevator.Order) elevator.Elevator {
    if order.Type == elevio.BT_Cab {
        fmt.Println("CAB REQ")
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
	var currentDuration float32 = 0
	var bestDuration float32 = 1000
	var bestElevator string

    var keys []string
    for key := range elevators {
        keys = append(keys, key)
    }
    sort.Strings(keys)
    for _, id := range keys {
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
        Up: make([]bool, config.NFloors),
        Down: make([]bool, config.NFloors),
        ToFloor: make([]bool, config.NFloors),
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


