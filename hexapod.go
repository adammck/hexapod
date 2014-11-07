package hexapod

import (
	"fmt"
	"github.com/adammck/dynamixel"
	"github.com/adammck/retroport"
	"github.com/jacobsa/go-serial/serial"
	"math"
	"time"
)

type Hexapod struct {
	Network         *dynamixel.DynamixelNetwork
	Controller      *retroport.SNES
	CurrentPosition Point3d
	TargetPosition  Point3d
	CurrentRotation float64
	TargetRotation  float64
	Legs            [6]*Leg
}

//
// NewHexapod creates a new Hexapod object on the given Dynamixel network.
//
func NewHexapod(network *dynamixel.DynamixelNetwork) *Hexapod {
	return &Hexapod{
		Network:         network,
		CurrentPosition: Point3d{0, 0, 0},
		TargetPosition:  Point3d{0, 0, 0},
		CurrentRotation: 0.0,
		TargetRotation:  0.0,
		Legs: [6]*Leg{

			// Points are the X/Y/Z offsets from the center of the top of the body to
			// the center of the coxa pivots.
			NewLeg(network, 10, "FL", NewPoint(-51.1769, -19, 98), -120), // Front Left  - 0
			NewLeg(network, 20, "FR", NewPoint(51.1769, -19, 98), -60),   // Front Right - 1
			NewLeg(network, 30, "MR", NewPoint(66, -19, 0), 0),           // Mid Right   - 2
			NewLeg(network, 40, "BR", NewPoint(51.1769, -19, -98), 60),   // Back Right  - 3
			NewLeg(network, 50, "BL", NewPoint(-51.1769, -19, -98), 120), // Back Left   - 4
			NewLeg(network, 60, "ML", NewPoint(-66, -19, 0), 180),        // Mid Left    - 5
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

func (hexapod *Hexapod) SyncWait(f func(), ms int) {
	hexapod.Sync(f)
	time.Sleep(time.Duration(ms) * time.Millisecond)
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

	radius := 220.0
	step := -5.0
	min := -220.0
	max := 30.0
	y := max

	for {
		hexapod.Sync(func() {
			fmt.Printf("----> %0.4f\n", y)
			for _, leg := range hexapod.Legs {
				x := math.Cos(rad(leg.Angle)) * radius
				z := math.Sin(rad(leg.Angle)) * radius
				leg.SetGoal(Point3d{x, y, -z})
			}

			//time.Sleep(200 * time.Millisecond)
		})

		if min == max {
			break
		}

		y += step
		if y <= min || y >= max {
			step = 0 - step
		}
	}
}

// MainLoop watches for changes to the target position and rotation, and tries
// to apply it as gracefully as possible.
func (hexapod *Hexapod) MainLoop() {
	for {
		if hexapod.TargetRotation != hexapod.CurrentRotation {
			fmt.Printf("Rotate to: %0.4f\n", hexapod.TargetRotation)
			hexapod.CurrentRotation = hexapod.TargetRotation
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// Rotate just rotates the hexapod in a counter-clockwise circle forever.
func (h *Hexapod) CrapRotate() {
	h.setMoveSpeed(256)

	// settings
	radius := 220.0
	footDown := -100.0
	footUp := -60.0
	sleep := 100
	stepSize := 50.0

	// controls
	up := false
	rot := 0.0
	mov := Point3d{}
	quit := false

	// state
	isUp := false

	// move legs in groups of two, for stability
	legSets := [][]int{
		[]int{0, 3},
		[]int{1, 4},
		[]int{2, 5},
	}

	// or three, for maximum speed
	// legSets := [][]int{
	// 	[]int{0, 2, 4},
	// 	[]int{1, 3, 5},
	// }

	footPos := func(debug bool, leg *Leg, angle float64, movX float64, movZ float64, altitude float64) Point3d {
		r := rad(leg.Angle-angle)

		x := math.Cos(r) * radius
		z := math.Sin(r) * radius
		p := Point3d{x, altitude, -z}

		if debug {
			// opp := (x+movX)-leg.Origin.X
			// adj := (-z+movZ) - leg.Origin.Z
			// theta := math.Atan2(-opp, adj)
			fmt.Printf("%s: %+0.4f,%+0.4f -> %+0.4f,%+0.4f -> %+0.4f,%+0.4f\n", leg.Name, leg.Origin.X, leg.Origin.Z, x, -z, x+movX, (-z)+movZ)
			//fmt.Printf("%s: adj=%0.4f, opp=%0.4f, theta=%0.4f\n", leg.Name, opp, adj, theta)
		}

		return p
	}

	setFoot := func(debug bool, leg *Leg, angle float64, movX float64, movZ float64, altitude float64) {
		leg.SetGoal(footPos(debug, leg, angle, movX, movZ, altitude))
	}

	setFeet := func(debug bool, i int, angle float64, movX float64, movZ float64, altitude float64) {
		for _, ii := range legSets[i] {
			setFoot(debug, h.Legs[ii], angle, movX, movZ, altitude)
		}
	}

	// main loop!

	for {

		// READ STATE

		rot = 0
		mov = Point3d{0, 0, 0}

		if h.Controller.L {
			rot = 20
		}

		if h.Controller.R {
			rot = -20
		}

		if h.Controller.Up {
			mov.Z = stepSize
		}

		if h.Controller.Down {
			mov.Z = -stepSize
		}

		if h.Controller.Left {
			mov.X = -stepSize
		}

		if h.Controller.Right {
			mov.X = stepSize
		}

		// Lower body
		if h.Controller.Y {
			footDown += 10
			isUp = false
		}

		// Raise body
		if h.Controller.X {
			footDown -= 10
			isUp = false
		}

		// toggle active state
		if h.Controller.Start {
			up = !up
		}

		// quit
		if h.Controller.Select {
			up = false
			quit = true
		}

		// MOVE

		// stand up
		if up && !isUp {
			h.SyncWait(func() {
				for _, leg := range h.Legs {
					setFoot(false, leg, 0, 0, 0, footDown)
				}
			}, 100)
			isUp = true

			// sit down
		} else if !up && isUp {
			h.SyncWait(func() {
				for _, leg := range h.Legs {
					setFoot(false, leg, 0, 0, 0, footUp)
				}
			}, 1000)
			isUp = false
		}

		if !quit && isUp && (rot != 0 || mov.X != 0 || mov.Z != 0) {
			for i, _ := range legSets {

				// three-step:
				// * raise the foot
				// * move to the target offset
				// * lower the foot
				h.SyncWait(func() { setFeet(false, i, 0,   0,     0,    footUp) }, sleep)
				h.SyncWait(func() { setFeet(false, i, rot, mov.X, mov.Z, footUp) }, sleep)
				h.SyncWait(func() { setFeet(false, i, rot, mov.X, mov.Z, footDown) }, sleep)
				//h.SyncWait(func() { setFeet(false, i, 0, 0, 0, footUp) }, sleep)
				//h.SyncWait(func() { setFeet(false, i, 0, 0, 0, footDown) }, sleep)

				// two-step:
				// h.SyncWait(func() { setFeet(i, rot*0.5, footUp) }, sleep)
				// h.SyncWait(func() { setFeet(i, rot, footDown) }, sleep)
			}

			time.Sleep(50 * time.Millisecond)

			// untwist
			h.SyncWait(func() {
				for _, leg := range h.Legs {
					setFoot(false, leg, 0, 0, 0, footDown)
				}
			}, sleep)

			// yield if there's nothing to do
		} else {
			time.Sleep(20 * time.Millisecond)
		}

		if quit {
			h.Relax()
			return
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
	hexapod.Relax()
}

func (hexapod *Hexapod) Relax() {
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
