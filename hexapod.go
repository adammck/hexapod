package hexapod

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/adammck/dynamixel/iface"
	"github.com/adammck/dynamixel/network"
	proto1 "github.com/adammck/dynamixel/protocol/v1"
	"github.com/adammck/hexapod/math3d"
	"github.com/adammck/hexapod/utils"
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

	// The offset from the actual home position which the feet should be
	// positioned at.
	Offset math3d.Vector3

	// The target pose of the origin, in the world space. This can be set to
	// instruct the legs to walk towards an arbitrary point, and the chassis to
	// orient itself strangely.
	Target math3d.Pose

	// The point to aim the head (camera) at, in the world space. This is a
	// pointer so it can be set to nil if there is no target.
	LookAt *math3d.Vector3

	// The index of the gait which should be used, mod however many gaits are
	// available. (This doesn't really belong here, but is the simplest way to
	// pass the selection from the controller to the chassis and I am lazy.)
	GaitIndex int

	// The increase (or decrease, if negative) from the default speed at which
	// we should walk. There is no unit; more is just faster.
	Speed int
}

// World returns a matrix to transform a vector in the coordinate space defined
// by the Position and Rotation attributes into the world space.
// TODO: Remove this method.
func (s *State) World() math3d.Matrix44 {
	return s.Pose.Add(math3d.Pose{s.Offset, 0, 0, 0}).ToWorld()
}

// Local returns a matrix to transform a vector in the world coordinate space
// into the space defined by the state (using the Position and Rotation attrs).
// TODO: Remove this method.
func (s *State) Local() math3d.Matrix44 {
	return s.Pose.Add(math3d.Pose{s.Offset, 0, 0, 0}).ToLocal()
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

	// The FPS which the main loop should try to run at.
	TargetFPS int

	// To count the number of times that Tick is called each second.
	fc *utils.FrameCounter

	// The time at which an FPS warning was last logged. To avoid flooding the
	// logs if we're running too slowly.
	prevWarnFPS time.Time
}

type Component interface {
	Boot() error
	Tick(time.Time, *State) error
}

// NewHexapod creates a new Hexapod object on the given Dynamixel network.
func NewHexapod(network *network.Network, targetFPS int) *Hexapod {
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
			Offset: math3d.Vector3{0, 0, 0},
			Target: math3d.Pose{
				Position: math3d.ZeroVector3,
				Heading:  0,
			},
			LookAt:    nil,
			GaitIndex: 0,
			Speed:     0,
		},
		TargetFPS: targetFPS,
		fc:        utils.NewFrameCounter(time.Second),
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

	// Trigger any buffered instructions written during boot.
	h.ActionInstruction()

	return nil
}

var log = logrus.WithFields(logrus.Fields{
	"pkg": "hex",
})

// Tick calls Tick on each component, then sends the ACTION instruction to
// trigger any buffered instructions.
func (h *Hexapod) Tick(now time.Time) error {

	// Lock the network during tick. Any other goroutines wanting to hit the
	// network (e.g. legs.waitForReady) must first acquire the lock.
	h.Network.Lock()
	defer h.Network.Unlock()

	// Update the fps counter.
	h.fc.Frame(now)
	h.State.FPS = h.fc.Count()

	// Send Tick to every component.
	for _, c := range h.Components {
		err := c.Tick(now, h.State)
		if err != nil {
			return fmt.Errorf("%T.Tick returned error: %v", c, err)
		}
	}

	if h.State.FPS < h.TargetFPS {
		if now.Sub(h.prevWarnFPS) > 5*time.Second {
			log.Warnf("fps=%d, target=%d", h.State.FPS, h.TargetFPS)
			h.prevWarnFPS = now
		}
	}

	// Trigger any buffered instructions written during this tick.
	h.ActionInstruction()

	return nil
}

func (h *Hexapod) ActionInstruction() error {
	for i := range h.Protocols {
		err := h.Protocols[i].Action()
		if err != nil {
			return err
		}
	}

	return nil
}

// TODO: Move this stuff to a separate package.

var log2 = logrus.WithFields(logrus.Fields{
	"pkg": "http",
})

// Remote starts an HTTP server which can update the configuration. It blocks
// forever, so start it in a goroutine.
func (h *Hexapod) RunServer(port int) {
	indexHTML := ""

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<pre>%s</pre>", indexHTML)
	})

	addr := fmt.Sprintf(":%d", port)
	log2.Infof("listening on %s", addr)
	err := http.ListenAndServe(addr, nil)
	panic(err)
}
