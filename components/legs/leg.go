package legs

import (
	"fmt"
	"math"

	"github.com/Sirupsen/logrus"
	"github.com/adammck/dynamixel/network"
	"github.com/adammck/dynamixel/servo"
	"github.com/adammck/hexapod/math3d"
	"github.com/adammck/hexapod/servos"
	"github.com/adammck/hexapod/utils"
)

const (

	// The offset between the start and end of the coxa segment, relative to the
	// zero vector of the start, which is relative to the origin of the leg. (So
	// away from the world origin is the Z axis.)
	coxaOffsetY = -12.0
	coxaOffsetZ = 39.0

	// The length of each segment is also measured on the Z axis (or "forwards"
	// from the origin), since each exists in its own coordinate space.
	femurLength  = 100.0
	tibiaLength  = 85.0
	tarsusLength = 80.5

	// How much extra angle (in degrees) to position the tarsus. This is a hack
	// to compensate for the amount of mechanical slack in the leg.
	tarsusExtraAngle = 5
)

type Leg struct {
	Name   string
	Origin *math3d.Vector3
	Coxa   *servo.Servo
	Femur  *servo.Servo
	Tibia  *servo.Servo
	Tarsus *servo.Servo

	// TODO: Rename this to 'Heading', since that's what it is.
	Angle float64
}

func NewLeg(network *network.Network, baseId int, name string, origin *math3d.Vector3, angle float64) *Leg {
	coxa := mustGetServo(network, baseId+1)
	femur := mustGetServo(network, baseId+2)
	tibia := mustGetServo(network, baseId+3)
	tarsus := mustGetServo(network, baseId+4)

	return &Leg{
		Origin: origin,
		Angle:  angle,
		Name:   name,
		Coxa:   coxa,
		Femur:  femur,
		Tibia:  tibia,
		Tarsus: tarsus,
	}
}

func mustGetServo(network *network.Network, ID int) *servo.Servo {
	s, err := servos.New(network, ID)
	if err != nil {
		panic(err)
	}

	return s
}

// Matrix returns a pointer to a 4x4 matrix, to transform a vector in the leg's
// coordinate space into the parent (hexapod) space.
func (leg *Leg) Matrix() math3d.Matrix44 {
	return *math3d.MakeMatrix44(*leg.Origin, *math3d.MakeSingularEulerAngle(math3d.RotationHeading, leg.Angle))
}

// Servos returns an array of all servos attached to this leg.
func (leg *Leg) Servos() []*servo.Servo {
	return []*servo.Servo{
		leg.Coxa,
		leg.Femur,
		leg.Tibia,
		leg.Tarsus,
	}
}

func (leg *Leg) SetLED(state bool) {
	for _, s := range leg.Servos() {
		s.SetLED(state)
	}
}

// rootSegment returns the segment at the origin of this leg.
func (leg *Leg) rootSegment() *Segment {

	// The position of the leg in world space must be specified by two segments.
	// The first positions it, then the second (which is always zero-length)
	// rotates it into the home orientation. This is the opposite of most
	// segments, which rotate from their start, rather than their end.
	s1 := MakeRootSegment(*leg.Origin)
	return MakeSegment("s2", s1, *math3d.MakeSingularEulerAngle(math3d.RotationHeading, leg.Angle), *math3d.MakeVector3(0, 0, 0))
}

// PresentPosition returns the actual present posion (relative to the center of
// the hexapod) of the end of this leg. This involves reading the position of
// each servo, so don't call it in the main loop.
func (leg *Leg) PresentPosition() (math3d.Vector3, error) {
	v := math3d.ZeroVector3

	coxPos, err := leg.Coxa.Angle()
	if err != nil {
		return v, fmt.Errorf("%s (while getting %s coxa (#%d) position)", err, leg.Name, leg.Coxa.ID)
	}

	femPos, err := leg.Femur.Angle()
	if err != nil {
		return v, fmt.Errorf("%s (while getting %s femur (#%d) position)", err, leg.Name, leg.Femur.ID)
	}

	tibPos, err := leg.Tibia.Angle()
	if err != nil {
		return v, fmt.Errorf("%s (while getting %s tibia (#%d) position)", err, leg.Name, leg.Tibia.ID)
	}

	tarPos, err := leg.Tarsus.Angle()
	if err != nil {
		return v, fmt.Errorf("%s (while getting %s tarsus (#%d) position)", err, leg.Name, leg.Tarsus.ID)
	}

	root := leg.rootSegment()
	coxa := MakeSegment("coxa", root, *math3d.MakeSingularEulerAngle(math3d.RotationHeading, coxPos), *math3d.MakeVector3(0, coxaOffsetY, coxaOffsetZ))
	femur := MakeSegment("femur", coxa, *math3d.MakeSingularEulerAngle(math3d.RotationPitch, femPos), *math3d.MakeVector3(0, 0, femurLength))
	tibia := MakeSegment("tibia", femur, *math3d.MakeSingularEulerAngle(math3d.RotationPitch, tibPos), *math3d.MakeVector3(0, 0, tibiaLength))
	tarsus := MakeSegment("tarsus", tibia, *math3d.MakeSingularEulerAngle(math3d.RotationPitch, tarPos), *math3d.MakeVector3(0, 0, tarsusLength))

	return tarsus.End(), nil
}

