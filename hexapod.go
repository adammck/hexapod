package hexapod

import (
	"fmt"
	"github.com/adammck/dynamixel/network"
	"github.com/adammck/hexapod/math3d"
	"time"
)

type State struct {

	// Components can set this to true to indicate that the hex should shut
	// down. Components which need to clean up before being terminated (e.g.
	// powering off servos) should check this value frequently.
	Shutdown bool

	// The world coordinates of the actual center of the hexapod. This should be
	// updated as the hexapod moves around.
	//
	// TODO (adammck): Store the rotation as Euler angles, and modify the
	//                 heading when rotating with L/R buttons. This is more
	//                 self-documenting than storing the heading as a float.
	//
	Position math3d.Vector3
	Rotation float64

	// The world coordinates of the desired center of the hexapod. This can be
	// set to instruct the legs to walk towards a point.
	TargetPosition math3d.Vector3
	TargetRotation float64
}

type Hexapod struct {
	Network    *network.Network
	Components []Component

	// Most components receive (and update) the state every tick, to instruct or
	// react to instructions. This is more easily testable than passing around
	// references to the Hexapod itself.
	State *State
}

type Component interface {
	Boot() error
	Tick(time.Time, *State) error
}

// NewHexapod creates a new Hexapod object on the given Dynamixel network.
func NewHexapod(network *network.Network) *Hexapod {
	return &Hexapod{
		Network:    network,
		Components: []Component{},
		State:      &State{},
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
func (h *Hexapod) Tick(now time.Time, state *State) error {
	for _, c := range h.Components {
		err := c.Tick(now, state)
		if err != nil {
			return fmt.Errorf("%T.Tick returned error: %v", c, err)
		}
	}

	return nil
}

// World returns a matrix to transform a vector in the hexapod coordinate space
// into the world space.
func (h *Hexapod) World() math3d.Matrix44 {
	return *math3d.MakeMatrix44(h.State.Position, *math3d.MakeSingularEulerAngle(math3d.RotationHeading, h.State.Rotation))
}

// Local returns a matrix to transform a vector in the world coordinate space
// into the hexapod's space, taking into account its current position and
// rotation.
func (h *Hexapod) Local() math3d.Matrix44 {
	return h.World().Inverse()
}
