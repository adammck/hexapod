package legs

import (
	"fmt"
	"math"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/adammck/dynamixel/network"
	"github.com/adammck/hexapod"
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

	// Minimum distance which the desired foot position should be from its actual
	// position before a step should be taken to correct it.
	minStepDistance = 20.0

	// The time (in seconds) between each leg initialization. This should be as
	// low as possible, since it delays startup.
	initInterval = 0.25

	// The number of ticks per step, i.e. a single foot is lifted, moved to its
	// new position, and put down.
	ticksPerStep = 60

	// The number of ticks per step cycle, i.e. all legs have made a single
	// step, and returned to their original positions.
	ticksPerStepCycle = 240

	// The distance (in mm) which the hex can move per step cycle. Since each
	// foot only moves once per cycle, this could also be named stepDistance.
	stepCycleDistance = 80.0

	// Calc
	moveDistancePerTick = stepCycleDistance / ticksPerStepCycle
)

type Legs struct {
	Network *network.Network

	// The state that the legs are currently in.
	State        State
	stateCounter int
	stateTime    time.Time

	// ???
	Legs [6]*Leg

	// ???
	baseClearance float64

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

// stepHeights contains the Y position of each foot, at each tick. This is
// computed at startup and repeated while stepping.
var stepHeights [6][]float64

// Ratios of start->end position for both X/Z.
var stepMoves [6][]float64

func init() {
	p := ticksPerStepCycle / 4.0

	// TODO: Encapsulate this, along with the other curve properties, into some
	//       sort of Gait object, to make them pluggable.
	stepCurveCenter := [6]float64{
		0: p,
		1: p * 3,
		2: p,
		3: p * 3,
		4: p,
		5: p * 3,
	}

	for i := 0; i < 6; i += 1 {
		y := make([]float64, ticksPerStepCycle)
		xz := make([]float64, ticksPerStepCycle)

		for ii := 0; ii < ticksPerStepCycle; ii += 1 {
			fii := float64(ii)

			// Step height is a bell curve
			y[ii] = baseFootUp * math.Pow(2, -math.Pow((fii-stepCurveCenter[i])*((math.E*2)/ticksPerStep), 2))

			// Step movement ratio is a sine from 0 to 1
			if fii < (stepCurveCenter[i] - ticksPerStep/2) {
				xz[ii] = 0.0

			} else if fii > (stepCurveCenter[i] + ticksPerStep/2) {
				xz[ii] = 1.0

			} else {
				x := (fii - (stepCurveCenter[i] - ticksPerStep/2)) / ticksPerStep
				xz[ii] = 0.5 - (math.Cos(x*math.Pi) / 2)
			}
		}

		stepHeights[i] = y
		stepMoves[i] = xz
	}
}

func New(n *network.Network) *Legs {
	l := &Legs{
		Network:       n,
		State:         sDefault,
		baseClearance: sitDownClearance,
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
	l.stateTime = time.Now()
	l.State = s
}

// Clearance returns the distance (on the Y axis) which the body should be off
// the ground. This is mostly constant, but can be increased temporarily by
// pressing R2.
func (l *Legs) Clearance() float64 {
	return l.baseClearance
}

// StateDuration returns the duration since the hexapod entered the current
// state. This is a pretty fragile and crappy way of synchronizing things.
func (l *Legs) StateDuration() time.Duration {
	return time.Since(l.stateTime)
}

// homeFootPosition returns a vector in the WORLD coordinate space for the home
// position of the given leg.
func (l *Legs) homeFootPosition(leg *Leg, pos math3d.Vector3, rot float64) *math3d.Vector3 {
	r := utils.Rad(rot + leg.Angle)
	x := math.Cos(r) * stepRadius
	z := -math.Sin(r) * stepRadius
	return pos.Add(math3d.Vector3{X: x, Y: sitDownClearance, Z: z})
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
		state.Position.Y += 1
		state.TargetPosition.Y += 1
		if state.Position.Y >= standUpClearance {
			l.SetState(sStanding)
		}

	// Lower the clearance until the body is sitting on the ground.
	case sSitDown:
		state.Position.Y -= 1
		state.TargetPosition.Y -= 1
		if state.Position.Y <= sitDownClearance {
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

			// state.TargetPosition.X = 1000
			// state.TargetPosition.Z = 500

			vecToGoal := state.TargetPosition.Subtract(state.Position)
			distToGoal := vecToGoal.Magnitude()

			// Cap the distance we wil (attempt to) step at the max.
			distToStep := math.Min(distToGoal, stepCycleDistance)

			if distToStep > 5.0 {

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
		r := float64(l.stateCounter) / float64(ticksPerStepCycle)
		v := l.nextTarget.Subtract(l.lastPosition)

		state.Position = *l.lastPosition.Add(v.MultiplyByScalar(r))
		//log.Infof("pos=%s", state.Position)

		// Update the Y goal (distance from ground) of each foot according to
		// the precomputed map.
		for i, _ := range l.Legs {

			// TODO: Move this to an attribute-- maybe we can just store the
			//       last position and offsets? Do we even need the targets?
			vv := l.nextFeet[i].Subtract(l.lastFeet[i])
			vvv := vv.MultiplyByScalar(stepMoves[i][l.stateCounter-1])

			l.feet[i].Y = l.lastPosition.Y + stepHeights[i][l.stateCounter-1]
			l.feet[i].X = l.lastFeet[i].X + vvv.X
			l.feet[i].Z = l.lastFeet[i].Z + vvv.Z
		}

		// If this is the last tick in the cycle, reset the state such that the
		// next tick is #1.
		if l.stateCounter >= ticksPerStepCycle {
			l.SetState(sStepping)
		}

	default:
		return fmt.Errorf("unknown state: %#v", l.State)
	}

	if l.State != sHalt {
		// Update the position of each foot
		utils.Sync(l.Network, func() {
			for i, leg := range l.Legs {
				if leg.Initialized {
					pp := l.feet[i].MultiplyByMatrix44(state.Local())
					//log.Infof("%s world=%v, local=%v, dist=%0.2f", leg.Name, l.feet[i], pp, l.feet[i].Subtract(state.Position).Magnitude())
					leg.SetGoal(pp)
				}
			}
		})
	}

	return nil
}
