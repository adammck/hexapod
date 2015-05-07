package math3d

import (
	"fmt"
	"github.com/adammck/hexapod/utils"
)

type EulerAngles struct {
	Heading float64 // y
	Pitch   float64 // x
	Bank    float64 // z
}

type rotation int

const (
	RotationHeading rotation = iota
	RotationPitch   rotation = iota
	RotationBank    rotation = iota
)

var (
	IdentityOrientation = EulerAngles{}
)

func MakeSingularEulerAngle(rot rotation, angle float64) *EulerAngles {
	ea := &EulerAngles{}

	switch rot {
	case RotationHeading:
		ea.Heading = utils.Rad(angle)

	case RotationPitch:
		ea.Pitch = utils.Rad(angle)

	case RotationBank:
		ea.Bank = utils.Rad(angle)

	default:
		panic("invalid rotation")
	}

	return ea
}

// TODO: GTFO?
func MakeEulerAngles(h float64, p float64, b float64) *EulerAngles {
	return &EulerAngles{h, p, b}
}

func (ea EulerAngles) String() string {
	return fmt.Sprintf("&Euler{h=%+.2f° p=%+.2f° b=%+.2f°}", utils.Deg(ea.Heading), utils.Deg(ea.Pitch), utils.Deg(ea.Bank))
}
