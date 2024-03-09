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
	"sanntid/request_assigner"
	"time"
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

	go peers.Transmitter(15647, id, peerTxEnable)
	go peers.Receiver(15647, peerUpdateCh)

	go bcast.Transmitter(16569, elevatorTxCh)
	go bcast.Receiver(16569, elevatorRxCh)

	go packethandler.TransmitRecieve(id,
		elevatorToNetworkCh,
		elevatorFromNetworkCh,
		elevatorTxCh,
		elevatorRxCh,
		connectedPeersCh,
	)

	e := elevator.New(id)
	networkElevator, hasRecovered := recoverFromNetwork(id, elevatorFromNetworkCh)

	if hasRecovered {
		elevator.Init(port, true)
		e = networkElevator
	} else {
		floor := elevator.Init(port, false)
		e.CurrentFloor = floor
	}

	go request_assigner.DistributeRequests(e,
		elevatorToNetworkCh,
		elevatorFromNetworkCh,
		lostPeersCh,
		peerTxEnable,
	)

	for {
		select {
		case p := <-peerUpdateCh:
			fmt.Println(p.Peers)
			connectedPeersCh <- p.Peers
			lostPeersCh <- p.Lost
		}
	}
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
	elevatorFromNetworkCh chan elevator.Elevator) (elevator.Elevator, bool) {
	timeout := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-timeout.C:
			return elevator.Elevator{}, false
		case e := <-elevatorFromNetworkCh:
			if e.Id == id {
				return e, true
			}
		}
	}
}
