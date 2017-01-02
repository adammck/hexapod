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

// http://en.wikipedia.org/wiki/Solution_of_triangles#Three_sides_given_.28SSS.29
func _sss(a float64, b float64, c float64) float64 {
	return utils.Deg(math.Acos(((b * b) + (c * c) - (a * a)) / (2 * b * c)))
}

// rootSegment returns the segment at the origin of this leg.
func (leg *Leg) rootSegment() *Segment {

	// The position of the leg in world space must be specified by two segments.
	// The first positions it, then the second (which is always zero-length)
	// rotates it into the home orientation.
	s1 := MakeRootSegment(*math3d.MakeVector3(leg.Origin.X, leg.Origin.Y, leg.Origin.Z))
	return MakeSegment("s2", s1, *math3d.MakeSingularEulerAngle(math3d.RotationHeading, leg.Angle), *math3d.MakeVector3(0, 0, 0))
}

// PresentPosition returns the actual present posion (relative to the center of
// the hexapod) of the end of this leg. This involves reading the position of
// each servo, so don't call it in the main loop.
func (leg *Leg) PresentPosition() (math3d.Vector3, error) {
	v := math3d.ZeroVector3

	coxaPos, err := leg.Coxa.Angle()
	if err != nil {
		return v, fmt.Errorf("%s (while getting %s coxa (#%d) position)", err, leg.Name, leg.Coxa.ID)
	}

	femurPos, err := leg.Femur.Angle()
	if err != nil {
		return v, fmt.Errorf("%s (while getting %s femur (#%d) position)", err, leg.Name, leg.Femur.ID)
	}

	tibiaPos, err := leg.Tibia.Angle()
	if err != nil {
		return v, fmt.Errorf("%s (while getting %s tibia (#%d) position)", err, leg.Name, leg.Tibia.ID)
	}

	tarsusPos, err := leg.Tarsus.Angle()
	if err != nil {
		return v, fmt.Errorf("%s (while getting %s tarsus (#%d) position)", err, leg.Name, leg.Tarsus.ID)
	}

	root := leg.rootSegment()
	coxa := MakeSegment("coxa", root, *math3d.MakeSingularEulerAngle(math3d.RotationHeading, coxaPos), *math3d.MakeVector3(0, coxaOffsetY, coxaOffsetZ))
	femur := MakeSegment("femur", coxa, *math3d.MakeSingularEulerAngle(math3d.RotationPitch, femurPos), *math3d.MakeVector3(0, 0, femurLength))
	tibia := MakeSegment("tibia", femur, *math3d.MakeSingularEulerAngle(math3d.RotationPitch, tibiaPos), *math3d.MakeVector3(0, 0, tibiaLength))
	tarsus := MakeSegment("tarsus", tibia, *math3d.MakeSingularEulerAngle(math3d.RotationPitch, tarsusPos), *math3d.MakeVector3(0, 0, tarsusLength))

	return tarsus.End(), nil
}

