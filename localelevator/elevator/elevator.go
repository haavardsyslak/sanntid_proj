package elevator

import (
	"Driver-go/elevio"
	"fmt"
	"time"
    "sanntid/config"
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
    Id           string
}


func New(Id string) Elevator {
    return Elevator{
        Dir:   elevio.MD_Stop,
		State: IDLE,
		Requests: Requests{
			Up:      make([]bool, config.NumFloors),
			Down:    make([]bool, config.NumFloors),
			ToFloor: make([]bool, config.NumFloors),
		},
        MaxFloor: config.MaxFloor,
        MinFloor: config.MinFloor,
		CurrentFloor: 0,
        Id: Id,
	}
}

func Init(port int, stateIsKnown bool) int {
    // Init elevator hw
    elevio.Init(fmt.Sprintf("localhost:%d", port), config.NumFloors)
    // Run elevator to known floor if its state is unknown
    if !stateIsKnown {
        floor := elevio.GetFloor()
        if floor == -1 {
            floorSensCh := make(chan int)
            go elevio.PollFloorSensor(floorSensCh)
            return goToKnownFloor(floorSensCh)
        }
        return floor
    }
    return -1
}

func goToKnownFloor(floorSenseCh chan int) int {
	// go up for 4 sec -> reach a floor? -> return floor
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

func OpenDoors(doorsOpenCh chan bool, obstructionCh chan bool) {
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
            counter += 1
            if counter >= 6 {
                elevio.SetDoorOpenLamp(false)
                doorsOpenCh <- true
                return
            }
        }
    }
}

func SetHallLights(e Elevator) {
    for f := e.MinFloor; f <= e.MaxFloor; f++ {
		elevio.SetButtonLamp(elevio.BT_HallUp, f, e.Requests.Up[f])
		elevio.SetButtonLamp(elevio.BT_HallDown, f, e.Requests.Down[f])
    }
}

func SetCabLights(e Elevator) {
    for f := e.MinFloor; f <= e.MaxFloor; f++ {
        elevio.SetButtonLamp(elevio.BT_Cab, f, e.Requests.ToFloor[f])
    }
}

func CopyElevator(e Elevator) Elevator {
    requests := Requests{
        Up:      make([]bool, config.NumFloors),
        Down:    make([]bool, config.NumFloors),
        ToFloor: make([]bool, config.NumFloors),
    }
    for f := e.MinFloor; f <= e.MaxFloor; f++ {
        requests.Up[f] = e.Requests.Up[f]
        requests.Down[f] = e.Requests.Down[f]
        requests.ToFloor[f] = e.Requests.ToFloor[f]
    }
    return Elevator {
        Dir:   e.Dir,
		State: e.State,
		Requests: requests,
		MaxFloor:     e.MaxFloor,
		MinFloor:     e.MinFloor,
		CurrentFloor: e.CurrentFloor,
        Id: e.Id,
	}
}



