package legs

import (
	"fmt"
	"math"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/adammck/dynamixel/network"
	"github.com/adammck/dynamixel/servo"
	"github.com/adammck/hexapod"
	"github.com/adammck/hexapod/components/legs/gait"
	"github.com/adammck/hexapod/math3d"
)

type State string

const (
	sDefault  State = ""
	sStandUp  State = "sStandUp"
	sSitDown  State = "sSitDown"
	sStepping State = "sStepping"

	moveSpeedSlow   = 256
	torqueLimitSlow = 128

	moveSpeedFast   = 1023
	torqueLimitFast = 1023

	// The distance (in mm) to adjust the Y position to meet the Y target each
	// tick. This mostly controls the time it takes to stand up and sit down.
	yMoveSpeed = 1

	bankMoveSpeed  = 1
	pitchMoveSpeed = 1

	// Distance (on the X/Z axis) from the origin to the point at which the feet
	// should be positioned. This isn't adjustable at runtime, because there are
	// very few valid settings.
	stepRadius = 240.0

	// The number of ticks per step, i.e. a single foot is lifted, moved to its
	// new position, and put down.
	ticksPerStep = 20

	// The offset (on the Y axis) which feet should be moved to on the up step,
	// relative to the origin.
	stepHeight = 40.0

	// Minimum distance which the desired foot position should be from its actual
	// position before a step should be taken to correct it.
	minStepDistance = 20.0

	// The distance (in mm) which the hex can move per step cycle. This should
	// be determined experimentally; too high and the legs get tangled up.
	maxStepDistance = 70.0
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

	// Defaults to false, and set to true by the goroutine started by Boot once
	// the feet have reached the home position and are ready to start the main
	// tick loop.
	ready bool

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

	// World positions of the NEXT foot position. These are nil if we're okay
	// with where the foot is now, but are set if the foot should be relocated.
	nextFeet [6]math3d.Vector3
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
			//
			// Note that the angles are the direction in which the leg is
			// pointing, NOT the angle between the hex and leg origins.
			//
			NewLeg(n, 40, "FL", math3d.MakeVector3(-61.167, 24, 98), 300),  // Front Left  - 0
			NewLeg(n, 50, "FR", math3d.MakeVector3(61.167, 24, 98), 60),    // Front Right - 1
			NewLeg(n, 60, "MR", math3d.MakeVector3(81, 24, 0), 90),         // Mid Right   - 2
			NewLeg(n, 10, "BR", math3d.MakeVector3(61.167, 24, -98), 120),  // Back Right  - 3
			NewLeg(n, 20, "BL", math3d.MakeVector3(-61.167, 24, -98), 240), // Back Left   - 4
			NewLeg(n, 30, "ML", math3d.MakeVector3(-81, 24, 0), 270),       // Mid Left    - 5
		},
	}

	// Initialize each foot to its home position. This will be written to the
	// servos during boot.
	l.feet = [6]math3d.Vector3{
		l.homeFootPosition(l.Legs[0], math3d.Pose{}),
		l.homeFootPosition(l.Legs[1], math3d.Pose{}),
		l.homeFootPosition(l.Legs[2], math3d.Pose{}),
		l.homeFootPosition(l.Legs[3], math3d.Pose{}),
		l.homeFootPosition(l.Legs[4], math3d.Pose{}),
		l.homeFootPosition(l.Legs[5], math3d.Pose{}),
	}

	// Reset the state, to set the timer.
	l.SetState(sDefault)

	return l
}

// TODO: Maybe provide State to boot, in case we have an initial pose? We're
//       using the zero value now, which seems like a shaky assumption.
func (l *Legs) Boot() error {

	// Set all servos very slow. Note that buffered mode is not enabled yet, so
	// this is applied immediately.

	for _, s := range l.Servos() {

		err := s.SetMovingSpeed(moveSpeedSlow)
		if err != nil {
			return fmt.Errorf("%s (while setting move speed)", err)
		}

		err = s.SetTorqueLimit(torqueLimitSlow)
		if err != nil {
			return fmt.Errorf("%s (while setting torque limit)", err)
		}
	}

	// Set the target for each foot to its home position.
	for i, leg := range l.Legs {
		leg.SetGoal(l.feet[i])
	}

	// TODO: Move this to a goroutine, and set ready when done.
	for {

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

			// Note that pv is in the hex local space, but l.feet is in the
			// world space. That's okay, because we're booting up, so haven't
			// moved yet. I should probably fix this anyway...

			//log.Infof("%s end is at: %v (home=%v, distance=%+07.2f)", leg.Name, pv, l.feet[i], pv.Distance(l.feet[i]))
			td += pv.Distance(l.feet[i])
		}

		// If the total distance is within the margin of error, reset move speed
		// (now that we know it won't be jerky, because the feet are already at
		// their destination), and proceed to stand up.

		if td < 3*6 {
			break
		}

		log.Infof("distance to home positions: %+07.2f", td)
		time.Sleep(100 * time.Millisecond)
	}

	for _, s := range l.Servos() {
		err := s.SetMovingSpeed(moveSpeedFast)
		if err != nil {
			return fmt.Errorf("%s (while setting move speed)", err)
		}

		err = s.SetTorqueLimit(torqueLimitFast)
		if err != nil {
			return fmt.Errorf("%s (while setting torque limit)", err)
		}

		// Enable buffered mode, so writes are batched at the end of each tick.
		// Otherwise the each leg would be slightly ahead of the previous.
		s.SetBuffered(true)
	}

	l.SetState(sStandUp)
	l.ready = true

	return nil
}

