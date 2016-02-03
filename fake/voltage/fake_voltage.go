package voltage

type FakeVoltage struct {
	voltage float64
}

func New(voltage float64) *FakeVoltage {
	return &FakeVoltage{voltage}
}

func (s FakeVoltage) Voltage() (float64, error) {
	return s.voltage, nil
}
