package main

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	// "os/exec"
	"sanntid/conn"
	"sanntid/localelevator/elevator"

	// "time"
	"os/signal"
	"runtime"
	"sanntid/fakeorderassigner"
	"syscall"
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
    // Create a channel to receive OS signals
    sigCh := make(chan os.Signal, 1)
    // Register the channel to receive SIGINT signals
    signal.Notify(sigCh, syscall.SIGINT)

    // Start a goroutine to capture SIGINT signals
    go func() {
        <-sigCh // Block until a SIGINT signal is received
        fmt.Println("Received SIGINT signal. Printing stack trace...")
        printStackTraces() // Print stack traces of all goroutines
        os.Exit(1) // Exit the program
    }()

    fmt.Println("Id: ", id)

	peerUpdateCh := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)

	elevatorRxCh := make(chan conn.ElevatorPacket)
	elevatorTxCh := make(chan conn.ElevatorPacket)
	elevatorToNetworkCh := make(chan elevator.Elevator, 100)
	elevatorFromNetworkCh := make(chan elevator.Elevator)

	elevatorLostCh := make(chan string)
	// elevatorUpdateCh := make(chan elevator.Elevator)

	go peers.Transmitter(15647, id, peerTxEnable)
	go peers.Receiver(15647, peerUpdateCh)

	go bcast.Transmitter(16569, elevatorTxCh, peerTxEnable)
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
     
    go fakeorderassigner.HandleOrders(elevators[id],
        elevatorToNetworkCh,
        elevatorFromNetworkCh,
		elevatorLostCh,
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

func printStackTraces() {
    // Create a buffer to hold the stack trace
    buf := make([]byte, 1<<20)
    // Retrieve the stack trace of all goroutines
    runtime.Stack(buf, true)
    // Print the stack trace
    fmt.Println(string(buf))
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
