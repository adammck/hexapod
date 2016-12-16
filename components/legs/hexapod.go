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
	sHalt     State = "sHalt"
	sStandUp  State = "sStandUp"
	sSitDown  State = "sSitDown"
	sStanding State = "sStanding"
	sStepping State = "sStepping"

	moveSpeedSlow = 32
	moveSpeedFast = 1023

	// The distance which the underside of the body should be raised off of the
	// ground when standing.
	defaultStandingClearance = 40.0

	// Distance (on the X/Z axis) from the origin to the point at which the feet
	// should be positioned. This isn't adjustable at runtime, because there are
	// very few valid settings.
	stepRadius = 220.0

	// The number of ticks per step, i.e. a single foot is lifted, moved to its
	// new position, and put down.
	ticksPerStep = 24

	// The offset (on the Y axis) which feet should be moved to on the up step,
	// relative to the origin.
	stepHeight = 40.0

	// Minimum distance which the desired foot position should be from its actual
	// position before a step should be taken to correct it.
	minStepDistance = 5.0

	// The distance (in mm) which the hex can move per step cycle. This should
	// be determined experimentally; too high and the legs get tangled up.
	maxStepDistance = 80.0

	// The time to transition from standing to sitting (and back).
	sitStandDuration = 1 * time.Second
)

type Legs struct {
	Network *network.Network

	// The state that the legs are currently in.
	State        State
	stateCounter int
	stateTime    time.Time

	Gait gait.Gait

	// ???
	Legs [6]*Leg

	// The pose (copied from the state) at the start of the current step cycle.
	// We use this to calculate the pose for each intra-cycle frame.
	lastPose math3d.Pose

	// Target pose at the end of the next step cycle. This is calculated (from
	// the state.Target) at the start of each cycle, to avoid moving it around
	// mid-cycle. This is encapsulated here (rather than in the state) because
	// it's an implementation detail of the legs.
	target math3d.Pose

	// Last known foot positions in the WORLD coordinate space. We must store
	// them in this space rather than the hexapod space, so they stay put when
	// we move the origin around.
	feet [6]math3d.Vector3

	// Foot positions at the start of current step cycle.
	lastFeet [6]math3d.Vector3

	// World positions of the NEXT foot position. These are nil if we're okay with
	// where the foot is now, but are set when the foot should be relocated.
	nextFeet [6]math3d.Vector3

	// The count (not index!) of the leg which we're currently initializing.
	// When it reaches six, we've finished initialzing.
	initCounter int

	// The number of mm to move downwards per tick, when sitting down. This is
	// only used while in the sSitDown state.
	sitDownSpeed float64
}

var log = logrus.WithFields(logrus.Fields{
	"pkg": "legs",
})

