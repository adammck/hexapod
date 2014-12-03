package hexapod

import (
	"fmt"
)

type Pair struct {
	one EulerAngles
	two EulerAngles
}

const (
	RotationHeading rotation = iota
	RotationPitch   rotation = iota
	RotationBank    rotation = iota
)

func MakePair(rot rotation, degOne float64, degTwo float64) *Pair {
	return &Pair{
		*MakeSingularEulerAngle(rot, degOne),
		*MakeSingularEulerAngle(rot, degTwo),
	}
}

func (p Pair) String() string {
	return fmt.Sprintf("&Pair{%s %s}", p.one, p.two)
}
