package controller

import (
	"fmt"
	"github.com/adammck/hexapod"
	"github.com/adammck/hexapod/math3d"
	"github.com/adammck/sixaxis"
	"io"
	"time"
)

const (

	// Maximum movement speed.
	// TODO: What the hell is the unit here?
	moveSpeed = 1.5

	// The maximum speed to rotate (i.e. when the right stick is fully pressed)
	// in degrees per loop.
	rotationSpeed = 0.8
)

type Controller struct {
	hex *hexapod.Hexapod
	sa  *sixaxis.SA
}

func New(hex *hexapod.Hexapod, r io.Reader) *Controller {
	return &Controller{
		hex: hex,
		sa:  sixaxis.New(r),
	}
}

// TODO: Log
func (c *Controller) Boot() error {
	go c.sa.Run()
	return nil
}

// TODO: Update the state of the hexapod based on the state of the controller.
func (c *Controller) Tick(now time.Time) error {

	// Rotate with the right stick
	if c.sa.RightStick.X != 0 {
		c.hex.Rotation += (float64(c.sa.RightStick.X) / 127.0) * rotationSpeed
	}

	// How much the origin should move this frame. Default is zero, but this
	// it mutated (below) by the various buttons.
	vecMove := math3d.MakeVector3(0, 0, 0)

	if c.sa.LeftStick.X != 0 {
		vecMove.X = (float64(c.sa.LeftStick.X) / 127.0) * moveSpeed
	}

	if c.sa.LeftStick.Y != 0 {
		vecMove.Z = (float64(-c.sa.LeftStick.Y) / 127.0) * moveSpeed
	}

	// Move the origin up (away from the ground) with the dpad. This alters
	// the gait by keeping the body up in the air. It looks weird but works.
	if c.sa.Up > 0 {
		vecMove.Y = +2
	}

	if c.sa.Down > 0 {
		vecMove.Y = -2
	}

	// Update the position, if it's changed.
	if !vecMove.Zero() {
		c.hex.Position = vecMove.MultiplyByMatrix44(c.hex.World())
	}

	//dontMove = (c.sa.Square > 0)

	// At any time, pressing start shuts down the hex.
	if c.sa.Start {
		fmt.Println("Pressed START, shutting down")
		c.hex.Shutdown = true
	}

	return nil
}
