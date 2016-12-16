package hexapod

import (
	"fmt"
	"github.com/adammck/dynamixel/iface"
	"github.com/adammck/dynamixel/network"
	proto1 "github.com/adammck/dynamixel/protocol/v1"
	"github.com/adammck/hexapod/math3d"
	"time"
)

type State struct {

	// The approximate number of frames per second which the main loop is
	// currently running at. This can vary quite a bit depending on the load.
	FPS int

	// Components can set this to true to indicate that the hex should shut
	// down. Components which need to clean up before being terminated (e.g.
	// powering off servos) should check this value frequently.
	Shutdown bool

	// The actual pose at the origin, in the world coordinate space. This should
	// be updated as accurately as possible as the hex walks around.
	Pose math3d.Pose

	// The target pose of the origin, in the world space. This can be set to
	// instruct the legs to walk towards an arbitrary point.
	Target math3d.Pose

	// The point to aim the head (camera) at, in the world space. This is a
	// pointer so it can be set to nil if there is no target.
	LookAt *math3d.Vector3
}

// World returns a matrix to transform a vector in the coordinate space defined
// by the Position and Rotation attributes into the world space.
func (s *State) World() math3d.Matrix44 {
	return *math3d.MakeMatrix44(s.Pose.Position, *math3d.MakeSingularEulerAngle(math3d.RotationHeading, s.Pose.Heading))
}

// Local returns a matrix to transform a vector in the world coordinate space
// into the space defined by the state (using the Position and Rotation attrs).
func (s *State) Local() math3d.Matrix44 {
	return s.World().Inverse()
}

type Hexapod struct {
	Network    *network.Network // TODO: Make this a io.ReadWriter
	Components []Component

	// Keep an unbound (i.e. having no particular servo ID) protocol for each
	// version present on the Dynamixel network. This is just v1, so long as
	// we're only using AX-12s.
	Protocols []iface.Protocol

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
		Protocols: []iface.Protocol{
			proto1.New(network),
		},
		State: &State{
			FPS: 0,
			Pose: math3d.Pose{
				Position: math3d.ZeroVector3,
				Heading:  0,
			},
			Target: math3d.Pose{
				Position: math3d.ZeroVector3,
				Heading:  0,
			},
			LookAt: nil,
		},
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

// Tick calls Tick on each component, then sends the ACTION instruction to
// trigger any buffered instructions.
func (h *Hexapod) Tick(now time.Time, state *State) error {
	for _, c := range h.Components {
		err := c.Tick(now, state)
		if err != nil {
			return fmt.Errorf("%T.Tick returned error: %v", c, err)
		}
	}

	for i := range h.Protocols {
		h.Protocols[i].Action()
	}

	return nil
}
