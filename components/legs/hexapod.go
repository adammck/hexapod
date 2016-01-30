package legs

import (
	"fmt"
	"github.com/adammck/dynamixel/network"
	"github.com/adammck/hexapod"
	"github.com/adammck/hexapod/math3d"
	"github.com/adammck/hexapod/utils"
	"math"
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

	// The offset (on the Y axis) which feet should be moved to on the up step,
	// relative to the origin.
	baseFootUp = 40.0

	// The offset (on the Y axis) which feet should be positioned at on the down
	// step (which is the default position when standing), relative to the
	// origin.
	baseFootDown = 0.0

	// Blah
	sitDownClearance = 0.0
	standUpClearance = 40.0

	// Distance (on the X/Z axis) from the origin to the point at which the feet
	// should be positioned. This isn't adjustable at runtime, because there are
	// very few valid settings.
	stepRadius = 220.0

	// The number of legs to move at once.
	legSetSize = 2

	// Minimum distance which the desired foot position should be from its actual
	// position before a step should be taken to correct it.
	minStepDistance = 20.0

	// The number of ticks which should be spent in each state.
	// TODO: Replace these with durations, ticks are variable now.
	stepUpCount   = 4
	stepOverCount = 4
	stepDownCount = 4

	// The time (in seconds) between each leg initialization. This should be as
	// low as possible, since it delays startup.
	initInterval = 0.25
)

type Legs struct {
	hexapod *hexapod.Hexapod
	Network *network.Network

	// The state that the legs are currently in.
	State        State
	stateCounter int
	stateTime    time.Time

	// ???
	Legs [6]*Leg

	// ???
	baseClearance float64

	// The order in which legs are initialized at startup. We start them one at
	// a time, rather than all at once, to reduce the load on the power supply.
	// When starting them all at once, quite often, the voltage drops low enough
	// to reset the RPi.
	//
	// TODO: This probably isn't the case any more, now that I have a proper power
	//       supply. Remove this?
	//
	initOrder []int

	// Last known foot positions in the WORLD coordinate space. We must store them
	// in this space rather than the hexapod space, so they stay put when we move
	// the origin around.
	feet [6]*math3d.Vector3

	// World positions of the NEXT foot position. These are nil if we're okay with
	// where the foot is now, but are set when the foot should be relocated.
	nextFeet [6]*math3d.Vector3

	// Whether the hexapod should be prevented from moving its feet. It can't
	// walk when this is enable, only lean, so this is only useful for testing.
	dontMove bool

	// The count (not index!) of the leg which we're currently initializing.
	// When it reaches six, we've finished initialzing.
	initCounter int

	// Which legset are we currently stepping?
	sLegsIndex int
}

func New(n *network.Network) *Legs {
	l := &Legs{
		Network:       n,
		State:         sDefault,
		baseClearance: sitDownClearance,
		initOrder:     []int{0, 3, 1, 4, 2, 5},
		Legs: [6]*Leg{

			// Leg origins are relative to the hexapod origin, which is the X/Z
			// center of the body, level with the bottom of the coxas (which
			// protrude slightly below the body) on the Y axis.
			NewLeg(n, 40, "FL", math3d.MakeVector3(-61.167, 24, 98), -120), // Front Left  - 0
			NewLeg(n, 50, "FR", math3d.MakeVector3(61.167, 24, 98), -60),   // Front Right - 1
			NewLeg(n, 60, "MR", math3d.MakeVector3(66, 24, 0), 1),          // Mid Right   - 2
			NewLeg(n, 10, "BR", math3d.MakeVector3(61.167, 24, -98), 60),   // Back Right  - 3
			NewLeg(n, 20, "BL", math3d.MakeVector3(-61.167, 24, -98), 120), // Back Left   - 4
			NewLeg(n, 30, "ML", math3d.MakeVector3(-66, 24, 0), 180),       // Mid Left    - 5
		},
	}

	// TODO: We're initializing the position to zero here, but that prevents us
	//       from settings the actual location of the hex at boot. Should we
	//       provide the initial state to these constructors?
	l.feet = [6]*math3d.Vector3{
		l.homeFootPosition(l.Legs[0], math3d.ZeroVector3, 0),
		l.homeFootPosition(l.Legs[1], math3d.ZeroVector3, 0),
		l.homeFootPosition(l.Legs[2], math3d.ZeroVector3, 0),
		l.homeFootPosition(l.Legs[3], math3d.ZeroVector3, 0),
		l.homeFootPosition(l.Legs[4], math3d.ZeroVector3, 0),
		l.homeFootPosition(l.Legs[5], math3d.ZeroVector3, 0),
	}

	return l
}

