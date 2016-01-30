package voltage

import (
	"fmt"
	"github.com/adammck/hexapod"
	"time"
)

const (

	// The number of seconds between voltage checks. These are pretty quick, but
	// not instant. Running at low voltage for too long will damage the battery,
	// so it should be checked pretty regularly.
	interval = 5

	// The voltage at which the hexapod should shut down.
	minimum = 9.6
)

type HasVoltage interface {
	Voltage() (float64, error)
}

type VoltageCheck struct {
	t time.Time
	HasVoltage
}

func New(servo HasVoltage) *VoltageCheck {
	return &VoltageCheck{
		time.Time{},
		servo,
	}
}

func (vc *VoltageCheck) Boot() error {
	return nil
}

func (vc *VoltageCheck) Tick(now time.Time, state *hexapod.State) error {
	if vc.NeedsVoltageCheck() {
		return vc.CheckVoltage()
	}

	return nil
}

// NeedsVoltageCheck returns true if it's been a while since we checked the
// voltage level. The timeout is pretty arbitrary.
func (vc *VoltageCheck) NeedsVoltageCheck() bool {
	return time.Since(vc.t) > (interval * time.Second)
}

// CheckVoltage fetches the voltage level of an arbitrary servo, and returns an
// error if it's too low. In this case, the program should be terminated as soon
// as possible to preserve the battery.
func (vc *VoltageCheck) CheckVoltage() error {
	val, err := vc.Voltage()
	vc.t = time.Now()
	if err != nil {
		return err
	}

	fmt.Printf("voltage: %.2fv\n", val)

	if val < minimum {
		return fmt.Errorf("low voltage: %.2fv", val)
	}

	return nil
}
