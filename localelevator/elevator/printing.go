package elevator

import (
    "fmt"
    "strings"
    "Driver-go/elevio"
    "os"
    "os/exec"
)

func PrintElevator(e Elevator) {
    cmd := exec.Command("clear")
    cmd.Stdout = os.Stdout
    cmd.Run()
	fmt.Printf("Current floor: %d\n", e.CurrentFloor)
    fmt.Printf("ID: %s\n", e.Id)
	printState(e)
	printDir(e.Dir)
	printRequests(e)
}

func printRequests(e Elevator) {
	up := make([]string, 0)
	down := make([]string, 0)
	to_floor := make([]string, 0)
	for f := e.MinFloor; f <= e.MaxFloor; f++ {
		if e.Requests.Up[f] {
			up = append(up, fmt.Sprintf("%d", f))
		}
		if e.Requests.Down[f] {
			down = append(down, fmt.Sprintf("%d", f))
		}
		if e.Requests.ToFloor[f] {
			to_floor = append(to_floor, fmt.Sprintf("%d", f))
		}
	}
	fmt.Println("Requests:")
	fmt.Printf("\tUp: %s\n", strings.Join(up, ","))
	fmt.Printf("\tDown: %s\n", strings.Join(down, ","))
	fmt.Printf("\tToFloor: %s\n", strings.Join(to_floor, ","))
}

func printState(e Elevator) {
	switch e.State {
	case IDLE:
		fmt.Println("State: Idle")
	case MOVING:
		fmt.Println("State: Moving")
	case DOOR_OPEN:
		fmt.Println("State: Door open")
	}
}

func printDir(dir elevio.MotorDirection) {
	switch dir {
	case elevio.MD_Stop:
		fmt.Println("Dir: Stop")
	case elevio.MD_Up:
		fmt.Println("Dir: Up")
	case elevio.MD_Down:
		fmt.Println("Dir: Down")
	}
}