// Boot pings all servos, and returns an error if any of them fail to respond.
func (l *Legs) Boot() error {

	// Don't bother sending ACKs for writes. We must do this first, to ensure that
	// the servos are in the expected state before sending other commands.
	for _, leg := range l.Legs {
		for _, servo := range leg.Servos() {
			setStatusErr := servo.SetStatusReturnLevel(1)
			if setStatusErr != nil {
				return fmt.Errorf("error while setting status return level of servo #%d: %s", servo.ID, setStatusErr)
			}
		}
	}

	// Ping all servos to ensure they're all alive.
	for _, leg := range l.Legs {
		for _, servo := range leg.Servos() {
			fmt.Printf("Pinging #%d\n", servo.ID)
			pingErr := servo.Ping()
			if pingErr != nil {
				return fmt.Errorf("error while pinging servo #%d: %s", servo.ID, pingErr)
			}
		}
	}

	return nil
}

func (l *Legs) SetState(s State) {
	l.stateCounter = 0
	l.stateTime = time.Now()
	l.State = s
}

// stepUpPosition returns the height (on the Y axis) which a foot should reach
// when stepping up. This is generally static, but is increased while the L2
// trigger is pressed. This is pretty handy for stepping over obstacles.
func (l *Legs) stepUpPosition() float64 {
	//return baseFootUp + ((float64(h.Controller.L2) / 255.0) * 100)
	return baseFootUp
}

func (l *Legs) stepDownPosition() float64 {
	return baseFootDown
}

// Clearance returns the distance (on the Y axis) which the body should be off
// the ground. This is mostly constant, but can be increased temporarily by
// pressing R2.
func (l *Legs) Clearance() float64 {
	//return h.baseClearance + ((float64(h.Controller.R2) / 255.0) * 100)
	return l.baseClearance
}

// StateDuration returns the duration since the hexapod entered the current
// state. This is a pretty fragile and crappy way of synchronizing things.
func (l *Legs) StateDuration() time.Duration {
	return time.Since(l.stateTime)
}

//
// Sync runs the given function while the network is in buffered mode, then
// initiates any movements at once by sending ACTION.
//
func (l *Legs) Sync(f func()) {
	l.Network.SetBuffered(true)
	f()
	l.Network.SetBuffered(false)
	l.Network.Action()
}

//
// SyncLegs runs the given function once for each leg while the network is in
// buffered mode, then initiates movements with ACTION. This is useful when
// resetting everything to a known state.
//
func (l *Legs) SyncLegs(f func(leg *Leg)) {
	l.Sync(func() {
		for _, leg := range l.Legs {
			f(leg)
		}
	})
}

// homeFootPosition returns a vector in the WORLD coordinate space for the home
// position of the given leg.
func (l *Legs) homeFootPosition(leg *Leg, pos math3d.Vector3, rot float64) *math3d.Vector3 {
	r := utils.Rad(rot + leg.Angle)
	x := math.Cos(r) * stepRadius
	z := -math.Sin(r) * stepRadius
	return pos.Add(math3d.Vector3{X: x, Y: sitDownClearance, Z: z})
}

// Projects a point in the World coordinate space into the coordinate space of
// given leg (by its index). This method is on the Hexapod rather than the Leg,
// to minimize the amount of state which we need to share with each leg.
func (l *Legs) Project(legIndex int, vec math3d.Vector3) math3d.Vector3 {
	hm := l.Legs[legIndex].Matrix()
	wm := math3d.MultiplyMatrices(hm, l.hexapod.Local())
	return vec.MultiplyByMatrix44(*wm)
}

func (l *Legs) legSet() [][]int {
	switch legSetSize {
	case 1:
		return [][]int{
			[]int{0},
			[]int{1},
			[]int{2},
			[]int{3},
			[]int{4},
			[]int{5},
		}
	case 2:
		return [][]int{
			[]int{0, 3},
			[]int{1, 4},
			[]int{2, 5},
		}
	case 3:
		return [][]int{
			[]int{0, 2, 4},
			[]int{1, 3, 5},
		}
	default:
		panic("invalid legSetSize!")
	}
}

