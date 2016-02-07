package legs

import (
	"fmt"
	"math"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/adammck/dynamixel/network"
	"github.com/adammck/hexapod"
	"github.com/adammck/hexapod/components/legs/gait"
	"github.com/adammck/hexapod/math3d"
	"github.com/adammck/hexapod/utils"
)

type State string

const (
	sDefault  State = ""
	sInit     State = "sInit"
	sHalt     State = "sHalt"
	sStandUp  State = "sStandUp"
	sSitDown  State = "sSitDown"
	sStanding State = "sStanding"
	sStepping State = "sStepping"

	// The distance which the underside of the body should be raised off of the
	// ground when standing.
	defaultStandingClearance = 40.0

	// Distance (on the X/Z axis) from the origin to the point at which the feet
	// should be positioned. This isn't adjustable at runtime, because there are
	// very few valid settings.
	stepRadius = 220.0

	//
	stepDuration = time.Duration(1 * time.Second)

	// The number of ticks per step, i.e. a single foot is lifted, moved to its
	// new position, and put down.
	ticksPerStep = 60

	// The offset (on the Y axis) which feet should be moved to on the up step,
	// relative to the origin.
	stepHeight = 40.0

	// Minimum distance which the desired foot position should be from its actual
	// position before a step should be taken to correct it.
	minStepDistance = 5.0

	// The distance (in mm) which the hex can move per step cycle. This should
	// be determined experimentally; too high and the legs get tangled up.
	maxStepDistance = 80.0
)

type Legs struct {
	Network *network.Network

	// The state that the legs are currently in.
	State        State
	stateCounter int

	Gait gait.Gait

	// ???
	Legs [6]*Leg

	// The position (copied from the state) at the start of the current step
	// cycle. We keep track of this (in addition to the actual current position)
	// to avoid changing the target position mid-cycle.
	lastPosition math3d.Vector3

	// Target position at the end of the next step cycle. This is encapsulated
	// here (rather than in the state) because it's an implementation detail.
	nextTarget math3d.Vector3

	// Last known foot positions in the WORLD coordinate space. We must store them
	// in this space rather than the hexapod space, so they stay put when we move
	// the origin around.
	feet [6]*math3d.Vector3

	// Foot positions at the start of current step cycle.
	lastFeet [6]math3d.Vector3

	// World positions of the NEXT foot position. These are nil if we're okay with
	// where the foot is now, but are set when the foot should be relocated.
	nextFeet [6]*math3d.Vector3

	// The count (not index!) of the leg which we're currently initializing.
	// When it reaches six, we've finished initialzing.
	initCounter int
}

var log = logrus.WithFields(logrus.Fields{
	"pkg": "legs",
})

