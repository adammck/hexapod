package hexapod

import (
	"github.com/adammck/dynamixel"
)

type Leg struct {
	Origin *Point3d
	Angle  float64
	Coxa   *dynamixel.DynamixelServo
	Femur  *dynamixel.DynamixelServo
	Tibia  *dynamixel.DynamixelServo
	Tarsus *dynamixel.DynamixelServo
}

func NewLeg(network *dynamixel.DynamixelNetwork, baseId int, origin *Point3d, angle float64) *Leg {
	return &Leg{
		Origin: origin,
		Angle:  angle,
		Coxa:   dynamixel.NewServo(network, uint8(baseId+1)),
		Femur:  dynamixel.NewServo(network, uint8(baseId+2)),
		Tibia:  dynamixel.NewServo(network, uint8(baseId+3)),
		Tarsus: dynamixel.NewServo(network, uint8(baseId+4)),
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
