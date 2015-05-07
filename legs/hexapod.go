package legs

import (
	"fmt"
	"github.com/adammck/dynamixel"
	"github.com/adammck/hexapod/utils"
	"github.com/adammck/hexapod/math3d"
	"github.com/adammck/sixaxis"
	"github.com/jacobsa/go-serial/serial"
	"math"
	"strings"
	"time"
)

type State string

const (
	sDefault  State = ""
	sInit     State = "sInit"
	sHalt     State = "sHalt"
	sStandUp  State = "sStandUp"
	sSitDown  State = "sSitDown"
	sStand    State = "sStand"
	sStepUp   State = "sStepUp"
	sStepOver State = "sStepOver"
	sStepDown State = "sStepDown"

	// The number of seconds between voltage checks. These are pretty quick, but
	// not instant. Running at low voltage for too long will damage the battery,
	// so it should be checked pretty regularly.
	voltageCheckInterval = 5

	// The voltage at which the hexapod should forcibly shut down.
	minimumVoltage = 9.6

	// The offset (on the Y axis) which feet should be moved to on the up step,
	// relative to the origin.
	baseFootUp = 40.0

	// The offset (on the Y axis) which feet should be positioned at on the down
	// step (which is the default position when standing), relative to the
	// origin.
	baseFootDown = 0.0

	sitDownClearance = 0.0

	// Blah
	standUpClearance = 40.0

	// Distance (on the X/Z axis) from the origin to the point at which the feet
	// should be positioned. This isn't adjustable at runtime, because there are
	// very few valid settings.
	stepRadius = 220.0
)

type Hexapod struct {
	Network    *dynamixel.DynamixelNetwork
	Controller *sixaxis.SA

	// The world coordinates of the center of the hexapod.
	// TODO (adammck): Store the rotation as Euler angles, and modify the
	//                 heading when rotating with L/R buttons. This is more
	//                 self-documenting than storing the heading as a float.
	Position math3d.Vector3
	Rotation float64

	// The state that the hexapod is currently in.
	State        State
	stateCounter int
	stateTime    time.Time

	// Set to true if the hexapod should shut down ASAP
	Halt bool

	// ???
	Legs [6]*Leg

	// The time at which the voltage level was checked.
	lastVoltageCheck time.Time

	baseClearance float64
}