// SetGoal sets the goal position of the leg to the given vector in the chassis
// coordinate space.
func (leg *Leg) SetGoal(vt math3d.Vector3) {

	// Solve the angle of the coxa by looking at the position of the target from
	// above (x,z). Note that "above" here is in the chassis space, which might
	// not be parallel to the actual ground. Fortunately, the coxa moves around
	// the Y axis in that space, so we can cheat with 2d trig.

	coxPos := utils.Deg(math.Atan2(vt.X-leg.Origin.X, vt.Z-leg.Origin.Z)) - leg.Angle

	// The other joints are all on the same plane, which we know intersects vt
	// from the above. So the rest of the function can use 2d trig on the (z,y)
	// axis in the coxa space. More cheating!

	root := leg.rootSegment()
	coxa := MakeSegment("coxa", root, *math3d.MakeSingularEulerAngle(math3d.RotationHeading, coxPos), *math3d.MakeVector3(0, coxaOffsetY, coxaOffsetZ))

	// The following points (vr,vt) and lengths (a,b,c) are known:
	//
	//         (?)
	//         / \
	//        /   \
	//       a     b
	//      /       \
	//     /         \
	//   (vr)        (?)
	//                |
	//                c
	//                |
	//              (vt)
	//
	vr := coxa.End()
	a := femurLength
	b := tibiaLength
	c := tarsusLength

	// Pick a totally arbitrary point below (vr), to make more triangles.
	vp := *vr.Add(math3d.Vector3{X: 0, Y: -50, Z: 0})

	// The tarsus joint should always be directly above the target. We want that
	// last segment to be perpendicular to the ground, because it looks cool.
	vq := *vt.Add(math3d.Vector3{X: 0, Y: tarsusLength, Z: 0})

	// The leg now looks like:
	//
	//         (?)
	//         / \
	//        /   \
	//       a     b
	//      /       \
	//     /         \
	//   (vr)       (vq)
	//    |           |
	//   (vp)         c
	//                |
	//              (vt)
	//

	// Calculate the length of the remaining edges.
	d := vr.Distance(vq)
	e := vr.Distance(vt)
	f := vr.Distance(vp) // always vr.Y-50?
	g := vp.Distance(vt)

	// Calculate the inner angles of the triangles using the law of cos.
	aa := sss(b, a, d)
	bb := sss(c, d, e)
	cc := sss(g, e, f)
	dd := sss(a, d, b)
	ee := sss(e, c, d)
	hh := 180 - (aa + dd)

	// Transform inner angles to servo angles. The zero angle of each servo
	// makes the leg stick directly outwards from the chassis.
	femPos := 90 - (aa + bb + cc)
	tibPos := 180 - hh
	tarPos := 180 - (dd + ee)

	// Crash if any of the angles are invalid.

	err := false

	if math.IsNaN(coxPos) {
		logrus.Errorf("invalid %s coxa angle: %0.2f", leg.Name, coxPos)
		err = true
	}

	if math.IsNaN(femPos) {
		logrus.Errorf("invalid %s femur angle: %0.2f", leg.Name, femPos)
		err = true
	}

	if math.IsNaN(tibPos) {
		logrus.Errorf("invalid %s tibia angle: %0.2f", leg.Name, tibPos)
		err = true
	}

	if math.IsNaN(tarPos) {
		logrus.Errorf("invalid %s tarsus angle: %0.2f", leg.Name, tarPos)
		err = true
	}

	// Dump a bunch of debugging info and crash if anything went wrong. This is
	// of course way too hasty, but handy for now.
	if err {
		logrus.Errorf("a=%0.2f, b=%0.2f, c=%0.2f, d=%0.2f, e=%0.2f, f=%0.2f, g=%0.2f", a, b, c, d, e, f, g)
		logrus.Errorf("aa=%0.2f, bb=%0.2f, cc=%0.2f, dd=%0.2f, ee=%0.2f, hh=%0.2f", aa, bb, cc, dd, ee, hh)
		panic("goal out of range")
	}

	// Move the servos!
	leg.Coxa.MoveTo(coxPos)
	leg.Femur.MoveTo(femPos)
	leg.Tibia.MoveTo(tibPos)
	leg.Tarsus.MoveTo(tarPos)
}

// sss returns the angle α, given the length of sides a, b, and c.
// See: http://en.wikipedia.org/wiki/Solution_of_triangles
func sss(a float64, b float64, c float64) float64 {
	return utils.Deg(math.Acos(((b * b) + (c * c) - (a * a)) / (2 * b * c)))
}
