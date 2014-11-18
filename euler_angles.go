package hexapod

import (
	"fmt"
	//"math"
)

type EulerAngles struct {
	Heading float64 // y
	Pitch   float64 // x
	Bank    float64 // z
}

type rotation int

var (
	IdentityOrientation = EulerAngles{}
)

func Euler(h float64, p float64, b float64) EulerAngles {
	return EulerAngles{rad(h), rad(p), rad(b)}
}

func MakeSingularEulerAngle(rot rotation, angle float64) *EulerAngles {
  ea := &EulerAngles{}

  switch rot {
  case RotationHeading:
    ea.Heading = rad(angle)

  case RotationPitch:
    ea.Pitch = rad(angle)

  case RotationBank:
    ea.Bank = rad(angle)

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
	return fmt.Sprintf("&Euler{h=%+.2f° p=%+.2f° b=%+.2f°}", deg(ea.Heading), deg(ea.Pitch), deg(ea.Bank))
}
