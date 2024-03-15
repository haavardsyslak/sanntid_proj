package main

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"flag"
	"fmt"
	"os"
	"sanntid/localelevator/elevator"
	"sanntid/packethandler"
	"sanntid/recovery"
	"sanntid/request_assigner"
)

func main() {
	var id string
	var port int

	parseCliArgs(&id, &port)

	peerUpdateCh := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)

	lostPeersCh := make(chan []string)
	connectedPeersCh := make(chan []string)

	elevatorRxCh := make(chan packethandler.ElevatorPacket)
	elevatorTxCh := make(chan packethandler.ElevatorPacket)
	elevatorToNetworkCh := make(chan elevator.Elevator, 1000)
	elevatorFromNetworkCh := make(chan elevator.Elevator, 1000)


	go packethandler.HandleElevatorPackets(id,
		elevatorToNetworkCh,
		elevatorFromNetworkCh,
		elevatorTxCh,
		elevatorRxCh,
		connectedPeersCh,
	)

	go peers.Transmitter(15647, id, peerTxEnable)
	go peers.Receiver(15647, peerUpdateCh)

	go bcast.Transmitter(16569, elevatorTxCh)
	go bcast.Receiver(16569, elevatorRxCh)

    go peers.PeerUpdateListener(peerUpdateCh, connectedPeersCh, lostPeersCh)


	thisElevator := elevator.New(id)
	networkElevator, hasRecovered := recovery.AttemptNetworkRecovery(id, elevatorFromNetworkCh)

	if hasRecovered {
		elevator.Init(port, true)
		thisElevator = networkElevator
	} else {
		floor := elevator.Init(port, false)
		thisElevator.CurrentFloor = floor
	}

	go request_assigner.DistributeRequests(thisElevator,
		elevatorToNetworkCh,
		elevatorFromNetworkCh,
		lostPeersCh,
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

