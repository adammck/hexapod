package hexapod

import (
	"fmt"
	"github.com/adammck/dynamixel"
	"github.com/adammck/retroport"
	"github.com/jacobsa/go-serial/serial"
	"math"
	"time"
)

type State string

const (
	sInit     State = "sInit"
	sHalt     State = "sHalt"
	sStandUp  State = "sStandUp"
	sSitDown  State = "sSitDown"
	sStand    State = "sStand"
	sStepUp   State = "sStepUp"
	sStepOver State = "sStepOver"
	sStepDown State = "sStepDown"
)

type Hexapod struct {
	Network    *dynamixel.DynamixelNetwork
	Controller *retroport.SNES

	// The world coordinates of the center of the hexapod.
	CurrentPosition Point3d
	CurrentRotation float64

	// The state that the hexapod is currently in.
	State           State
	stateCounter    int

	// Set to true if the hexapod should shut down ASAP
	Halt            bool

	// ???
	TargetPosition  Point3d
	TargetRotation  float64
	StepRadius      float64
	Legs            [6]*Leg
}

//
// NewHexapod creates a new Hexapod object on the given Dynamixel network.
//
func NewHexapod(network *dynamixel.DynamixelNetwork) *Hexapod {
	return &Hexapod{
		Network:         network,
		CurrentPosition: Point3d{0, 0, 0},
		TargetPosition:  Point3d{0, 0, 0},
		CurrentRotation: 0.0,
		TargetRotation:  0.0,
		StepRadius:      220,
		Legs: [6]*Leg{

			// Points are the X/Y/Z offsets from the center of the top of the body to
			// the center of the coxa pivots.
			NewLeg(network, 10, "FL", NewPoint(-51.1769, -19, 98), -120), // Front Left  - 0
			NewLeg(network, 20, "FR", NewPoint(51.1769, -19, 98), -60),   // Front Right - 1
			NewLeg(network, 30, "MR", NewPoint(66, -19, 0), 0),           // Mid Right   - 2
			NewLeg(network, 40, "BR", NewPoint(51.1769, -19, -98), 60),   // Back Right  - 3
			NewLeg(network, 50, "BL", NewPoint(-51.1769, -19, -98), 120), // Back Left   - 4
			NewLeg(network, 60, "ML", NewPoint(-66, -19, 0), 180),        // Mid Left    - 5
		},
	}
}

// NewHexapodFromPortName creates a new Hexapod object by opening the given
// serial port with the default options. This only exists to reduce boilerplate
// in my development utils.
func NewHexapodFromPortName(portName string) (*Hexapod, error) {
	options := serial.OpenOptions{
		PortName:              portName,
		BaudRate:              1000000,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       0,
		InterCharacterTimeout: 100,
	}

	serial, openErr := serial.Open(options)
	if openErr != nil {
		return nil, openErr
	}

	network := dynamixel.NewNetwork(serial)
	flushErr := network.Flush()
	if flushErr != nil {
		return nil, flushErr
	}

	hexapod := NewHexapod(network)
	return hexapod, nil
}

func (h *Hexapod) SetState(s State) {
	fmt.Printf("State=%s\n", s)
	h.stateCounter = 0
	h.State = s
}

//
// Sync runs the given function while the network is in buffered mode, then
// initiates any movements at once by sending ACTION.
//
func (hexapod *Hexapod) Sync(f func()) {
	hexapod.Network.SetBuffered(true)
	f()
	hexapod.Network.SetBuffered(false)
	hexapod.Network.Action()
}

//
// SyncLegs runs the given function once for each leg while the network is in
// buffered mode, then initiates movements with ACTION. This is useful when
// resetting everything to a known state.
//
func (hexapod *Hexapod) SyncLegs(f func(leg *Leg)) {
	hexapod.Sync(func() {
		for _, leg := range hexapod.Legs {
			f(leg)
		}
	})
}

func (h *Hexapod) homeFootPosition(leg *Leg, o Point3d) *Point3d {
	//o := h.CurrentPosition
	r := rad(leg.Angle)
	//r := rad(leg.Angle - angle)
	x := math.Cos(r) * h.StepRadius
	z := -math.Sin(r) * h.StepRadius
	return &Point3d{o.X + x, o.Y - 20, o.Z + z}
}

