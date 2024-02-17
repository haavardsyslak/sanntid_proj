package elevator

import (
	"Driver-go/elevio"
	"fmt"
	"strings"
	"time"
)

type ElevatorState int

const (
	IDLE ElevatorState = iota
	MOVING
	DOOR_OPEN
)

type Order struct {
	Type    elevio.ButtonType
	AtFloor int
}

type Requests struct {
	Up      []bool
	Down    []bool
	ToFloor []bool
}

type Elevator struct {
	Dir          elevio.MotorDirection
	State        ElevatorState
	Requests     Requests
	MaxFloor     int
	MinFloor     int
	CurrentFloor int
}

func Init() int {
	// Init elevator
	// Run elevator to known floor
	elevio.Init("localhost:15657", 4)
	floor := elevio.GetFloor()
	if floor == -1 {
		floorSensCh := make(chan int)
		go elevio.PollFloorSensor(floorSensCh)
		return goToKnownFloor(floorSensCh)
	}
	return floor
}

func goToKnownFloor(floorSenseCh chan int) int {
	// go up for 4 sec -> reach a floor -> return floor
	// Else go down for 4 sec -> reach a floor? -> return floor
	// Else panic
	ticker := time.NewTicker(time.Second * 4)
	hasTimedOut := false
	GoUp()
	for {
		select {
		case floor := <-floorSenseCh:
			Stop()
			return floor
		case <-ticker.C:
			if !hasTimedOut {
				hasTimedOut = true
				GoDown()
			} else {
				Stop()
				panic("Elevator is stuck!")
			}
		}
	}
}

func PollElevatorIO(buttonCh chan elevio.ButtonEvent,
	florrSensCh chan int,
	stopButtonCh chan bool,
	obstructionSwitchCh chan bool) {
	go elevio.PollButtons(buttonCh)
	go elevio.PollFloorSensor(florrSensCh)
	go elevio.PollStopButton(stopButtonCh)
	go elevio.PollObstructionSwitch(obstructionSwitchCh)
}

func GoUp() {
	elevio.SetMotorDirection(elevio.MD_Up)
}

func GoDown() {
	elevio.SetMotorDirection(elevio.MD_Down)
}

func Stop() {
	elevio.SetMotorDirection(elevio.MD_Stop)
}

func ServeOrder(currentFloor int, toFloor int) elevio.MotorDirection {
	dir := get_elevator_dir(currentFloor, toFloor)
	elevio.SetMotorDirection(dir)
	return dir
}

func get_elevator_dir(floor int, toFloor int) elevio.MotorDirection {
	if toFloor == floor {
		return elevio.MD_Stop
	} else if toFloor > floor {
		return elevio.MD_Up
	} else {
		return elevio.MD_Down
	}
}

func OpenDoors(doorsOpenCh chan bool, obstructionCh chan bool) {
    fmt.Println("doors")
    ticker := time.NewTicker(500*time.Millisecond)
    counter := 0
	elevio.SetDoorOpenLamp(true)
    for {
        select{
            case obstructed := <- obstructionCh:
            if obstructed {
                ticker.Stop()
            } else {
                ticker.Reset(500 * time.Millisecond)
            }
        case <- ticker.C:
            fmt.Println("Ticker..")
            counter += 1
            if counter >= 6 {
                elevio.SetDoorOpenLamp(false)
                doorsOpenCh <- true
                return
            }
        }
    }
}

func SetLights(e Elevator) {
	for f := e.MinFloor; f <= e.MaxFloor; f++ {
		elevio.SetButtonLamp(elevio.BT_HallUp, f, e.Requests.Up[f])
		elevio.SetButtonLamp(elevio.BT_HallDown, f, e.Requests.Down[f])
		elevio.SetButtonLamp(elevio.BT_Cab, f, e.Requests.ToFloor[f])
	}
}

func PrintElevator(e Elevator) {
	fmt.Printf("Current floor: %d\n", e.CurrentFloor)
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
