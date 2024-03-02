package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"sanntid/conn"
	"sanntid/localelevator/elevator"
	//"sanntid/fakeorderassigner"
    "sanntid/request_assigner"
)

func main() {
	var id string
    var port int
	flag.StringVar(&id, "id", "", "Peer ID")

    flag.IntVar(&port, "port", 15657, "port of the hw")
	flag.Parse()
    fmt.Println("Port: ", port)

	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peers-%s-%d", localIP, os.Getpid())
	}

    fmt.Println("Id: ", id)

	peerUpdateCh := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)

	elevatorRxCh := make(chan conn.ElevatorPacket)
	elevatorTxCh := make(chan conn.ElevatorPacket)
	elevatorToNetworkCh := make(chan elevator.Elevator, 1000)
	elevatorFromNetworkCh := make(chan elevator.Elevator, 1000)

	// elevatorLostCh := make(chan string, 2)
	// elevatorUpdateCh := make(chan elevator.Elevator)

	go peers.Transmitter(15647, id, peerTxEnable)
	go peers.Receiver(15647, peerUpdateCh)

	go bcast.Transmitter(16569, elevatorTxCh)
	go bcast.Receiver(16569, elevatorRxCh)

	// ticker := time.NewTicker(100 * time.Millisecond)

	go conn.TransmitRecieve(elevatorToNetworkCh,
		elevatorFromNetworkCh,
		elevatorTxCh,
		elevatorRxCh,
    )

	elevators := make(map[string]elevator.Elevator)
    e := elevator.New(id)
    networkElevator, err := RecoverFromNetwork(id, elevatorFromNetworkCh)
    if err != nil {
        floor := elevator.Init(port, false)
        e.CurrentFloor = floor
        elevators[id] = e
    } else {
        elevator.Init(port, true)
        elevators[id] = networkElevator
    }
    
    fmt.Println(elevators[id])

    go request_assigner.HandleOrders(elevators[id],
        elevatorToNetworkCh,
        elevatorFromNetworkCh,
		peerUpdateCh,
        peerTxEnable,
    )
    


    for {}
}

func RecoverFromNetwork(id string, 
elevatorFromNetworkCh chan elevator.Elevator) (elevator.Elevator, error) {
    timeout := time.NewTicker(500 * time.Millisecond)   
    for {
        select {
        case <- timeout.C:
            return elevator.Elevator{}, errors.New("Unable to recover from network")
        case e := <- elevatorFromNetworkCh:
            if e.Id == id {
                return e, nil
            }
        }
    }
}