// MainLoop watches for changes to the target position and rotation, and tries
// to apply it as gracefully as possible. Returns an exit code.
func (h *Hexapod) MainLoop() int {

	// Shorthand
	o := h.CurrentPosition
	r := h.CurrentRotation

	// Initial state
	h.State = sInit

	// settings
	mov := 2.0
	footUp := -40.0
	footDown := -80.0
	minStepDistance := 30.0
	initCount := 10
	stepUpCount := 2
	stepOverCount := 2
	stepDownCount := 2

	// World foot positions
	feet := [6]*Point3d{
		h.homeFootPosition(h.Legs[0], o),
		h.homeFootPosition(h.Legs[1], o),
		h.homeFootPosition(h.Legs[2], o),
		h.homeFootPosition(h.Legs[3], o),
		h.homeFootPosition(h.Legs[4], o),
		h.homeFootPosition(h.Legs[5], o),
	}

	// World positions of the NEXT foot position. These are nil if we're okay with
	// where the foot is now, but are set when the foot should be relocated.
	nextFeet := [6]*Point3d{
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	}

 	// move legs in groups of two, for stability
	legSets := [][]int{
		[]int{0, 3},
		[]int{1, 4},
		[]int{2, 5},
	}

	// Which legset are we currently stepping?
	sLegsIndex := 0

	for {
		h.stateCounter += 1
		//fmt.Printf("State=%s[%d]\n", h.State, h.stateCounter)

		if h.Controller.Up {
			o.Z += mov
		}

		if h.Controller.Down {
			o.Z -= mov
		}

		if h.Controller.Left {
			o.X -= mov
		}

		if h.Controller.Right {
			o.X += mov
		}

		if h.Controller.L {
			r -= mov
		}

		if h.Controller.R {
			r += mov
		}

		if h.Controller.Y {
			o.Y -= mov
		}

		if h.Controller.X {
			o.Y += mov
		}

		// At any time, pressing select terminates. This can also be set from
		// another goroutine, to handle e.g. SIGTERM.
		if h.Controller.Start || h.Halt {
			if h.State != sSitDown && h.State != sHalt {
				h.SetState(sSitDown)
			}
		}

		switch h.State {
		case sInit:

			// Initialize each servo
			if h.stateCounter == 1 {
				for _, leg := range h.Legs {
					for _, servo := range leg.Servos() {
						servo.SetStatusReturnLevel(1)
						servo.SetTorqueEnable(true)
						servo.SetMovingSpeed(512)
					}
				}
			}

			// Pause at this state for a while, then stand up.
			if h.stateCounter >= initCount {
				h.SetState(sStandUp)
			}

		case sHalt:
			for _, leg := range h.Legs {
				for _, servo := range leg.Servos() {
					servo.SetStatusReturnLevel(2)
					servo.SetTorqueEnable(false)
					servo.SetLed(false)
				}
			}

			return 0

		// After initializing, push the feet downloads to lift the hex off the
		// ground. This is to reduce torque on the joints when moving into the
		// initial stance.
		case sStandUp:
			for _, foot := range feet {
				foot.Y -= 2
			}

			// Once we've stood up, advance to the walking state.
			if feet[0].Y <= footDown {
				h.SetState(sStand)
			}

		case sSitDown:
			for _, foot := range feet {
				foot.Y += 2
			}

			if feet[0].Y >= footUp {
				h.SetState(sHalt)
			}

		// TODO: Move feet back to home positions when standing!
		case sStand:
			needsMove := false

			for i, _ := range h.Legs {
				a := h.homeFootPosition(h.Legs[i], o)
				a.Y = feet[i].Y
				if feet[i].Distance(*a) > minStepDistance {
					needsMove = true
				}
			}

			if needsMove {
				h.SetState(sStepUp)
			}

		case sStepUp:
			if h.stateCounter == 1 {
				for _, ii := range legSets[sLegsIndex] {
					h.Legs[ii].SetLED(true)
					feet[ii].Y = footUp
				}
			}

			// TODO: Project the next step position, rather than just moving it home
			//       every time. This will half (!!) the number of steps to move in a
			//       constant direciton.
			if h.stateCounter >= stepUpCount {
				for _, ii := range legSets[sLegsIndex] {
					nextFeet[ii] = h.homeFootPosition(h.Legs[ii], o)
				}

				h.SetState(sStepOver)
			}

		case sStepOver:
			if h.stateCounter == 1 {
				for _, ii := range legSets[sLegsIndex] {
					feet[ii].X = nextFeet[ii].X
					feet[ii].Z = nextFeet[ii].Z
				}

			}

			if h.stateCounter >= stepOverCount {
				h.SetState(sStepDown)
			}

		case sStepDown:
			if h.stateCounter == 1 {
				for _, ii := range legSets[sLegsIndex] {
					h.Legs[ii].SetLED(false)
					feet[ii].Y = footDown
				}
			}

			if h.stateCounter >= stepDownCount {
				sLegsIndex += 1

				if sLegsIndex >= len(legSets) {
					h.SetState(sStand)
					sLegsIndex = 0
				} else {
					h.SetState(sStepUp)
				}
			}

		default:
			fmt.Println("unknown state!")
			return 0
		}

		// Update the position of each foot
		h.Sync(func() {
			for i, leg := range h.Legs {
				pp := Point3d{feet[i].X - o.X, feet[i].Y - o.Y, feet[i].Z - o.Z}
				leg.SetGoal(pp)
			}
		})

		time.Sleep(1 * time.Millisecond)
	}
}

//
// Shutdown moves all servos to a hard-coded default position, then turns them
// off. This should be called when finished
//
func (hexapod *Hexapod) Shutdown() {
	for _, leg := range hexapod.Legs {
		for _, servo := range leg.Servos() {
			servo.SetTorqueEnable(true)
			servo.SetMovingSpeed(128)
		}
	}

	hexapod.SyncLegs(func(leg *Leg) {
		leg.Coxa.MoveTo(0)
		leg.Femur.MoveTo(-60)
		leg.Tibia.MoveTo(60)
		leg.Tarsus.MoveTo(60)
	})

	// TODO: Wait for servos to stop moving, instead of hard-coding a timer.
	wait(2000)
	hexapod.Relax()
}

func (hexapod *Hexapod) Relax() {
	for _, leg := range hexapod.Legs {
		for _, servo := range leg.Servos() {
			servo.SetTorqueEnable(false)
			servo.SetLed(false)
		}
	}
}

func wait(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func (hexapod *Hexapod) setLegs(c float64, f float64, t float64, tt float64) {
	hexapod.SyncLegs(func(leg *Leg) {
		leg.Coxa.MoveTo(c)
		leg.Femur.MoveTo(f)
		leg.Tibia.MoveTo(t)
		leg.Tarsus.MoveTo(tt)
	})
}
