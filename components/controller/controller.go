package controller

import (
	"github.com/adammck/hexapod"
	"github.com/adammck/sixaxis"
	"io"
	"time"
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
	return nil
}