func New(n *network.Network, fps int) *Legs {
	l := &Legs{
		Network: n,
		State:   sDefault,
		Gait:    gait.TheGait(ticksPerStep),
		Legs: [6]*Leg{

			// Leg origins are relative to the hexapod origin, which is the X/Z
			// center of the body, level with the bottom of the coxas (which
			// protrude slightly below the body) on the Y axis.
			NewLeg(n, 40, "FL", math3d.MakeVector3(-61.167, 24, 98), -120), // Front Left  - 0
			NewLeg(n, 50, "FR", math3d.MakeVector3(61.167, 24, 98), -60),   // Front Right - 1
			NewLeg(n, 60, "MR", math3d.MakeVector3(66, 24, 0), 0),          // Mid Right   - 2
			NewLeg(n, 10, "BR", math3d.MakeVector3(61.167, 24, -98), 60),   // Back Right  - 3
			NewLeg(n, 20, "BL", math3d.MakeVector3(-61.167, 24, -98), 120), // Back Left   - 4
			NewLeg(n, 30, "ML", math3d.MakeVector3(-66, 24, 0), 179),       // Mid Left    - 5
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

	for i, _ := range l.feet {
		l.lastFeet[i] = *l.feet[i]
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
			log.Infof("pinging servo #%d", servo.ID)
			pingErr := servo.Ping()
			if pingErr != nil {
				return fmt.Errorf("error while pinging servo #%d: %s", servo.ID, pingErr)
			}
		}
	}

	// Initialize each servo.
	for _, leg := range l.Legs {
		for _, servo := range leg.Servos() {
			servo.SetTorqueEnable(true)
			servo.SetMovingSpeed(1024)
		}

		leg.Initialized = true
	}

	return nil
}

func (l *Legs) SetState(s State) {
	log.Infof("state=%v", s)
	l.stateCounter = 0
	l.State = s
}

// Clearance returns the distance (on the Y axis) which the body should be off
// the ground. This is mostly constant, but can be increased temporarily by
// pressing R2.
func (l *Legs) standingClearance() float64 {
	return defaultStandingClearance
}

// homeFootPosition returns a vector in the WORLD coordinate space for the home
// position of the given leg.
func (l *Legs) homeFootPosition(leg *Leg, pos math3d.Vector3, rot float64) *math3d.Vector3 {
	r := utils.Rad(rot + leg.Angle)
	x := math.Cos(r) * stepRadius
	z := -math.Sin(r) * stepRadius
	return pos.Add(math3d.Vector3{X: x, Y: 0.0, Z: z})
}

func (l *Legs) Tick(now time.Time, state *hexapod.State) error {
	l.stateCounter += 1

	switch l.State {
	case sDefault:
		l.SetState(sStandUp)

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
		if state.Shutdown {
			l.SetState(sSitDown)
			break
		}

		if l.stateCounter == 1 {
			state.TargetPosition.Y = l.standingClearance()
		}

		state.Position.Y += 1
		if state.Position.Y >= state.TargetPosition.Y {
			l.SetState(sStanding)
		}

	// Lower the clearance until the body is sitting on the ground.
	case sSitDown:
		if l.stateCounter == 1 {
			state.TargetPosition.Y = 0.0
		}

		state.Position.Y -= 1
		if state.Position.Y <= state.TargetPosition.Y {
			l.SetState(sHalt)
		}

	case sStanding:
		if state.Shutdown {
			l.SetState(sSitDown)
		} else { // if l.needsMove(state.Position, state.Rotation) {
			l.SetState(sStepping)
		}

	// TODO: This is the new needsMove()
	//case sStepWait:
	//	if state.TargetPosition.Subtract(state.Position).Magnitude() > 1 {
	//		l.SetState(sStepping)
	//	}

	case sStepping:

		// If this is the first tick in a step cycle, calculate the next target
		// position, which is simply the move distance in the direction of the
		// actual target position (which may be further away).
		if l.stateCounter == 1 {

			// Record current state
			l.lastPosition = state.Position
			for i, _ := range l.Legs {
				l.lastFeet[i] = *l.feet[i]
			}

			vecToGoal := state.TargetPosition.Subtract(state.Position)
			distToGoal := vecToGoal.Magnitude()

			// Cap the distance we wil (attempt to) step at the max.
			distToStep := math.Min(distToGoal, maxStepDistance)

			if distToStep > minStepDistance {

				// Calculate the target position for the origin.
				vecToStep := vecToGoal.Unit().MultiplyByScalar(distToStep)
				l.nextTarget = *l.lastPosition.Add(vecToStep)
				log.Infof("stepping from %v to %v", l.lastPosition, l.nextTarget)

			} else {
				l.nextTarget = l.lastPosition
				log.Infof("not stepping")
			}

			// Calculate the target position for each foot. Might be where they
			// already are, if we're not stepping.
			for i, leg := range l.Legs {
				l.nextFeet[i] = l.homeFootPosition(leg, l.nextTarget, state.Rotation)
			}
		}

		// Move continuously towards target. Note that we don't bother with the
		// rotation (for now), so the hex will walk sideways or backwards if the
		// target happens to be in that direction.
		r := float64(l.stateCounter) / float64(l.Gait.Length())
		v := l.nextTarget.Subtract(l.lastPosition)

		state.Position = *l.lastPosition.Add(v.MultiplyByScalar(r))

		// Update the Y goal (distance from ground) of each foot according to
		// the precomputed map.
		for i, _ := range l.Legs {
			f := l.Gait.Frame(i, l.stateCounter-1)

			// TODO: Move this to an attribute-- maybe we can just store the
			//       last position and offsets? Do we even need the targets?
			vv := l.nextFeet[i].Subtract(l.lastFeet[i])
			vvv := vv.MultiplyByScalar(f.XZ)

			l.feet[i].Y = stepHeight * f.Y
			l.feet[i].X = l.lastFeet[i].X + vvv.X
			l.feet[i].Z = l.lastFeet[i].Z + vvv.Z
		}

		// If this is the last tick in the cycle, reset the state such that the
		// next tick is #1.
		if l.stateCounter >= l.Gait.Length() {
			l.SetState(sStepping)
		}

	default:
		return fmt.Errorf("unknown state: %#v", l.State)
	}

	log.Infof("pos=%s", state.Position)

	if l.State != sHalt {
		// Update the position of each foot
		utils.Sync(l.Network, func() {
			for i, leg := range l.Legs {
				if leg.Initialized {
					pp := l.feet[i].MultiplyByMatrix44(state.Local())
					log.Infof("%s world=%v, local=%v, dist=%0.2f", leg.Name, l.feet[i], pp, l.feet[i].Subtract(state.Position).Magnitude())
					leg.SetGoal(pp)
				}
			}
		})
	}

	return nil
}