// Returns true if any of the feet are of sufficient distance from their desired
// positions that we need to take a step.
func (l *Legs) needsMove(pos math3d.Vector3, rot float64) bool {
	for i, _ := range l.Legs {
		a := l.homeFootPosition(l.Legs[i], pos, rot)
		a.Y = l.feet[i].Y
		if l.feet[i].Distance(*a) > minStepDistance {
			return true
		}
	}

	return false
}

func (l *Legs) Tick(now time.Time, state *hexapod.State) error {
	l.stateCounter += 1

	switch l.State {
	case sDefault:
		l.SetState(sInit)

	case sInit:

		// Initialize one leg each second.
		if int(l.StateDuration().Seconds()/initInterval) > l.initCounter {

			// If we still have legs to initialize, do the next one.
			if l.initCounter < len(l.Legs) {
				leg := l.Legs[l.initOrder[l.initCounter]]

				for _, servo := range leg.Servos() {
					servo.SetTorqueEnable(true)
					servo.SetMovingSpeed(1024)
				}

				leg.Initialized = true
				l.initCounter += 1

			} else {
				// No more legs to initialize, so advance to the next state.
				// We wait until the next initCounter before advancing, to
				// give the last leg a second to start.
				l.SetState(sStandUp)
			}
		}

	// TODO: Remove this state? Maybe we should add a separate interface method
	//       which is called when the parent wants to shut everything down.
	case sHalt:
		if l.stateCounter == 1 {
			for _, leg := range l.Legs {
				for _, servo := range leg.Servos() {
					servo.SetStatusReturnLevel(2)
					servo.SetTorqueEnable(false)
					servo.SetLED(false)
				}
			}
		}

	// After initialzation, raise the clearance to lift the body off the
	// ground, into the standing position.
	case sStandUp:
		state.Position.Y += 2
		if state.Position.Y >= standUpClearance {
			l.SetState(sStand)
		}

	// Lower the clearance until the body is sitting on the ground.
	case sSitDown:
		state.Position.Y -= 2
		if state.Position.Y <= sitDownClearance {
			l.SetState(sHalt)
		}

	case sStand:
		if state.Shutdown {
			l.SetState(sSitDown)
		} else if !l.dontMove && l.needsMove(state.Position, state.Rotation) {
			l.SetState(sStepUp)
		}

	case sStepUp:
		if state.Shutdown {
			l.SetState(sSitDown)
		} else {
			if l.stateCounter == 1 {
				for _, ii := range l.legSet()[l.sLegsIndex] {
					l.feet[ii].Y = l.stepUpPosition()
				}
			}

			// TODO: Project the next step position, rather than just moving it home
			//       every time. This will half (!!) the number of steps to move in a
			//       constant direciton.
			if l.stateCounter >= stepUpCount {
				for _, ii := range l.legSet()[l.sLegsIndex] {
					l.nextFeet[ii] = l.homeFootPosition(l.Legs[ii], state.Position, state.Rotation)
				}

				l.SetState(sStepOver)
			}
		}

	case sStepOver:
		if l.stateCounter == 1 {
			for _, ii := range l.legSet()[l.sLegsIndex] {
				l.feet[ii].X = l.nextFeet[ii].X
				l.feet[ii].Z = l.nextFeet[ii].Z
			}

		}

		if l.stateCounter >= stepOverCount {
			l.SetState(sStepDown)
		}

	case sStepDown:
		if l.stateCounter == 1 {
			for _, ii := range l.legSet()[l.sLegsIndex] {
				l.feet[ii].Y = l.stepDownPosition()
			}
		}

		if l.stateCounter >= stepDownCount {
			l.sLegsIndex += 1

			if l.sLegsIndex >= len(l.legSet()) {
				l.sLegsIndex = 0

				// If we still need to move, switch back to StepUp.
				// Otherwise, stand still.
				if l.needsMove(state.Position, state.Rotation) {
					l.SetState(sStepUp)
				} else {
					l.SetState(sStand)
				}

			} else {
				l.SetState(sStepUp)
			}
		}

	default:
		return fmt.Errorf("unknown state: %#v", l.State)
	}

	if l.State != sHalt {
		// Update the position of each foot
		l.Sync(func() {
			for i, leg := range l.Legs {
				if leg.Initialized {
					pp := l.feet[i].MultiplyByMatrix44(l.hexapod.Local())
					leg.SetGoal(pp)
				}
			}
		})
	}

	return nil
}
