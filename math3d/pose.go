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

func (p Pose) ToWorld() Matrix44 {
	return *MakeMatrix44(p.Position, p.ea())
}

func (p Pose) Add(pp Pose) Pose {
	return Pose{
		Position: pp.Position.MultiplyByMatrix44(p.ToWorld()),
		Heading:  p.Heading + pp.Heading,
	}
}

func (p Pose) ToLocal() Matrix44 {
	return MakeMatrix44(p.Position, p.ea()).Inverse()
}

// Out returns the given pose (which is assumed to be in this pose's coordinate
// space) in the parent space.
func (p Pose) Out(pp Pose) Pose {
	return Pose{
		Position: pp.Position.MultiplyByMatrix44(p.ToLocal()),
		Heading:  pp.Heading - p.Heading,
	}
}

func (p Pose) ea() EulerAngles {
	return *MakeSingularEulerAngle(RotationHeading, p.Heading)
}
