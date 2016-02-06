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
	return fmt.Sprintf("&Vec3{x=%+07.2f y=%+07.2f z=%+07.2f}", v.X, v.Y, v.Z)
}

// Zero returns true if the vector is at 0,0,0.
func (v Vector3) Zero() bool {
	return (v.X == 0) && (v.Y == 0) && (v.Z == 0)
}

// Add adds two vectors, and returns a pointer to the result.
// TODO: Return a value rather than a pointer. The caller can always create a
//       pointer if they really want.
func (v Vector3) Add(vv Vector3) *Vector3 {
	return &Vector3{
		(v.X + vv.X),
		(v.Y + vv.Y),
		(v.Z + vv.Z),
	}
}

// Subtract one vector from another.
func (v Vector3) Subtract(vv Vector3) Vector3 {
	return Vector3{
		(v.X - vv.X),
		(v.Y - vv.Y),
		(v.Z - vv.Z),
	}
}

// Distance calculates and returns the distance between this vector and another.
// TODO: Remove this; users should just Subtract and Magnitude themselves.
func (v Vector3) Distance(vv Vector3) float64 {
	return v.Subtract(vv).Magnitude()
}

func (v Vector3) Magnitude() float64 {
	return math.Sqrt((v.X * v.X) + (v.Y * v.Y) + (v.Z * v.Z))
}

// Unit returns the vector scaled to a length of 1, such that it represents a
// direction rather than a point.
func (v Vector3) Unit() Vector3 {
	m := v.Magnitude()
	if m == 0 {
		return ZeroVector3
	}

	return Vector3{
		X: v.X / m,
		Y: v.Y / m,
		Z: v.Z / m,
	}
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

// MultiplyByScaler returns a new vector by multiply each attribute by the given
// scalar. This can be used to project a distance along the vector.
func (v Vector3) MultiplyByScalar(s float64) Vector3 {
	return Vector3{
		(v.X * s),
		(v.Y * s),
		(v.Z * s),
	}
}
