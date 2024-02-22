package main

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"flag"
	"fmt"
	"math/rand"
	"os"
	// "os/exec"
	"sanntid/conn"
	"sanntid/localelevator/elevator"
	// "time"
    "sanntid/fakeorderassigner"
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
		id = fmt.Sprint("peers-%s-%d", localIP, os.Getpid())
	}
    fakeorderassigner.Dummy()

	fmt.Println(id)

	peerUpdateCh := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)

	elevatorRxCh := make(chan conn.ElevatorPacket)
	elevatorTxCh := make(chan conn.ElevatorPacket)
	elevatorToNetworkCh := make(chan elevator.Elevator)
	elevatorFromNetworkCh := make(chan elevator.Elevator)

	elevatorLostCh := make(chan string)
	// elevatorUpdateCh := make(chan elevator.Elevator)

	go peers.Transmitter(15647, id, peerTxEnable)
	go peers.Receiver(15647, peerUpdateCh)

	go bcast.Transmitter(16569, elevatorTxCh, peerTxEnable)
	go bcast.Receiver(16569, elevatorRxCh)

	// ticker := time.NewTicker(100 * time.Millisecond)

	elevators := make(map[string]elevator.Elevator)
    e := elevator.New(id)
    floor := elevator.Init(port)
    e.CurrentFloor = floor
    elevators[id] = e

     

	go conn.TransmitRecieve(elevatorToNetworkCh,
		elevatorFromNetworkCh,
		elevatorLostCh,
		elevatorTxCh,
		elevatorRxCh,
    )
    go fakeorderassigner.HandleOrders(elevators[id],
        elevatorToNetworkCh,
        elevatorFromNetworkCh,
    )


    for {
        select {
        case p :=  <- peerUpdateCh:
            for _, peer := range p.Lost {
                elevatorLostCh <- peer
            }
            fmt.Println("Lost: ", p.Lost)
            fmt.Println("New: ", p.New)
        }
    }

}

// func OldLoop() {
// 	for {
// 		select {
// 		case p := <-peerUpdateCh:
// 			for _, peer := range p.Lost {
// 				elevatorLostCh <- peer
// 			}
//             fmt.Println("Lost: ", p.Lost)
//             fmt.Println("New: ", p.New)
// 		case <-ticker.C:
//             ticker.Reset(5 * time.Second)
// 			elevators[id] = InsertRandom(elevators[id])
// 			elevatorToNetworkCh <- elevators[id]
//
// 		case e := <-elevatorFromNetworkCh:
// 			elevators[e.Id] = e
//                 cmd := exec.Command("clear")
//                 cmd.Stdout = os.Stdout
//                 // cmd.Run()
//             for id, elev := range elevators {
//                 fmt.Println("ID: ", id)
//                 elevator.PrintElevator(elev)
//             }
// 		}
// 	}
// }

func InsertRandom(e elevator.Elevator) elevator.Elevator {
	e.Requests.Up = getRandomRequest(e.Requests.Up)
	e.Requests.Down= getRandomRequest(e.Requests.Down)
	e.Requests.ToFloor = getRandomRequest(e.Requests.ToFloor)
	return e
}

func getRandomRequest(slice []bool) []bool {
	// Generate a random boolean value
	randomValue := randBool()

	// Generate a random index within the bounds of the slice
	randomIndex := rand.Intn(len(slice))

    slice[randomIndex] = randomValue

    return slice
}

// randBool generates a random boolean value.
func randBool() bool {
	return rand.Intn(2) == 0
}
