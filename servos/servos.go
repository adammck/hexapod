package servos

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/adammck/dynamixel/network"
	"github.com/adammck/dynamixel/servo"
	"github.com/adammck/dynamixel/servo/ax"
)

type Pool []*servo.Servo

var servos Pool

// New adds a Servo (with sensible defaults) to the pool.
func New(n *network.Network, ID int) (*servo.Servo, error) {
	s, err := ax.New(n, ID)
	if err != nil {
		return nil, err
	}

	// Don't bother sending ACKs for writes. We must do this first, to ensure
	// that the servos are in the expected state before sending other commands.
	err = s.SetReturnLevel(1)
	if err != nil {
		return nil, fmt.Errorf("%s (while setting return level)", err)
	}

	// Add to the pool as soon as we know the servo is available, to ensure that
	// we power it down at shutdown even if the next lines fail.
	servos = append(servos, s)

	err = s.SetReturnDelayTime(0)
	if err != nil {
		return nil, fmt.Errorf("%s (while setting return delay)", err)
	}

	return s, nil
}

// Shutdown powers off all servos in the pool. This should be called before
// terminating the program, to ensure that servos don't stay powered up
// indefinitely.
func Shutdown() {
	for _, s := range servos {
		s.SetBuffered(false)

		err := s.SetMovingSpeed(0)
		if err != nil {
			log.Warnf("%s (while resetting moving speed)", err)
		}

		err = s.SetTorqueLimit(0)
		if err != nil {
			log.Warnf("%s (while resetting torque limit)", err)
		}

		err = s.SetTorqueEnable(false)
		if err != nil {
			log.Warnf("%s (while disabling torque)", err)
		}

		err = s.SetLED(false)
		if err != nil {
			log.Warnf("%s (while disabling LED)", err)
		}
	}
}
