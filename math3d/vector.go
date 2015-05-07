package math3d

import (
	"fmt"
	"math"
)

type Vector3 struct {
	X float64
	Y float64
	Z float64
}

var (
	ZeroVector3 = Vector3{}
)

// MakeVector3 returns a pointer to a new Vector3.
func MakeVector3(x float64, y float64, z float64) *Vector3 {
	return &Vector3{x, y, z}
}

func (v Vector3) String() string {
	return fmt.Sprintf("&Vec3{x=%0.2f y=%0.2f z=%0.2f}", v.X, v.Y, v.Z)
}

// Zero returns true if the vector is at 0,0,0.
func (v Vector3) Zero() bool {
	return (v.X == 0) && (v.Y == 0) && (v.Z == 0)
}

// Add adds two vectors, and returns a pointer to the result.
func (v Vector3) Add(vv Vector3) *Vector3 {
	return &Vector3{
		(v.X + vv.X),
		(v.Y + vv.Y),
		(v.Z + vv.Z),
	}
}

// Distance calculates and returns the distance between this vector and another,
// as a float64.
func (v Vector3) Distance(vv Vector3) float64 {
	dx := v.X - vv.X
	dy := v.Y - vv.Y
	dz := v.Z - vv.Z
	return math.Sqrt((dx * dx) + (dy * dy) + (dz * dz))
}

// MultiplyByMatrix44 returns a new Vector3, by multiplying this vector my a 4x4
// matrix.
func (v Vector3) MultiplyByMatrix44(m Matrix44) Vector3 {
	return Vector3{
		(v.X * m.m11) + (v.Y * m.m21) + (v.Z * m.m31) + m.m41,
		(v.X * m.m12) + (v.Y * m.m22) + (v.Z * m.m32) + m.m42,
		(v.X * m.m13) + (v.Y * m.m23) + (v.Z * m.m33) + m.m43,
	}
}
