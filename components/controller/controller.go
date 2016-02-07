package controller

import (
	log "github.com/Sirupsen/logrus"
	"github.com/adammck/hexapod"
	"github.com/adammck/hexapod/math3d"
	"github.com/adammck/sixaxis"
	"io"
	"time"
)

const (
	moveSpeed = 10.0
	rotSpeed  = 0.8
)

type Controller struct {
	sa *sixaxis.SA
}

func New(r io.Reader) *Controller {
	return &Controller{
		sa: sixaxis.New(r),
	}
}

func (c *Controller) Boot() error {
	go c.sa.Run()
	return nil
}

func (c *Controller) Tick(now time.Time, state *hexapod.State) error {

	// Set the target position by creating a vector from the left stick (for the
	// X/Z axes), and the dpad (for the Z axis or ground clearance), and adding
	// it to the current position.

	v := math3d.ZeroVector3
	if c.sa.LeftStick.X > 2.0 {
		v.X = (float64(c.sa.LeftStick.X) / 127.0) * moveSpeed
	}
	if c.sa.LeftStick.Y > 2.0 {
		v.Z = (float64(-c.sa.LeftStick.Y) / 127.0) * moveSpeed
	}
	if c.sa.Up > 0 {
		v.Y = 2
	}
	if c.sa.Down > 0 {
		v.Y = 2
	}
	if !v.Zero() {
		state.TargetPosition = *state.Position.Add(v)
	}

	// Rotate with the right stick
	if c.sa.RightStick.X > 2.0 {
		r := (float64(c.sa.RightStick.X) / 127.0) * rotSpeed
		state.TargetRotation = state.Rotation + r
	}

	// At any time, pressing start shuts down the hex.
	if c.sa.Start {
		log.Warn("Pressed START, shutting down")
		state.Shutdown = true
	}

	return nil
}
