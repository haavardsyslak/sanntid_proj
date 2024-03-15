package elevator

import (
    "fmt"
    "strings"
    "Driver-go/elevio"
    "os"
    "os/exec"
)

func PrintElevator(elevatorStatus Elevator) {
    cmd := exec.Command("clear")
    cmd.Stdout = os.Stdout
    cmd.Run()
	fmt.Printf("Current floor: %d\n", elevatorStatus.CurrentFloor)
    fmt.Printf("ID: %s\n", elevatorStatus.Id)
	printState(elevatorStatus)
	printDirection(elevatorStatus.Dir)
	printRequests(elevatorStatus)
}

func printRequests(elevatorStatus Elevator) {
	up := make([]string, 0)
	down := make([]string, 0)
	to_floor := make([]string, 0)
	for f := elevatorStatus.MinFloor; f <= elevatorStatus.MaxFloor; f++ {
		if elevatorStatus.Requests.Up[f] {
			up = append(up, fmt.Sprintf("%d", f))
		}
		if elevatorStatus.Requests.Down[f] {
			down = append(down, fmt.Sprintf("%d", f))
		}
		if elevatorStatus.Requests.ToFloor[f] {
			to_floor = append(to_floor, fmt.Sprintf("%d", f))
		}
	}
	fmt.Println("Requests:")
	fmt.Printf("\tUp: %s\n", strings.Join(up, ","))
	fmt.Printf("\tDown: %s\n", strings.Join(down, ","))
	fmt.Printf("\tToFloor: %s\n", strings.Join(to_floor, ","))
}

func printState(elevatorStatus Elevator) {
	switch elevatorStatus.State {
	case IDLE:
		fmt.Println("State: Idle")
	case MOVING:
		fmt.Println("State: Moving")
	case DOOR_OPEN:
		fmt.Println("State: Door open")
	}
}

func printDirection(direction elevio.MotorDirection) {
	switch direction {
	case elevio.MD_Stop:
		fmt.Println("Dir: Stop")
	case elevio.MD_Up:
		fmt.Println("Dir: Up")
	case elevio.MD_Down:
		fmt.Println("Dir: Down")
	}
}
