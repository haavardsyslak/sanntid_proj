package packethandler

import (
	"errors"
	"fmt"
	"sanntid/localelevator/elevator"
	"sync"
	"time"
    "crypto/md5"
    "encoding/hex"
)

type ElevatorPacket struct {
    SenderID   string
	Elevator       elevator.Elevator
	Checksum       string
	SequenceNumber uint32
}

var sequenceNumbers = make(map[string]uint32)
var sequenceNumbersMutex sync.Mutex

func TransmitRecieve(thisId string,
    elevatorUpdateToNetworkCh <-chan elevator.Elevator,
	elevatorUpdateFromNetworkCh chan<- elevator.Elevator,
	elevatorTxCh chan <- ElevatorPacket,
	elevatorRxCh <- chan ElevatorPacket,
    connectedPeersCh <- chan []string) {

	bcastTimer := time.NewTicker(25 * time.Millisecond)
	elevators := make(map[string]elevator.Elevator)
    var connectedPeers []string
	for {
		select {
		case e := <- elevatorUpdateToNetworkCh:
			elevators[e.Id] = e
			incrementSequenceNumber(e.Id)

		case <-bcastTimer.C:
			for _, elevator := range elevators {
				packet := makeElevatorPacket(elevator, thisId)
				elevatorTxCh <- packet
			}
		case packet := <-elevatorRxCh:
			elevator, err := handleIncommingPacket(packet, elevators)
			if err != nil {
                continue
            }

            elevators[packet.Elevator.Id] = packet.Elevator
            if packet.SenderID == thisId && len(connectedPeers) <= 1 {
                elevatorUpdateFromNetworkCh <- elevator
            } else if packet.SenderID != thisId && isElevatorAlive(connectedPeers, packet.Elevator.Id) {
                elevatorUpdateFromNetworkCh <- elevator
            }

        case connectedPeers = <- connectedPeersCh:
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

func incrementSequenceNumber(Id string) {
	sequenceNumbersMutex.Lock()
	defer sequenceNumbersMutex.Unlock()
	sequenceNumbers[Id]++
}

func updateSequenceNumber(Id string, number uint32) {
	sequenceNumbersMutex.Lock()
	defer sequenceNumbersMutex.Unlock()
	sequenceNumbers[Id] = number
}

func handleIncommingPacket(packet ElevatorPacket, 
elevators map[string]elevator.Elevator) (elevator.Elevator, error) {
	if shouldScrapPacket(&packet, elevators) {
		return elevator.Elevator{}, errors.New("Packet scraped was scraped")
	}
	return packet.Elevator, nil
}

func shouldScrapPacket(packet *ElevatorPacket, elevators map[string]elevator.Elevator) bool {
	sequenceNumbersMutex.Lock()
	currentSequenceNumber := sequenceNumbers[packet.Elevator.Id]
	sequenceNumbersMutex.Unlock()
    if !verifyChecksum(*packet) {
        fmt.Println("Cheksum failed")
        return true 
    }
	if packet.SequenceNumber < currentSequenceNumber {
		return true

    // } else if packet.SequenceNumber == currentSequenceNumber {
    //     mergeRequests(packet, elevators[packet.Elevator.Id])
    //     return false
    } else {
		updateSequenceNumber(packet.Elevator.Id, packet.SequenceNumber)
		return false
	}
}

func mergeRequests(packet *ElevatorPacket, localElevator elevator.Elevator) elevator.Elevator {
    for f := localElevator.MinFloor; f <= localElevator.MaxFloor; f++ {
        packet.Elevator.Requests.Up[f] = localElevator.Requests.Up[f] || packet.Elevator.Requests.Up[f]
        packet.Elevator.Requests.Down[f] = localElevator.Requests.Down[f] || packet.Elevator.Requests.Down[f]
        packet.Elevator.Requests.ToFloor[f] = localElevator.Requests.ToFloor[f] || packet.Elevator.Requests.ToFloor[f]
    }
    return packet.Elevator 
}

func makeElevatorPacket(e elevator.Elevator, id string) ElevatorPacket {
	sequenceNumbersMutex.Lock()
	defer sequenceNumbersMutex.Unlock()
    packet := ElevatorPacket{
        SenderID: id,
		SequenceNumber: sequenceNumbers[e.Id],
		Checksum:       "",
		Elevator:       e,
	}
    packet.Checksum = computeChecksum(packet)
    return packet
}

func computeChecksum(packet ElevatorPacket) string {
	packetBytes := []byte(fmt.Sprintf("%v", packet.Elevator) + fmt.Sprintf("%d", packet.SequenceNumber))
    hash := md5.Sum(packetBytes)
    return hex.EncodeToString(hash[:])
}


func verifyChecksum(packet ElevatorPacket) bool {
    return computeChecksum(packet) == packet.Checksum
}
