package utils

import (
	"fmt"
	"github.com/adammck/dynamixel/network"
	"github.com/adammck/dynamixel/servo"
	"github.com/adammck/dynamixel/servo/ax"
	"math"
)

func Deg(rads float64) float64 {
	return rads / (math.Pi / 180)
}

func Rad(degrees float64) float64 {
	return (math.Pi / 180) * degrees
}

// Sync runs the given function while the network is in buffered mode, then
// initiates any movements at once by sending ACTION.
func Sync(n *network.Network, f func()) {
	n.SetBuffered(true)
	defer n.SetBuffered(false)
	defer n.Action()
	f()
}

// Servo returns a new Servo with sensible defaults.
func Servo(n *network.Network, ID int) (*servo.Servo, error) {
	s, err := ax.New(n, ID)
	if err != nil {
		return nil, err
	}

	err = s.SetStatusReturnLevel(1)
	if err != nil {
		return nil, fmt.Errorf("%s (while setting return level)", err)
	}

	err = s.Ping()
	if err != nil {
		return nil, fmt.Errorf("%s (while pinging)", err)
	}

	err = s.SetTorqueEnable(true)
	if err != nil {
		return nil, fmt.Errorf("%s (while enabling torque)", err)
	}

	err = s.SetMovingSpeed(1023)
	if err != nil {
		return nil, fmt.Errorf("%s (while setting move speed)", err)
	}

	return s, nil
}
