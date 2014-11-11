package hexapod

import (
	"math"
)

type Point3d struct {
	X float64
	Y float64
	Z float64
}

func NewPoint(X float64, Y float64, Z float64) *Point3d {
	return &Point3d{
		X: X,
		Y: Y,
		Z: Z,
	}
}

func (p *Point3d) Zero() bool {
	return (p.X == 0) && (p.Y == 0) && (p.Z == 0)
}

// Distance calculates and returns the distance between this vector and another,
// as a float64.
func (p *Point3d) Distance(pp Point3d) float64 {
	dx := p.X - pp.X
	dy := p.Y - pp.Y
	dz := p.Z - pp.Z
	return math.Sqrt((dx * dx) + (dy * dy) + (dz * dz))
}