package gait

const (
	numLegs = 6
)

type Frame struct {
	XZ float64
	Y  float64
}

type Frames []Frame

type Gait struct {
	legs   [numLegs]Frames
	length int
}

// Length returns the number of ticks necessary to complete a full cycle of the
// gait, such that the feet are back in their original position relative to the
// origin.
func (g *Gait) Length() int {
	return g.length
}

// Frame returns the frame (containing the XZ/Y ratios) for the given leg index
// at the given frame number. This is just to spare the caller from checking the
// bounds of the slices.
func (g *Gait) Frame(leg int, n int) Frame {
	return g.legs[leg][n]
}
