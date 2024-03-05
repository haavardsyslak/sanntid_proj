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
	"sanntid/packethandler"
	"sanntid/localelevator/elevator"
    "sanntid/request_assigner"
)

func main() {
	var id string
    var port int

    parseCliArgs(&id, &port)
    
	peerUpdateCh := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)

	elevatorRxCh := make(chan packethandler.ElevatorPacket)
	elevatorTxCh := make(chan packethandler.ElevatorPacket)
	elevatorToNetworkCh := make(chan elevator.Elevator, 1000)
	elevatorFromNetworkCh := make(chan elevator.Elevator, 1000)

	go peers.Transmitter(15647, id, peerTxEnable)
	go peers.Receiver(15647, peerUpdateCh)

	go bcast.Transmitter(16569, elevatorTxCh)
	go bcast.Receiver(16569, elevatorRxCh)

	go packethandler.TransmitRecieve(elevatorToNetworkCh,
		elevatorFromNetworkCh,
		elevatorTxCh,
		elevatorRxCh,
    )

    e := elevator.New(id)
    networkElevator, err := recoverFromNetwork(id, elevatorFromNetworkCh)
    if err != nil {
        floor := elevator.Init(port, false)
        e.CurrentFloor = floor
    } else {
        elevator.Init(port, true)
        e = networkElevator
    }

    go request_assigner.DistributeRequests(e,
        elevatorToNetworkCh,
        elevatorFromNetworkCh,
		peerUpdateCh,
        peerTxEnable,
    )

    for {}
}

func parseCliArgs(id *string, port *int) {

	flag.StringVar(id, "id", "", "Peer ID")

    flag.IntVar(port, "port", 15657, "port of the hw")
	flag.Parse()
    fmt.Println("Port: ", port)

	if *id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		*id = fmt.Sprintf("peers-%s-%d", localIP, os.Getpid())
	}
}

func recoverFromNetwork(id string, 
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
