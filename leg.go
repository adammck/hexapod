package hexapod

import (
	"github.com/adammck/dynamixel"
	"github.com/adammck/ik"
	"fmt"
	"math"
)

type Leg struct {
	Origin  *Point3d
	Angle   float64
	Coxa    *dynamixel.DynamixelServo
	Femur   *dynamixel.DynamixelServo
	Tibia   *dynamixel.DynamixelServo
	Tarsus  *dynamixel.DynamixelServo
}

func NewLeg(network *dynamixel.DynamixelNetwork, baseId int, origin *Point3d, angle float64) *Leg {
	return &Leg{
		Origin:  origin,
		Angle:   angle,
		Coxa:    dynamixel.NewServo(network, uint8(baseId+1)),
		Femur:   dynamixel.NewServo(network, uint8(baseId+2)),
		Tibia:   dynamixel.NewServo(network, uint8(baseId+3)),
		Tarsus:  dynamixel.NewServo(network, uint8(baseId+4)),
	}
}

// Servos returns an array of all servos attached to this leg.
func (leg *Leg) Servos() [4]*dynamixel.DynamixelServo {
	return [4]*dynamixel.DynamixelServo{
		leg.Coxa,
		leg.Femur,
		leg.Tibia,
		leg.Tarsus,
	}
}

// calculateCoxaAngle calculates the angle (in degrees) which the coxa should be
// set to, for the target vector to be reachable.
func (leg *Leg) calculateCoxaAngle(v ik.Vector3) float64 {
	x := v.X - leg.Origin.X
	z := v.Z - leg.Origin.Z
	theta := deg(math.Atan2(-z, x))
	return 0 - (leg.Angle - theta)
}

// http://en.wikipedia.org/wiki/Solution_of_triangles#Three_sides_given_.28SSS.29
func _sss(a float64, b float64, c float64) float64 {
	return deg(math.Acos(((b*b) + (c*c) - (a*a)) / (2 * b * c)))
}

// Sets the goal position of this leg to the given x/y/z coordinates, relative
// to the center of the hexapod.
func (leg *Leg) SetGoal(x float64, y float64, z float64) {

	// The position of the object in space must be specified by two segments. The
	// first positions it, then the second (which is always zero-length) rotates
	// it into the home orientation.
	r1 := ik.MakeRootSegment(*ik.MakeVector3(leg.Origin.X, leg.Origin.Y, leg.Origin.Z))
	r2 := ik.MakeSegment("r2", r1, *ik.MakePair(ik.RotationHeading, leg.Angle, leg.Angle), *ik.MakeVector3(0, 0, 0))

	// Movable segments (angles in deg, vectors in mm)
	coxa   := ik.MakeSegment("coxa",   r2,    *ik.MakePair(ik.RotationHeading, 40,  -40), *ik.MakeVector3(39, -12,  0))
	femur  := ik.MakeSegment("femur",  coxa,  *ik.MakePair(ik.RotationBank,    90,    0), *ik.MakeVector3(100, 0,  0))
	tibia  := ik.MakeSegment("tibia",  femur, *ik.MakePair(ik.RotationBank,     0, -135), *ik.MakeVector3(85,  0,  0))
	tarsus := ik.MakeSegment("tarsus", tibia, *ik.MakePair(ik.RotationBank,    90,  -90), *ik.MakeVector3(64,  0,  0))
	_ = tarsus

	v := &ik.Vector3{x, y, z}
	vv := v.Add(ik.Vector3{0, 64, 0})

	// Solve the angle of the coxa by looking at the position of the target from
	// above (x,z). It's the only joint which rotates around the Y axis, so we can
	// cheat.

	adj := v.X - leg.Origin.X
	opp := v.Z - leg.Origin.Z
	theta := deg(math.Atan2(-opp, adj))
	coxaAngle := (leg.Angle - theta)
	coxa.Angle = *ik.MakeSingularEulerAngle(ik.RotationHeading, 0 - coxaAngle)
	//fmt.Printf("coxaAngle=%0.4f (s/o=%0.4f) (s/v=%0.4f) (e/o=%0.4f) (e/v=%0.4f)\n", coxaAngle, coxa.Start().Distance(ik.ZeroVector3), coxa.Start().Distance(*v), coxa.End().Distance(ik.ZeroVector3), coxa.End().Distance(*v))

	// Solve the other joints with a bunch of trig. Since we've already set the Y
	// rotation and the other joints only rotate around X (relative to the coxa,
	// anyway), we can solve them with a shitload of triangles.

	r := femur.Start()
	t := r
	t.Y = v.Y

	a := 100.0 // femur length
	b := 85.0  // tibia length
	c := 64.0  // tarsus length
	d := r.Distance(*vv)
	e := r.Distance(*v)
	f := r.Distance(t)
	g := t.Distance(*v)
	//fmt.Printf("a=%0.4f, b=%0.4f, c=%0.4f, d=%0.4f, e=%0.4f, f=%0.4f, g=%0.4f\n", a, b, c, d, e, f, g)

	aa := _sss(b, a, d)
	bb := _sss(c, d, e)
	cc := _sss(g, e, f)
	dd := _sss(a, d, b)
	ee := _sss(e, c, d)
	hh := 180 - aa - dd
	//fmt.Printf("aa=%0.4f, bb=%0.4f, cc=%0.4f, dd=%0.4f, ee=%0.4f\n", aa, bb, cc, dd, ee)

	femurAngle := (aa + bb + cc) - 90
	femur.Angle = *ik.MakeSingularEulerAngle(ik.RotationBank, 0 - femurAngle)
	//fmt.Printf("femurAngle=%0.4f\n", femurAngle)

	tibiaAngle := 180 - hh
	tibia.Angle = *ik.MakeSingularEulerAngle(ik.RotationBank, 0 - tibiaAngle)
	//fmt.Printf("tibiaAngle=%0.4f\n", tibiaAngle)

	tarsusAngle := 180 - (dd + ee)
	tarsus.Angle = *ik.MakeSingularEulerAngle(ik.RotationBank, 0 - tarsusAngle)
	//fmt.Printf("tarsusAngle=%0.4f\n", tarsusAngle)

	if math.IsNaN(coxaAngle) || math.IsNaN(femurAngle) || math.IsNaN(tibiaAngle) || math.IsNaN(tarsusAngle) {
		fmt.Println("ERROR")
		return
	}

	leg.Coxa.MoveTo(coxaAngle)
	leg.Femur.MoveTo(0 - femurAngle)
	leg.Tibia.MoveTo(tibiaAngle)
	leg.Tarsus.MoveTo(tarsusAngle)
}

func deg(rads float64) float64 {
	return rads / (math.Pi / 180)
}


func rad(degrees float64) float64 {
	return (math.Pi / 180) * degrees
}
