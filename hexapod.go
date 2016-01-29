package hexapod

import (
	"github.com/adammck/dynamixel/network"
	"github.com/adammck/hexapod/math3d"
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
		Position:   math3d.Vector3{X: 0, Y: 0, Z: 0},
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
