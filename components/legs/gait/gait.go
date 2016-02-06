package gait

import (
	"math"
)

type Frame struct {
	XZ float64
	Y  float64
}

type FrameList []Frame
type Gait []FrameList

const (
	numLegs = 6

	// The offset (on the Y axis) which feet should be moved to on the up step,
	// relative to the origin.
	baseFootUp = 40.0
)

func TheGait(ticksPerStepCycle, ticksPerStep int) Gait {
	gait := make(Gait, numLegs)
	p := float64(ticksPerStepCycle) / 4.0

	// TODO: Encapsulate this, along with the other curve properties, into some
	//       sort of Gait object, to make them pluggable.
	stepCurveCenter := [numLegs]float64{
		0: p,
		1: p * 3,
		2: p,
		3: p * 3,
		4: p,
		5: p * 3,
	}

	for i := 0; i < numLegs; i += 1 {
		gait[i] = singleLegGait(ticksPerStepCycle, ticksPerStep, stepCurveCenter[i])
	}

	return gait
}

func singleLegGait(ticksPerStepCycle, ticksPerStep int, stepCurveCenter float64) FrameList {
	frameList := make(FrameList, ticksPerStepCycle)
	tps := float64(ticksPerStep)

	curveStart := stepCurveCenter - tps/2
	curveEnd := stepCurveCenter + tps/2

	for i := 0.0; i < float64(ticksPerStepCycle); i += 1.0 {
		f := Frame{}

		// Step height is a bell curve
		f.Y = baseFootUp * math.Pow(2, -math.Pow((i-stepCurveCenter)*((math.E*2)/tps), 2))

		// Step movement ratio is a sine from 0 to 1
		if i < curveStart {
			f.XZ = 0.0

		} else if i > curveEnd {
			f.XZ = 1.0

		} else {
			x := (i - curveStart) / tps
			f.XZ = 0.5 - (math.Cos(x*math.Pi) / 2)
		}

		frameList[int(i)] = f
	}

	return frameList
}
