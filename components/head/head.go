package head

import (
	log "github.com/Sirupsen/logrus"
	"github.com/adammck/dynamixel/servo"
	"github.com/adammck/hexapod"
	"github.com/adammck/hexapod/math3d"
	"github.com/adammck/hexapod/utils"
	"math"
	"time"
)

const (
	upLimit    = 10
	downLimit  = -10
	leftLimit  = -20
	rightLimit = 20
)

var logger = log.WithFields(log.Fields{
	"pkg": "head",
})

type Head struct {
	o math3d.Pose
	h *servo.Servo
	v *servo.Servo
}

func New(o math3d.Pose, h, v *servo.Servo) *Head {
	return &Head{o, h, v}
}

func (h *Head) Boot() error {
	return nil
}

func (h *Head) Tick(now time.Time, state *hexapod.State) error {

	// Nothing to do if there is no target.
	if state.LookAt == nil {
		return nil
	}

	// Transform the lookat vector into the hexapod space, then into the head
	// space, such that the point at the origin of the head is [0, 0, 0].
	v := state.LookAt.MultiplyByMatrix44(state.Pose.ToLocal()).MultiplyByMatrix44(h.o.ToLocal())

	// Transform the vector (from the origin to the dest) into horizontal (X)
	// and vertical (Y) angles. There is no Z between two vectors.
	x := utils.Deg(math.Atan(v.X / v.Z))
	y := utils.Deg(math.Atan(v.Y / v.Z))

	// Constrain angles to avoid mechanical damage.
	// TODO: Encapsulate this in a Servo object, or use the limit registers.
	x = math.Max(math.Min(x, rightLimit), leftLimit)
	y = math.Max(math.Min(y, upLimit), downLimit)

	//
	h.h.MoveTo(x)
	h.v.MoveTo(y)

	log.Infof("head=%v, h=%v, v=%v", v, x, y)
	return nil
}
