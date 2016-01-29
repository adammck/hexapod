package hexapod

import (
	"fmt"
	"github.com/adammck/dynamixel/network"
	"github.com/adammck/hexapod/math3d"
	"github.com/adammck/sixaxis"
	"github.com/jacobsa/go-serial/serial"
	"math"
	"time"
)

type Hexapod struct {
	Network    *network.Network
	Components []Component

	// The world coordinates of the center of the hexapod.
	// TODO (adammck): Store the rotation as Euler angles, and modify the
	//                 heading when rotating with L/R buttons. This is more
	//                 self-documenting than storing the heading as a float.
	Position math3d.Vector3
	Rotation float64

	// Components can set this to true to indicate that the hex should shut down.
	// TODO: Is this the same as returning an error from Tick()?
	Shutdown bool
}

type Component interface {
	Boot() error
	Tick(time.Time) error
}

// NewHexapod creates a new Hexapod object on the given Dynamixel network.
func NewHexapod(network *network.Network) *Hexapod {
	return &Hexapod{
		Network:    network,
		Components: []Component{},
		Position:   math3d.Vector3{0, 0, 0},
		Rotation:   0.0,
	}
}

// Add registers a component to receive ticks every frame.
func (h *Hexapod) Add(c Component) {
	h.Components = append(h.Components, c)
}

// Boot calls Boot on each component.
func (h *Hexapod) Boot() error {
	for _, c := range h.Components {
		err := c.Boot()
		if err != nil {
			return err
		}
	}

	return nil
}

// Tick calls Tick on each component.
func (h *Hexapod) Tick(now time.Time) {
	for _, c := range h.Components {
		c.Tick(now)
	}
}

// NeedsVoltageCheck returns true if it's been a while since we checked the
// voltage level. The timeout is pretty arbitrary.
func (h *Hexapod) NeedsVoltageCheck() bool {
	return time.Since(h.lastVoltageCheck) > 2*time.Second
}

// CheckVoltage fetches the voltage level of an arbitrary servo, and returns an
// error if it's below 9.6v.
func (h *Hexapod) CheckVoltage() error {
	v, err := h.Legs[0].Coxa.Voltage()
	h.lastVoltageCheck = time.Now()
	if err != nil {
		return err
	}

	fmt.Printf("voltage: %fv\n", v)

	if v < 9.6 {
		return fmt.Errorf("low voltage: %fv", v)
	}

	return nil
}

// World returns a matrix to transform a vector in the hexapod coordinate space
// into the world space.
func (h *Hexapod) World() math3d.Matrix44 {
	return *math3d.MakeMatrix44(h.Position, *math3d.MakeSingularEulerAngle(math3d.RotationHeading, h.Rotation))
}

// Local returns a matrix to transform a vector in the world coordinate space
// into the hexapod's space, taking into account its current position and
// rotation.
func (h *Hexapod) Local() math3d.Matrix44 {
	return h.World().Inverse()
}
