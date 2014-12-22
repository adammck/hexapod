package hexapod

import (
	"fmt"
	"github.com/adammck/dynamixel"
	"github.com/adammck/sixaxis"
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
	Controller *sixaxis.SA

	// The world coordinates of the center of the hexapod.
	// TODO (adammck): Store the rotation as Euler angles, and modify the
	//                 heading when rotating with L/R buttons. This is more
	//                 self-documenting than storing the heading as a float.
	Position Vector3
	Rotation float64

	// The state that the hexapod is currently in.
	State        State
	stateCounter int

	// Set to true if the hexapod should shut down ASAP
	Halt bool

	// ???
	StepRadius float64
	Legs       [6]*Leg

	// The time at which the voltage level was checked.
	lastVoltageCheck time.Time
}

// NewHexapod creates a new Hexapod object on the given Dynamixel network.
func NewHexapod(network *dynamixel.DynamixelNetwork) *Hexapod {
	return &Hexapod{
		Network:    network,
		Position:   Vector3{0, 0, 0},
		Rotation:   0.0,
		StepRadius: 220,
		Legs: [6]*Leg{

			// Points are the X/Y/Z offsets from the center of the top of the body to
			// the center of the coxa pivots.
			NewLeg(network, 10, "FL", MakeVector3(-51.1769, -19, 98), -120), // Front Left  - 0
			NewLeg(network, 20, "FR", MakeVector3(51.1769, -19, 98), -60),   // Front Right - 1
			NewLeg(network, 30, "MR", MakeVector3(66, -19, 0), 0),           // Mid Right   - 2
			NewLeg(network, 40, "BR", MakeVector3(51.1769, -19, -98), 60),   // Back Right  - 3
			NewLeg(network, 50, "BL", MakeVector3(-51.1769, -19, -98), 120), // Back Left   - 4
			NewLeg(network, 60, "ML", MakeVector3(-66, -19, 0), 180),        // Mid Left    - 5
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

// homeFootPosition returns a vector in the WORLD coordinate space for the home
// position of the given leg.
func (h *Hexapod) homeFootPosition(leg *Leg) *Vector3 {
	r := rad(leg.Angle)
	x := math.Cos(r) * h.StepRadius
	z := -math.Sin(r) * h.StepRadius
	return h.Position.Add(Vector3{x, -20, z})
}

// Projects a point in the World coordinate space into the coordinate space of
// given leg (by its index). This method is on the Hexapod rather than the Leg,
// to minimize the amount of state which we need to share with each leg.
func (h *Hexapod) Project(legIndex int, vec Vector3) Vector3 {
	hm := h.Legs[legIndex].Matrix()
	wm := MultiplyMatrices(hm, h.Local())
	return vec.MultiplyByMatrix44(*wm)
}

// NeedsVoltageCheck returns true if it's been a while since we checked the
// voltage level. The timeout is pretty arbitrary.
func (h *Hexapod) NeedsVoltageCheck() bool {
	return time.Since(h.lastVoltageCheck) > 2 * time.Second
}

// CheckVoltage fetches the voltage level of an arbitrary servo, and returns an
// error if it's below 9.6v.
func (h *Hexapod) CheckVoltage() error {
	v, err := h.Legs[0].Coxa.Voltage()
	h.lastVoltageCheck = time.Now()
	if err != nil {
		return err
	}

	fmt.Printf("voltage: %fv\n", v)

	if v < 9.6 {
		return fmt.Errorf("low voltage: %fv", v)
	}

	return nil
}

// World returns a matrix to transform a vector in the hexapod coordinate space
// into the world space.
func (h *Hexapod) World() Matrix44 {
	return *MakeMatrix44(h.Position, *MakeSingularEulerAngle(RotationHeading, h.Rotation))
}

// Local returns a matrix to transform a vector in the world coordinate space
// into the hexapod's space, taking into account its current position and
// rotation.
func (h *Hexapod) Local() Matrix44 {
	return h.World().Inverse()
}

// MainLoop watches for changes to the target position and rotation, and tries
// to apply it as gracefully as possible. Returns an exit code.
func (h *Hexapod) MainLoop() int {

	// Initial state
	h.State = sInit

	// settings
	legSetSize := 2
	sleepTime := 10 * time.Millisecond
	mov := 2.0
	footUp := -40.0
	footDown := -80.0
	minStepDistance := 20.0
	initCount := 10
	stepUpCount := 2
	stepOverCount := 2
	stepDownCount := 3

	// Foot positions in the WORLD coordinate space. We must store them in this
	// space rather than the hexapod space, so they stay put when we move the
	// origin around.
	feet := [6]*Vector3{
		h.homeFootPosition(h.Legs[0]),
		h.homeFootPosition(h.Legs[1]),
		h.homeFootPosition(h.Legs[2]),
		h.homeFootPosition(h.Legs[3]),
		h.homeFootPosition(h.Legs[4]),
		h.homeFootPosition(h.Legs[5]),
	}

	// World positions of the NEXT foot position. These are nil if we're okay with
	// where the foot is now, but are set when the foot should be relocated.
	nextFeet := [6]*Vector3{
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	}

	var legSets [][]int
	switch legSetSize {
	case 1:
		legSets = [][]int{
			[]int{0},
			[]int{1},
			[]int{2},
			[]int{3},
			[]int{4},
			[]int{5},
		}
	case 2:
		legSets = [][]int{
			[]int{0, 3},
			[]int{1, 4},
			[]int{2, 5},
		}
	case 3:
		legSets = [][]int{
			[]int{0, 2, 4},
			[]int{1, 3, 5},
		}
	default:
		fmt.Println("invalid legSetSize!")
		return 0
	}

	// Which legset are we currently stepping?
	sLegsIndex := 0

	for {


		h.stateCounter += 1
		//fmt.Printf("State=%s[%d]\n", h.State, h.stateCounter)

		if h.Controller.LeftStick.X != 0 {
			h.Position.X += (float64(h.Controller.LeftStick.X) / 127.0) * mov
		}

		if h.Controller.LeftStick.Y != 0 {
			h.Position.Z -= (float64(h.Controller.LeftStick.Y) / 127.0) * mov
		}

		if h.Controller.Up > 0 {
			h.Position.Y += 2
		}

		if h.Controller.Down > 0 {
			h.Position.Y -= 2
		}

		// TODO (adammck): This terminates the program, and shuts down the RPi.
		//                 Should we set the state to halt first?
		if h.Controller.Start && h.Controller.Select {
			return 1
		}

		// Check the voltage level regularly, and halt if it gets too low, to
		// avoid damaging the LiPo (again).
		if h.NeedsVoltageCheck() {
			err := h.CheckVoltage()
			if err != nil {
				fmt.Printf("voltage error: %s\n", err)
				//fmt.Printf("halting due to: %s\n", err)
				//h.SetState(sHalt)
			}
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
				a := h.homeFootPosition(h.Legs[i])
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
					nextFeet[ii] = h.homeFootPosition(h.Legs[ii])
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
				//pp := Vector3{feet[i].X - h.Position.X, feet[i].Y - h.Position.Y, feet[i].Z - h.Position.Z}
				pp := feet[i].MultiplyByMatrix44(h.Local())
				leg.SetGoal(pp)
			}
		})

		time.Sleep(sleepTime)
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
	time.Sleep(2000 * time.Millisecond)
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