// Sets the goal position of this leg to the given x/y/z coords in the hexapod
// space. (Note that this also includes the orientation on all three axis.)
func (leg *Leg) SetGoal(v math3d.Vector3) {

	// Solve the angle of the coxa by looking at the position of the target from
	// above (x,z). It's the only joint which rotates around the Y axis, so we
	// can cheat.

	opp := v.X - leg.Origin.X
	adj := v.Z - leg.Origin.Z
	theta := utils.Deg(math.Atan2(opp, adj))
	coxaAngle := theta - leg.Angle

	// Solve the other joints with a bunch of trig. Since we've already set the Y
	// rotation and the other joints only rotate around X (relative to the coxa,
	// anyway), we can solve them with a shitload of triangles.

	root := leg.rootSegment()
	coxa := MakeSegment("coxa", root, *math3d.MakeSingularEulerAngle(math3d.RotationHeading, coxaAngle), *math3d.MakeVector3(0, coxaOffsetY, coxaOffsetZ))
	r := coxa.End()

	// totally arbitrary point below femur origin, just to figure out angles.
	t := *r.Add(math3d.Vector3{X: 0, Y: -50, Z: 0})
	//t.Y = -50 // TODO: can this be r.Y-50? absolute seems like a bad idea

	// tarsus is known to be directly above foot.
	vv := v.Add(math3d.Vector3{X: 0, Y: tarsusLength, Z: 0})

	a := femurLength
	b := tibiaLength
	c := tarsusLength
	d := r.Distance(*vv)
	e := r.Distance(v)
	f := r.Distance(t) // always be r.Y-50?
	g := t.Distance(v)

	aa := _sss(b, a, d) //
	bb := _sss(c, d, e) // angle between f and a (femur)
	cc := _sss(g, e, f) //

	dd := _sss(a, d, b) // angle between b and c (tarsus)
	ee := _sss(e, c, d) //

	hh := 180 - (aa + dd) // between a, b (tibia)

	femurAngle := 90 - (aa + bb + cc)
	tibiaAngle := 180 - hh
	tarsusAngle := 180 - (dd + ee)

	//logrus.Errorf("%s femur angle: %0.2f", leg.Name, femurAngle)
	//logrus.Errorf("%s tibia angle: %0.2f", leg.Name, tibiaAngle)
	//logrus.Errorf("%s tarsus angle: %0.2f", leg.Name, tarsusAngle)
	//return

	// fmt.Printf("v=%v, vv=%v, r=%v, t=%v\n", v, vv, r, t)
	// fmt.Printf("a=%0.4f, b=%0.4f, c=%0.4f, d=%0.4f, e=%0.4f, f=%0.4f, g=%0.4f\n", a, b, c, d, e, f, g)
	// fmt.Printf("aa=%0.4f, bb=%0.4f, cc=%0.4f, dd=%0.4f, ee=%0.4f\n", aa, bb, cc, dd, ee)
	// fmt.Printf("coxaAngle=%0.4f (s/o=%0.4f) (s/v=%0.4f) (e/o=%0.4f) (e/v=%0.4f)\n", coxaAngle, coxa.Start().Distance(ik.ZeroVector3), coxa.Start().Distance(*v), coxa.End().Distance(ik.ZeroVector3), coxa.End().Distance(*v))
	// fmt.Printf("femurAngle=%0.4f (s/o=%0.4f) (s/v=%0.4f) (e/o=%0.4f) (e/v=%0.4f)\n", femurAngle, femur.Start().Distance(ik.ZeroVector3), femur.Start().Distance(*v), femur.End().Distance(ik.ZeroVector3), femur.End().Distance(*v))
	// fmt.Printf("tibiaAngle=%0.4f (s/o=%0.4f) (s/v=%0.4f) (e/o=%0.4f) (e/v=%0.4f)\n", tibiaAngle, tibia.Start().Distance(ik.ZeroVector3), tibia.Start().Distance(*v), tibia.End().Distance(ik.ZeroVector3), tibia.End().Distance(*v))
	// fmt.Printf("tarsusAngle=%0.4f (s/o=%0.4f) (s/v=%0.4f) (e/o=%0.4f) (e/v=%0.4f)\n", tarsusAngle, tarsus.Start().Distance(ik.ZeroVector3), tarsus.Start().Distance(*v), tarsus.End().Distance(ik.ZeroVector3), tarsus.End().Distance(*v))

	//logrus.Infof("%s coxa=%0.2f, femur=%0.2f, tibia=%0.2f, tarsus=%0.2f", leg.Name, coxaAngle, femurAngle, tibiaAngle, tarsusAngle)

	err := false

	if math.IsNaN(coxaAngle) {
		logrus.Errorf("invalid %s coxa angle: %0.2f", leg.Name, coxaAngle)
		err = true
	}

	if math.IsNaN(femurAngle) {
		logrus.Errorf("invalid %s femur angle: %0.2f", leg.Name, femurAngle)
		err = true
	}

	if math.IsNaN(tibiaAngle) {
		logrus.Errorf("invalid %s tibia angle: %0.2f", leg.Name, tibiaAngle)
		err = true
	}

	if math.IsNaN(tarsusAngle) {
		logrus.Errorf("invalid %s tarsus angle: %0.2f", leg.Name, tarsusAngle)
		err = true
	}

	// Dump a bunch of debugging info and crash if anything went wrong. This is
	// of course way too hasty, but handy for now.
	if err {
		logrus.Errorf("a=%0.2f, b=%0.2f, c=%0.2f, d=%0.2f, e=%0.2f, f=%0.2f, g=%0.2f", a, b, c, d, e, f, g)
		logrus.Errorf("aa=%0.2f, bb=%0.2f, cc=%0.2f, dd=%0.2f, ee=%0.2f, hh=%0.2f", aa, bb, cc, dd, ee, hh)
		panic("goal out of range")
	}

	leg.Coxa.MoveTo(coxaAngle)
	leg.Femur.MoveTo(femurAngle)
	leg.Tibia.MoveTo(tibiaAngle)
	leg.Tarsus.MoveTo(tarsusAngle)
}
