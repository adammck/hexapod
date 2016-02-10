package math3d

import (
	"fmt"
)

type Pose struct {
	Position Vector3
	Heading  float64
}

func (p Pose) String() string {
	return fmt.Sprintf("Pose{x=%+07.2f y=%+07.2f z=%+07.2f, r=%+07.2f}", p.Position.X, p.Position.Y, p.Position.Z, p.Heading)
}

func (p Pose) Add(pp Pose) Pose {
	m := Matrix44{}
	m.SetRotation(p.ea())
	return Pose{
		Position: *p.Position.Add(pp.Position.MultiplyByMatrix44(m)),
		Heading:  p.Heading + pp.Heading,
	}
}

func (p Pose) ea() EulerAngles {
	return *MakeSingularEulerAngle(RotationHeading, p.Heading)
}
