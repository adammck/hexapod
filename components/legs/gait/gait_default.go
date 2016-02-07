package gait

import (
	"math"
)

func TheGait(ticksPerStep int) Gait {
	ticksPerStepCycle := ticksPerStep * 3
	cc := curveCenters(2, ticksPerStepCycle)

	var legs [numLegs]Frames
	for i := 0; i < numLegs; i += 1 {
		legs[i] = singleLegGait(ticksPerStepCycle, ticksPerStep, cc[i])
	}

	return Gait{
		legs:   legs,
		length: ticksPerStepCycle,
	}
}

// curveCenters
func curveCenters(groupSize int, ticksPerStepCycle int) [numLegs]float64 {
	p := float64(ticksPerStepCycle) / 12
	switch groupSize {

	// Move one leg at a time (six groups):
	//
	// |1|2|3|4|5|6|7|8|9|0|1|2|
	// |1|1|1|1|1|1|1|1|1|1|1|1|
	//   ^   ^   ^   ^   ^   ^
	//   1   3   5   7   9  11
	case 1:
		return [numLegs]float64{
			0: p * 1,
			1: p * 3,
			2: p * 5,
			3: p * 7,
			4: p * 9,
			5: p * 11,
		}

	// Two at a time (three groups):
	//
	// |1|2|3|4|5|6|7|8|9|0|1|2|
	// |-2-|---4---|---4---|-2-|
	//     ^       ^       ^
	//     2       6      10
	case 2:
		return [numLegs]float64{
			0: p * 2,
			1: p * 6,
			2: p * 10,
			3: p * 2,
			4: p * 6,
			5: p * 10,
		}

	// Three (two groups):
	//
	// |1|2|3|4|5|6|7|8|9|0|1|2|
	// |--3--|-----6-----|--3--|
	//       ^           ^
	//       3           9
	case 3:
		return [numLegs]float64{
			0: p * 3,
			1: p * 9,
			2: p * 3,
			3: p * 9,
			4: p * 3,
			5: p * 9,
		}

	default:
		panic("invalid groupSize")
	}
}

func singleLegGait(ticksPerStepCycle, ticksPerStep int, stepCurveCenter float64) Frames {
	frameList := make(Frames, ticksPerStepCycle)
	tps := float64(ticksPerStep)

	curveStart := stepCurveCenter - tps/2
	curveEnd := stepCurveCenter + tps/2

	for i := 0.0; i < float64(ticksPerStepCycle); i += 1.0 {
		f := Frame{}

		// Step height is a bell curve
		f.Y = math.Pow(2, -math.Pow((i-stepCurveCenter)*((math.E*2)/tps), 2))

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
