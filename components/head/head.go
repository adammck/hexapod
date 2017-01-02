package head

import (
	"fmt"
	"math"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/adammck/dynamixel/servo"
	"github.com/adammck/hexapod"
	"github.com/adammck/hexapod/math3d"
	"github.com/adammck/hexapod/utils"
)

var log = logrus.WithFields(logrus.Fields{
	"pkg": "head",
})

const (
	moveSpeed   = 1023
	torqueLimit = 1023
)

type Config struct {
	UpLimit    float64
	DownLimit  float64
	LeftLimit  float64
	RightLimit float64
}

var defaultConfig = &Config{
	UpLimit:    10,
	DownLimit:  -20,
	LeftLimit:  -45,
	RightLimit: 45,
}

type Head struct {
	o math3d.Pose
	h *servo.Servo
	v *servo.Servo
	c *Config
}

func New(o math3d.Pose, h, v *servo.Servo) *Head {
	return &Head{o, h, v, defaultConfig}
}

func (h *Head) Servos() []*servo.Servo {
	return []*servo.Servo{
		h.h,
		h.v,
	}
}

func (h *Head) Boot() error {
	for _, s := range h.Servos() {

		err := s.SetMovingSpeed(moveSpeed)
		if err != nil {
			return fmt.Errorf("%s (while setting move speed)", err)
		}

		err = s.SetTorqueLimit(torqueLimit)
		if err != nil {
			return fmt.Errorf("%s (while setting torque limit)", err)
		}

		// Buffer all moves until end of tick.
		s.SetBuffered(true)
	}

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
	x := 0 - utils.Deg(math.Atan(v.X/v.Z))
	y := 0 - utils.Deg(math.Atan(v.Y/v.Z))

	// Constrain angles to avoid mechanical damage.
	// TODO: Encapsulate this in a Servo object, or use the limit registers.
	x = math.Max(math.Min(x, h.c.RightLimit), h.c.LeftLimit)
	y = math.Max(math.Min(y, h.c.UpLimit), h.c.DownLimit)

	// Update servos every tick.
	// TODO: Maybe only update if the x/y has changed.
	h.h.MoveTo(x)
	h.v.MoveTo(y)
	return nil
}