// NewHexapod creates a new Hexapod object on the given Dynamixel network.
func NewHexapod(network *dynamixel.DynamixelNetwork) *Hexapod {
	return &Hexapod{
		Network:       network,
		Position:      math3d.Vector3{0, 0, 0},
		Rotation:      0.0,
		State:         sDefault,
		baseClearance: sitDownClearance,
		Legs: [6]*Leg{

			// Leg origins are relative to the hexapod origin, which is the X/Z
			// center of the body, level with the bottom of the coxas (which
			// protrude slightly below the body) on the Y axis.
			NewLeg(network, 40, "FL", math3d.MakeVector3(-61.167, 24, 98), -120), // Front Left  - 0
			NewLeg(network, 50, "FR", math3d.MakeVector3(61.167, 24, 98), -60),   // Front Right - 1
			NewLeg(network, 60, "MR", math3d.MakeVector3(66, 24, 0), 1),          // Mid Right   - 2
			NewLeg(network, 10, "BR", math3d.MakeVector3(61.167, 24, -98), 60),   // Back Right  - 3
			NewLeg(network, 20, "BL", math3d.MakeVector3(-61.167, 24, -98), 120), // Back Left   - 4
			NewLeg(network, 30, "ML", math3d.MakeVector3(-66, 24, 0), 180),       // Mid Left    - 5
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
	// flushErr := network.Flush()
	// if flushErr != nil {
	// 	return nil, flushErr
	// }

	hexapod := NewHexapod(network)
	return hexapod, nil
}

// Ping checks that all servos are responding. This should be called immediately
// after instantiating a hexapod, to avoid weird behavior later. Returns nil if
// everything is responding, otherwise an error.
func (h *Hexapod) Ping() error {
	notResponding := make([]string, 0)

	for _, leg := range h.Legs {
		for _, servo := range leg.Servos() {
			fmt.Printf("Pinging #%d\n", servo.Ident)
			pingErr := servo.Ping()
			if pingErr != nil {
				notResponding = append(notResponding, string(servo.Ident))
			}
		}
	}

	if len(notResponding) != 0 {
		strings.Join(notResponding, ", ")
		return fmt.Errorf("servos not responding to ping: %s")
	}

	return nil
}

func (h *Hexapod) SetState(s State) {
	//fmt.Printf("State=%s\n", s)
	h.stateCounter = 0
	h.stateTime = time.Now()
	h.State = s
}

// stepUpPosition returns the height (on the Y axis) which a foot should reach
// when stepping up. This is generally static, but is increased while the L2
// trigger is pressed. This is pretty handy for stepping over obstacles.
func (h *Hexapod) stepUpPosition() float64 {
	return baseFootUp + ((float64(h.Controller.L2) / 255.0) * 100)
}

func (h *Hexapod) stepDownPosition() float64 {
	return baseFootDown
}

// Clearance returns the distance (on the Y axis) which the body should be off
// the ground. This is mostly constant, but can be increased temporarily by
// pressing R2.
func (h *Hexapod) Clearance() float64 {
	return h.baseClearance + ((float64(h.Controller.R2) / 255.0) * 100)
}

// StateDuration returns the duration since the hexapod entered the current
// state. This is a pretty fragile and crappy way of synchronizing things.
func (h *Hexapod) StateDuration() time.Duration {
	return time.Since(h.stateTime)
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
func (h *Hexapod) homeFootPosition(leg *Leg) *math3d.Vector3 {
	r := utils.Rad(h.Rotation + leg.Angle)
	x := math.Cos(r) * stepRadius
	z := -math.Sin(r) * stepRadius
	return h.Position.Add(math3d.Vector3{x, h.stepDownPosition(), z})
}

// Projects a point in the World coordinate space into the coordinate space of
// given leg (by its index). This method is on the Hexapod rather than the Leg,
// to minimize the amount of state which we need to share with each leg.
func (h *Hexapod) Project(legIndex int, vec math3d.Vector3) math3d.Vector3 {
	hm := h.Legs[legIndex].Matrix()
	wm := math3d.MultiplyMatrices(hm, h.Local())
	return vec.MultiplyByMatrix44(*wm)
}

// NeedsVoltageCheck returns true if it's been a while since we checked the
// voltage level. The timeout is pretty arbitrary.
func (h *Hexapod) NeedsVoltageCheck() bool {
	return time.Since(h.lastVoltageCheck) > (voltageCheckInterval * time.Second)
}

// CheckVoltage fetches the voltage level of an arbitrary servo, and returns an
// error if it's too low. In this case, the program should be terminated as soon
// as possible to preserve the battery.
func (h *Hexapod) CheckVoltage() error {
	v, err := h.Legs[0].Coxa.Voltage()
	h.lastVoltageCheck = time.Now()
	if err != nil {
		return err
	}

	fmt.Printf("voltage: %.2fv\n", v)

	if v < minimumVoltage {
		return fmt.Errorf("low voltage: %.2fv", v)
	}

	return nil
}

// World returns a matrix to transform a vector in the hexapod coordinate space
// into the world space.
func (h *Hexapod) World() math3d.Matrix44 {
	return *math3d.MakeMatrix44(h.Position, *math3d.MakeSingularEulerAngle(math3d.RotationHeading, h.Rotation))
}

// Local returns a matrix to transform a vector in the world coordinate space
// into the hexapod's space, taking into account its current position and
// rotation.
func (h *Hexapod) Local() math3d.Matrix44 {
	return h.World().Inverse()
}

// MainLoop watches for changes to the target position and rotation, and tries
// to apply it as gracefully as possible. Returns an exit code.
func (h *Hexapod) MainLoop() (exitCode int) {

	// settings
	legSetSize := 2
	sleepTime := 5 * time.Millisecond
	mov := 1.5
	minStepDistance := 20.0
	stepUpCount := 4
	stepOverCount := 4
	stepDownCount := 4

	// The maximum speed to rotate (i.e. when the right stick is fully pressed)
	// in degrees per loop.
	rotationSpeed := 0.8

	// Foot positions in the WORLD coordinate space. We must store them in this
	// space rather than the hexapod space, so they stay put when we move the
	// origin around.
	feet := [6]*math3d.Vector3{
		h.homeFootPosition(h.Legs[0]),
		h.homeFootPosition(h.Legs[1]),
		h.homeFootPosition(h.Legs[2]),
		h.homeFootPosition(h.Legs[3]),
		h.homeFootPosition(h.Legs[4]),
		h.homeFootPosition(h.Legs[5]),
	}

	// World positions of the NEXT foot position. These are nil if we're okay with
	// where the foot is now, but are set when the foot should be relocated.
	nextFeet := [6]*math3d.Vector3{
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	}

	// The order in which legs are initialized at startup. We start them one at
	// a time, rather than all at once, to reduce the load on the power supply.
	// When starting them all at once, quite often, the voltage drops low enough
	// to reset the RPi.
	initOrder := []int{0, 3, 1, 4, 2, 5}

	// The time (in seconds) between each leg initialization. This should be as
	// low as possible, since it delays startup.
	initInterval := 0.25

	// The count (not index!) of the leg which we're currently initializing.
	// When it reaches six, we've finished initialzing.
	initCounter := 0

	// Whether the hexapod should be prevented from moving its feet. It can't
	// walk when this is enable, only lean, so this is only useful for testing.
	dontMove := false

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
		return
	}

	// Which legset are we currently stepping?
	sLegsIndex := 0

	for _, leg := range h.Legs {
		for _, servo := range leg.Servos() {
			servo.SetStatusReturnLevel(1)
		}
	}

	// TODO (adammck): Move this to an actual method, once the feet targets are
	//                 available to the object (they're just vars in this method
	//                 right now).
	needsMove := func() bool {
		for i, _ := range h.Legs {
			a := h.homeFootPosition(h.Legs[i])
			a.Y = feet[i].Y
			if feet[i].Distance(*a) > minStepDistance {
				return true
			}
		}

		return false
	}

	for {

		h.stateCounter += 1
		//fmt.Printf("State=%s[%d]\n", h.State, h.stateCounter)

		// Rotate with the right stick
		if h.Controller.RightStick.X != 0 {
			h.Rotation += (float64(h.Controller.RightStick.X) / 127.0) * rotationSpeed
		}

		// How much the origin should move this frame. Default is zero, but this
		// it mutated (below) by the various buttons.
		vecMove := math3d.MakeVector3(0, 0, 0)

		if h.Controller.LeftStick.X != 0 {
			vecMove.X = (float64(h.Controller.LeftStick.X) / 127.0) * mov
		}

		if h.Controller.LeftStick.Y != 0 {
			vecMove.Z = (float64(-h.Controller.LeftStick.Y) / 127.0) * mov
		}

		// Move the origin up (away from the ground) with the dpad. This alters
		// the gait my keeping the body up in the air. It looks weird but works.
		if h.Controller.Up > 0 {
			h.baseClearance += 2
		}

		if h.Controller.Down > 0 {
			h.baseClearance -= 2
		}

		// Update the position, if it's changed.
		if !vecMove.Zero() {
			h.Position = vecMove.MultiplyByMatrix44(h.World())
		}

		dontMove = (h.Controller.Square > 0)

		// wat
		h.Position.Y = h.Clearance()

		// Check the voltage level regularly, and halt if it gets too low, to
		// avoid damaging the LiPo (again).
		if h.NeedsVoltageCheck() {
			err := h.CheckVoltage()
			if err != nil {
				fmt.Printf("halting due to: %s\n", err)
				h.SetState(sHalt)
				exitCode = 1
			}
		}

		// At any time, pressing select terminates. This can also be set from
		// another goroutine, to handle e.g. SIGTERM.
		if h.Controller.Start || h.Halt {
			if h.Controller.Select {
				exitCode = 1
			}
			if h.State != sSitDown && h.State != sHalt {
				h.SetState(sSitDown)
			}
		}

		switch h.State {
		case sDefault:
			h.SetState(sInit)

		case sInit:

			// Initialize one leg each second.
			if int(h.StateDuration().Seconds()/initInterval) > initCounter {

				// If we still have legs to initialize, do the next one.
				if initCounter < len(h.Legs) {
					leg := h.Legs[initOrder[initCounter]]

					for _, servo := range leg.Servos() {
						servo.SetTorqueEnable(true)
						servo.SetMovingSpeed(1024)
					}

					leg.Initialized = true
					initCounter += 1

				} else {
					// No more legs to initialize, so advance to the next state.
					// We wait until the next initCounter before advancing, to
					// give the last leg a second to start.
					h.SetState(sStandUp)
				}
			}

		case sHalt:
			for _, leg := range h.Legs {
				for _, servo := range leg.Servos() {
					servo.SetStatusReturnLevel(2)
					servo.SetTorqueEnable(false)
					servo.SetLed(false)
				}
			}

			return

		// After initialzation, raise the clearance to lift the body off the
		// ground, into the standing position.
		case sStandUp:
			h.baseClearance += 2
			if h.baseClearance >= standUpClearance {
				h.SetState(sStand)
			}

		// Before halting, lower the clearance until the body is sitting on the
		// ground.
		case sSitDown:
			h.baseClearance -= 2
			if h.baseClearance <= sitDownClearance {
				h.SetState(sHalt)
			}

		case sStand:
			if !dontMove && needsMove() {
				h.SetState(sStepUp)
			}

		case sStepUp:
			if h.stateCounter == 1 {
				for _, ii := range legSets[sLegsIndex] {
					feet[ii].Y = h.stepUpPosition()
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
					feet[ii].Y = h.stepDownPosition()
				}
			}

			if h.stateCounter >= stepDownCount {
				sLegsIndex += 1

				if sLegsIndex >= len(legSets) {
					sLegsIndex = 0

					// If we still need to move, switch back to StepUp.
					// Otherwise, stand still.
					if needsMove() {
						h.SetState(sStepUp)
					} else {
						h.SetState(sStand)
					}

				} else {
					h.SetState(sStepUp)
				}
			}

		default:
			fmt.Println("unknown state!")
			return
		}

		// Update the position of each foot
		h.Sync(func() {
			for i, leg := range h.Legs {
				if leg.Initialized {
					pp := feet[i].MultiplyByMatrix44(h.Local())
					leg.SetGoal(pp)
				}
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