func (l *Legs) Servos() []*servo.Servo {
	s := make([]*servo.Servo, 0, 4*6)

	for _, leg := range l.Legs {
		for _, servo := range leg.Servos() {
			s = append(s, servo)
		}
	}

	return s
}

func (l *Legs) SetState(s State) {
	//log.Infof("state=%v", s)
	l.stateCounter = 0
	l.stateTime = time.Now()
	l.State = s
}

// homeFootPosition returns a vector in the WORLD coordinate space for the home
// position of the given leg.
func (l *Legs) homeFootPosition(leg *Leg, pose math3d.Pose) math3d.Vector3 {
	hyp := math.Sqrt((leg.Origin.X * leg.Origin.X) + (leg.Origin.Z * leg.Origin.Z))
	v := pose.Add(math3d.Pose{*leg.Origin, leg.Angle, 0, 0}).Add(math3d.Pose{math3d.Vector3{0, 0, stepRadius - hyp}, 0, 0, 0}).Position
	v.Y = 0.0
	return v
}

func (l *Legs) Tick(now time.Time, state *hexapod.State) error {
	l.stateCounter += 1

	if !l.ready {
		return nil
	}

	// TODO: Remove the state machine altogether? The first two are just waiting
	//       for the pose to converge with target, which the third also does.
	switch l.State {

	// After init, wait until the Y position has met the target Y position
	// before proceeding.
	case sStandUp:
		if state.Shutdown {
			l.SetState(sSitDown)
			break
		}

		yOffset := (state.Target.Position.Y - state.Pose.Position.Y)
		if math.Abs(yOffset) < 1 {
			l.SetState(sStepping)
		}

	// While in the sitdown state, force the target Y position to zero and wait
	// for the position to meet it before halting. Don't check state.Shutdown,
	// because we're already on the way.
	case sSitDown:
		state.Target.Position.Y = 0
		state.Target.Bank = 0
		state.Target.Pitch = 0

		yOffset := (state.Target.Position.Y - state.Pose.Position.Y)
		if math.Abs(yOffset) < 1 {
			l.ready = false
		}

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

			// Ignore Y axis for target and pose; we take care of that below.
			// TODO: Fix this ugly mess.
			xzPosePos := state.Pose.Position
			xzPosePos.Y = 0

			xzTargetPos := state.Target.Position
			xzTargetPos.Y = 0

			vecToGoal := xzTargetPos.Subtract(xzPosePos)
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

		// Ignore Y axis; we set that below, without tweening.
		y := state.Pose.Position.Y
		state.Pose.Position = *l.lastPose.Position.Add(v.MultiplyByScalar(r))
		state.Pose.Position.Y = y

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

	// Adjust the clearance if that's gotten off. This is how we stand up, sit
	// down, and adjust the clearance at runtime.
	yOffset := math.Max(-yMoveSpeed, math.Min(yMoveSpeed, (state.Target.Position.Y-state.Pose.Position.Y)))
	if yOffset != 0 {
		state.Pose.Position.Y += yOffset
	}

	bankOffset := math.Max(-bankMoveSpeed, math.Min(bankMoveSpeed, (state.Target.Bank-state.Pose.Bank)))
	if bankOffset != 0 {
		state.Pose.Bank += bankOffset
	}

	pitchOffset := math.Max(-pitchMoveSpeed, math.Min(pitchMoveSpeed, (state.Target.Pitch-state.Pose.Pitch)))
	if pitchOffset != 0 {
		state.Pose.Pitch += pitchOffset
	}

	// Update the goal of each leg.
	for i, leg := range l.Legs {
		pp := l.feet[i].MultiplyByMatrix44(state.Local())
		leg.SetGoal(pp)
	}

	return nil
}
