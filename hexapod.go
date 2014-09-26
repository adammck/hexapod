package hexapod

import (
	"github.com/adammck/dynamixel"
	"github.com/jacobsa/go-serial/serial"
	"time"
	"math"
	"fmt"
)

type Hexapod struct {
	Network *dynamixel.DynamixelNetwork
	Legs    [6]*Leg
}

//
// NewHexapod creates a new Hexapod object on the given Dynamixel network.
//
func NewHexapod(network *dynamixel.DynamixelNetwork) *Hexapod {
	return &Hexapod{
		Network: network,
		Legs: [6]*Leg{

			// Points are the X/Y/Z offsets from the center of the top of the body to
			// the center of the coxa pivots.
			NewLeg(network, 10, NewPoint(-51.1769, -19,  98), -120), // Front Left  - 0
			NewLeg(network, 20, NewPoint(51.1769,  -19,  98),  -60), // Front Right - 1
			NewLeg(network, 30, NewPoint(66,       -19,   0),    0), // Mid Right   - 2
			NewLeg(network, 40, NewPoint(51.1769,  -19, -98),   60), // Back Right  - 3
			NewLeg(network, 50, NewPoint(-51.1769, -19, -98),  120), // Back Left   - 4
			NewLeg(network, 60, NewPoint(-66,      -19,   0),  180), // Mid Left    - 5
		},
	}
}

//
// NewHexapodFromPortName creates a new Hexapod object by opening the given
// serial port with the default options. This only exists to reduce boilerplate
// in my development utils.
//
// Note: This only works with OSX for the time being, because the upstream
//       serial port library (jacobsa/go-serial), while being otherwise super,
//       only supports OSX. There are other serial port libraries which support
//       Linux and Windows, but you'll have to instantiate those yourself.
//
func NewHexapodFromPortName(portName string) (*Hexapod, error) {
	options := serial.OpenOptions{
		PortName:              portName,
		BaudRate:              1000000,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       0,
		InterCharacterTimeout: 100,
	}

	serial, openErr := serial.Open(options)
	if openErr != nil {
		return nil, openErr
	}

	network := dynamixel.NewNetwork(serial)
	hexapod := NewHexapod(network)
	return hexapod, nil
}

//
// Sync runs the given function while the network is in buffered mode, then
// initiates any movements at once by sending ACTION.
//
func (hexapod *Hexapod) Sync(f func()) {
	hexapod.Network.SetBuffered(true)
	f()
	hexapod.Network.SetBuffered(false)
	hexapod.Network.Action()
}

//
// SyncLegs runs the given function once for each leg while the network is in
// buffered mode, then initiates movements with ACTION. This is useful when
// resetting everything to a known state.
//
func (hexapod *Hexapod) SyncLegs(f func(leg *Leg)) {
	hexapod.Sync(func() {
		for _, leg := range hexapod.Legs {
			f(leg)
		}
	})
}

// setMoveSpeed sets the moving speed of all servos. This is only really useful
// for testing and debugging.
func (hexapod *Hexapod) setMoveSpeed(speed int) {
	for _, leg := range hexapod.Legs {
		for _, servo := range leg.Servos() {
			servo.SetTorqueEnable(true)
			servo.SetMovingSpeed(speed)
		}
	}
}


// Demo moves the legs around arbitrarily to demonstrate that everything is
// working as it should.
func (hexapod *Hexapod) Demo() {
	hexapod.setMoveSpeed(128)

	radius := 230.0
	step := -10.0
	min := -200.0
	max := -70.0
	y := max

	for {

		hexapod.Sync(func() {
			fmt.Printf("----> %0.4f\n", y)
			for _, leg := range hexapod.Legs {
				x := math.Cos(rad(leg.Angle)) * radius
				z := math.Sin(rad(leg.Angle)) * radius
				leg.SetGoal(x, y, -z)
			}

			//time.Sleep(20 * time.Millisecond)
		})

		y += step
		if y < min || y > max {
			step = 0 - step
		}
	}
}

//
// Shutdown moves all servos to a hard-coded default position, then turns them
// off. This should be called when finished
//
func (hexapod *Hexapod) Shutdown() {
	for _, leg := range hexapod.Legs {
		for _, servo := range leg.Servos() {
			servo.SetTorqueEnable(true)
			servo.SetMovingSpeed(128)
		}
	}

	hexapod.SyncLegs(func(leg *Leg) {
		leg.Coxa.MoveTo(0)
		leg.Femur.MoveTo(-60)
		leg.Tibia.MoveTo(60)
		leg.Tarsus.MoveTo(60)
	})

	// TODO: Wait for servos to stop moving, instead of hard-coding a timer.
	wait(2000)

	for _, leg := range hexapod.Legs {
		for _, servo := range leg.Servos() {
			servo.SetTorqueEnable(false)
			servo.SetLed(false)
		}
	}
}

func wait(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func (hexapod *Hexapod) setLegs(c float64, f float64, t float64, tt float64) {
	hexapod.SyncLegs(func(leg *Leg) {
		leg.Coxa.MoveTo(c)
		leg.Femur.MoveTo(f)
		leg.Tibia.MoveTo(t)
		leg.Tarsus.MoveTo(tt)
	})
}
