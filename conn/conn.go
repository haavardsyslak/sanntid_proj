package conn

import (
	"Network-go/network/peers"
	"sanntid/localelevator/elevator"
	"time"
)

type elevatorPacket struct {
	Elevator       elevator.Elevator
	Checksum       string
	SequenceNumber uint32
}


func UpdateElevators(elevatorUpdateCh chan elevator.Elevator) {
	elevators := make(map[string]elevator.Elevator)
	for {
		select {
		case e := <-elevatorUpdateCh:
			elevators[e.Id] = e
		}
	}
}

func TransmittRecieve(elevatorUpdateCh chan elevator.Elevator,
	elevatorLostCh chan string,
	elevatorTxCh chan elevatorPacket,
    elevatorRxCh chan elevatorPacket) {

	bcastTimer := time.NewTicker(100 * time.Millisecond)
	elevators := make(map[string]elevator.Elevator)
	for {
		select {
		case e := <-elevatorUpdateCh:
			elevators[e.Id] = e
	        
        case id := <- elevatorLostCh:
            delete(elevators, id)
		case <-bcastTimer.C:
			for _, elevator := range elevators {
				elevatorTxCh <- makeElevatorPacket(elevator)
			}
        case packet := <- elevatorRxCh:
            HandleIncommingPacket(packet)
		}
	}
}

func HandleIncommingPacket(packet elevatorPacket) {
    if shouldScrapPacket(packet) {
        return nil 
    }
    return packet.Elevator 
}

func makeElevatorPacket(e elevator.Elevator) elevatorPacket {
	return elevatorPacket{
		SequenceNumber: 0,
		Checksum:       "",
		Elevator:       e,
	}
}

func computeChecksum(packet elevatorPacket) string {
	//packetBytes := []byte(fmt.Sprint("%v", packet.Elevator) + fmt.Sprintf("%d"))
	return ""
}