func New(n *network.Network) *Legs {
	l := &Legs{
		Network: n,
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

	// Initialize the position of each foot to (roughly) its current position,
	// by fetching the angle of each servo. They could be all over the place,
	// because we're booting up.
	for i, leg := range l.Legs {
		v, err := leg.PresentPosition()

		// ಠ_ಠ
		if err != nil {
			panic(err)
		}

		log.Infof("%s initialized at %v", leg.Name, v)
		l.lastFeet[i] = v
		l.feet[i] = v
	}

	// Reset the state, to set the timer.
	l.SetState(sDefault)

	return l
}

// Boot does nothing. Legs (and hence servos) are booted by the initializer now.
func (l *Legs) Boot() error {
	return nil
}

func (l *Legs) SetState(s State) {
	//log.Infof("state=%v", s)
	l.stateCounter = 0
	l.stateTime = time.Now()
	l.State = s
}

// SetMovingSpeed sets the moving speed of all leg servos. This is only useful
// during startup; most of the time, the speed should be constant.
func (l *Legs) SetMovingSpeed(speed int) error {
	for _, leg := range l.Legs {
		for _, servo := range leg.Servos() {
			err := servo.SetMovingSpeed(speed)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Clearance returns the distance (on the Y axis) which the body should be off
// the ground. This is mostly constant, but can be increased temporarily by
// pressing R2.
func (l *Legs) standingClearance() float64 {
	return defaultStandingClearance
}

// homeFootPosition returns a vector in the WORLD coordinate space for the home
// position of the given leg.
func (l *Legs) homeFootPosition(leg *Leg, pose math3d.Pose) math3d.Vector3 {
	r := utils.Rad(pose.Heading + leg.Angle)
	x := math.Cos(r) * stepRadius
	z := -math.Sin(r) * stepRadius
	return *pose.Position.Add(math3d.Vector3{X: x, Y: 0.0, Z: z})
}

func (l *Legs) Tick(now time.Time, state *hexapod.State) error {
	l.stateCounter += 1

	switch l.State {
	case sHalt:
		return nil

	case sDefault:
		if state.Shutdown {
			l.SetState(sSitDown)
			break
		}

		if l.stateCounter == 1 {

			// Set all servos very slow
			err := l.SetMovingSpeed(moveSpeedSlow)
			if err != nil {
				return err
			}

			// Set the target for each foot to its home position.
			for i, leg := range l.Legs {
				l.feet[i] = l.homeFootPosition(leg, state.Pose)
			}
		}

		// Count the total distance between the actual foot positions and the
		// target/home positions set above. We use this to wait indefinitely
		// until each foot has reached its destination.

		var td float64
		for i, leg := range l.Legs {
			pv, err := leg.PresentPosition()
			if err != nil {
				log.Error(err)
				continue
			}

			//log.Infof("%s end is at: %v (home=%v, distance=%+07.2f)", leg.Name, pv, l.feet[i], pv.Distance(hv))
			td += pv.Distance(l.feet[i])
		}

		// If the total distance is within the margin of error, reset move speed
		// (now that we know it won't be jerky, because the feet are already at
		// their destination), and proceed to stand up.

		if td < 5*6 {

			err := l.SetMovingSpeed(moveSpeedFast)
			if err != nil {
				return err
			}

			l.SetState(sStandUp)
			break
		}

		if l.stateCounter%10 == 0 {
			log.Infof("distance to home positions: %+07.2f", td)
		}

	// After initialzation, raise the clearance to lift the body off the
	// ground, into the standing position.
	case sStandUp:
		if state.Shutdown {
			l.SetState(sSitDown)
			break
		}

		if l.stateCounter == 1 {
			l.lastPose = state.Pose
			state.Target.Position.Y = l.standingClearance()
		}

		d := float64(time.Since(l.stateTime)) / float64(sitStandDuration)
		p := math.Min(float64(d), 1)
		//log.Infof("ts=%s, d=%s, p=%0.2f, ty=%0.2f, ly=%0.2f", ts, d, p, state.Target.Position.Y, l.lastPose.Position.Y)
		state.Pose.Position.Y = l.lastPose.Position.Y + (p * (state.Target.Position.Y - l.lastPose.Position.Y))

		if p >= 1 {
			l.SetState(sStanding)
		}

	// Lower the clearance until the body is sitting on the ground.
	case sSitDown:
		if l.stateCounter == 1 {
			l.lastPose = state.Pose
			state.Target.Position.Y = -2.0
		}

		d := float64(time.Since(l.stateTime)) / float64(sitStandDuration)
		p := math.Min(float64(d), 1)
		state.Pose.Position.Y = l.lastPose.Position.Y + (p * (state.Target.Position.Y - l.lastPose.Position.Y))

		if p >= 1 {
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
			l.lastPose = state.Pose
			for i, _ := range l.Legs {
				l.lastFeet[i] = l.feet[i]
			}

			vecToGoal := state.Target.Position.Subtract(state.Pose.Position)
			distToGoal := vecToGoal.Magnitude()

			// Cap the distance we wil (attempt to) step at the max.
			distToStep := math.Min(distToGoal, maxStepDistance)

			if distToStep > minStepDistance || math.Abs(state.Target.Heading-state.Pose.Heading) > 5.0 {

				// Calculate the target position for the origin.
				vecToStep := vecToGoal.Unit().MultiplyByScalar(distToStep)
				l.target.Position = *l.lastPose.Position.Add(vecToStep)
				l.target.Heading = state.Target.Heading
				log.Infof("stepping from %v to %v", l.lastPose, l.target)

			} else {
				l.target = l.lastPose
				//log.Infof("not stepping")
				if state.Shutdown {
					l.SetState(sSitDown)
				} else {
					l.SetState(sStepping)
				}
				break
			}

			// Calculate the target position for each foot. Might be where they
			// already are, if we're not stepping.
			for i, leg := range l.Legs {
				l.nextFeet[i] = l.homeFootPosition(leg, l.target)
			}
		}

		// Move continuously towards target. Note that we don't bother with the
		// rotation (for now), so the hex will walk sideways or backwards if the
		// target happens to be in that direction.
		r := float64(l.stateCounter) / float64(l.Gait.Length())
		v := l.target.Position.Subtract(l.lastPose.Position)
		rr := l.target.Heading - l.lastPose.Heading

		state.Pose.Position = *l.lastPose.Position.Add(v.MultiplyByScalar(r))
		state.Pose.Heading = l.lastPose.Heading + (r * rr)

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
			if state.Shutdown {
				l.SetState(sSitDown)
			} else {
				l.SetState(sStepping)
			}
		}

	default:
		return fmt.Errorf("unknown state: %#v", l.State)
	}

	// Update the position of each foot
	for i, leg := range l.Legs {
		pp := l.feet[i].MultiplyByMatrix44(state.Local())
		//log.Infof("%s world=%v, local=%v, dist=%0.2f", leg.Name, l.feet[i], pp, l.feet[i].Subtract(state.Position).Magnitude())
		leg.SetGoal(pp)
	}

	return nil
}
