package packethandler

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"sanntid/localelevator/elevator"
	"sync"
	"time"
)

type ElevatorPacket struct {
	SenderID       string
	Elevator       elevator.Elevator
	Checksum       string
	SequenceNumber uint32
}

var sequenceNumbers = make(map[string]uint32)
var sequenceNumbersMutex sync.Mutex

func HandleElevatorPackets(thisId string,
	elevatorUpdateToNetworkCh <-chan elevator.Elevator,
	elevatorUpdateFromNetworkCh chan<- elevator.Elevator,
	elevatorTxCh chan<- ElevatorPacket,
	elevatorRxCh <-chan ElevatorPacket,
	connectedPeersCh <-chan []string) {

	bcastTimer := time.NewTicker(5 * time.Millisecond)
	elevators := make(map[string]elevator.Elevator)
	var connectedPeers []string
	for {
		select {
		case e := <-elevatorUpdateToNetworkCh:
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

			elevators[packet.Elevator.Id] = elevator
			if shouldForwardElevatorPacket(packet, connectedPeers, thisId) {
				elevatorUpdateFromNetworkCh <- packet.Elevator
			}
		case connectedPeers = <-connectedPeersCh:
		}
	}
}

func shouldForwardElevatorPacket(packet ElevatorPacket, connectedPeers []string, thisId string) bool {
	return (packet.Elevator.Id == thisId || isElevatorAlive(connectedPeers, packet.Elevator.Id)) &&
		(packet.SenderID != thisId || len(connectedPeers) <= 1)
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
	} else {
		updateSequenceNumber(packet.Elevator.Id, packet.SequenceNumber)
	}
	return packet.Elevator, nil
}

func shouldScrapPacket(packet *ElevatorPacket, elevators map[string]elevator.Elevator) bool {
	sequenceNumbersMutex.Lock()
	currentSequenceNumber := sequenceNumbers[packet.Elevator.Id]
	sequenceNumbersMutex.Unlock()

	if !verifyChecksum(*packet) {
		return true
	}
	if packet.SequenceNumber < currentSequenceNumber {
		return true

	} else {
		return false
	}
}

func makeElevatorPacket(elevator elevator.Elevator, id string) ElevatorPacket {
	sequenceNumbersMutex.Lock()
	defer sequenceNumbersMutex.Unlock()
	packet := ElevatorPacket{
		SenderID:       id,
		SequenceNumber: sequenceNumbers[elevator.Id],
		Checksum:       "",
		Elevator:       elevator,
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
