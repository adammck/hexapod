package math3d

import (
	"fmt"
	"github.com/adammck/hexapod/utils"
)

type Pose struct {
	Position Vector3

	// TODO: Remove these, replace with EulerAngles
	Heading float64 // y, yaw
	Pitch   float64 // x, pitch
	Bank    float64 // z, roll
}

func (p Pose) String() string {
	return fmt.Sprintf("Pose{x=%+07.2f y=%+07.2f z=%+07.2f h=%+07.2f p=%+07.2f b=%+07.2f}", p.Position.X, p.Position.Y, p.Position.Z, p.Heading, p.Pitch, p.Bank)
}

func (p Pose) Add(pp Pose) Pose {
	return Pose{
		Position: pp.Position.MultiplyByMatrix44(p.ToWorld()),
		Heading:  p.Heading + pp.Heading,
		Pitch:    p.Pitch + pp.Pitch,
		Bank:     p.Bank + pp.Bank,
	}
}

func (p Pose) ToWorld() Matrix44 {
	return *MakeMatrix44(p.Position, p.ea())
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
		Pitch:    p.Pitch - pp.Pitch,
		Bank:     p.Bank - pp.Bank,
	}
}

func (p Pose) ea() EulerAngles {

	// TODO: Why are these stored as degs not rads?
	return EulerAngles{
		Heading: utils.Rad(p.Heading),
		Pitch:   utils.Rad(p.Pitch),
		Bank:    utils.Rad(p.Bank),
	}
}
