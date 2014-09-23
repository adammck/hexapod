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

// func (leg *Leg) UpdateSegments(coxa float64, femur float64, tibia float64, tarsus float64) {
// 	coxa.Angle.Heading = rad(coxa)
// 	leg.femur.Angle.Bank = rad(femur)
// 	leg.tibia.Angle.Bank = rad(tibia)
// 	leg.tarsus.Angle.Bank = rad(tarsus)
// }

// func (leg *Leg) End() ik.Vector3 {
// 	return leg.tarsus.End()
// }

//
// Sets the goal position of this leg to the given x/y/z coordinates, relative
// to the center of the hexapod.
//
func (leg *Leg) SetGoal(x float64, y float64, z float64) {

	// The position of the object in space must be specified by two segments. The
	// first positions it, then the second (which is always zero-length) rotates
	// it into the home orientation.
	r1 := ik.MakeRootSegment(*ik.MakeVector3(leg.Origin.X, leg.Origin.Y, leg.Origin.Z))
	r2 := ik.MakeSegment("r2", r1, *ik.MakePair(ik.RotationHeading, leg.Angle, leg.Angle), *ik.MakeVector3(0, 0, 0))

	// Movable segments (angles in deg, vectors in mm)
	coxa   := ik.MakeSegment("coxa",   r2,    *ik.MakePair(ik.RotationHeading, 40,  -40), *ik.MakeVector3(39,  9, -21))
	femur  := ik.MakeSegment("femur",  coxa,  *ik.MakePair(ik.RotationBank,    90,    0), *ik.MakeVector3(100, 0,   0))
	tibia  := ik.MakeSegment("tibia",  femur, *ik.MakePair(ik.RotationBank,     0, -135), *ik.MakeVector3(85,  0,  21))
	tarsus := ik.MakeSegment("tarsus", tibia, *ik.MakePair(ik.RotationBank,    90,  -90), *ik.MakeVector3(64,  0,   0))
	_ = tarsus

	v := &ik.Vector3{x, y, z}

	//fmt.Println("Solving")
	//fmt.Printf("segment: %v\n", coxa)
	fmt.Printf("root:   %v\n", coxa.Start())
	fmt.Printf("target: %v\n", v)

	vv := v.Add(ik.Vector3{0, 64, 0})
	best := ik.Solve(coxa, tibia, vv, 1)
	bestCoxa   := best.Segment
	bestFemur  := bestCoxa.Child
	bestTibia  := bestFemur.Child
	//bestTarsus := bestTibia.Child
	//_ = bestTarsus

	fmt.Printf("THE BEST: %v\n", best)

	if best.Distance > 10 {
		fmt.Printf("No thanks!\n")
		//return
	}

	//coxaAngle   := deg(theta)
	coxaAngle   := deg(0 - bestCoxa.Angle.Heading)
	femurAngle  := deg(0 - bestFemur.Angle.Bank)
	tibiaAngle  := deg(0 - bestTibia.Angle.Bank)
	tarsusAngle := 90 - deg(0 - bestFemur.Angle.Bank) - tibiaAngle

	// fmt.Printf("coxaAngle: %v\n", coxaAngle)
	// fmt.Printf("femurAngle: %v\n", femurAngle)
	// fmt.Printf("tibiaAngle: %v\n", tibiaAngle)
	// fmt.Printf("tarsusAngle: %v\n", tarsusAngle)

	leg.Coxa.MoveTo(coxaAngle)
	leg.Femur.MoveTo(femurAngle)
	leg.Tibia.MoveTo(tibiaAngle)
	leg.Tarsus.MoveTo(tarsusAngle)

	_ = coxaAngle
	_ = femurAngle
	_ = tibiaAngle
	_ = tarsusAngle
}

func deg(rads float64) float64 {
	return rads / (math.Pi / 180)
}


func rad(degrees float64) float64 {
	return (math.Pi / 180) * degrees
}
