package math3d

import (
	"fmt"
	"math"
)

type Matrix44 struct {
	m11 float64 // 0
	m12 float64 // 1
	m13 float64 // 2
	m14 float64 // 3
	m21 float64 // 4
	m22 float64 // 5
	m23 float64 // 6
	m24 float64 // 7
	m31 float64 // 8
	m32 float64 // 9
	m33 float64 // 10
	m34 float64 // 11
	m41 float64 // 12
	m42 float64 // 13
	m43 float64 // 14
	m44 float64 // 15
}

func MakeMatrix44(v Vector3, ea EulerAngles) *Matrix44 {
	m := &Matrix44{}
	m.SetRotation(ea)
	m.SetTranslation(v)
	return m
}

func (m Matrix44) String() string {
	return fmt.Sprintf(
		"&M44{%+.4f %+.4f %+.4f %+.4f | %+.4f %+.4f %+.4f %+.4f | %+.4f %+.4f %+.4f %+.4f | %+.4f %+.4f %+.4f %+.4f}",
		m.m11, m.m12, m.m13, m.m14,
		m.m21, m.m22, m.m23, m.m24,
		m.m31, m.m32, m.m33, m.m34,
		m.m41, m.m42, m.m43, m.m44)
}

// Array returns the matrix as a 4D array of float64s. This is pretty much only
// useful for dumping its contents.
func (m Matrix44) Elements() [4][4]float64 {
	return [4][4]float64{
		[4]float64{m.m11, m.m12, m.m13, m.m14},
		[4]float64{m.m21, m.m22, m.m23, m.m24},
		[4]float64{m.m31, m.m32, m.m33, m.m34},
		[4]float64{m.m41, m.m42, m.m43, m.m44},
	}
}

// Inverse returns the inverse of the matrix.
//
// This implementation is stolen from threejs, because I don't fully understand
// it yet. I was expecting it to be rather simpler.
//
// See: https://github.com/mrdoob/three.js/blob/master/src/math/Matrix4.js#L605
//
func (m Matrix44) Inverse() Matrix44 {
	return Matrix44{
		(m.m23 * m.m34 * m.m42) - (m.m24 * m.m33 * m.m42) + (m.m24 * m.m32 * m.m43) - (m.m22 * m.m34 * m.m43) - (m.m23 * m.m32 * m.m44) + (m.m22 * m.m33 * m.m44),
		(m.m14 * m.m33 * m.m42) - (m.m13 * m.m34 * m.m42) - (m.m14 * m.m32 * m.m43) + (m.m12 * m.m34 * m.m43) + (m.m13 * m.m32 * m.m44) - (m.m12 * m.m33 * m.m44),
		(m.m13 * m.m24 * m.m42) - (m.m14 * m.m23 * m.m42) + (m.m14 * m.m22 * m.m43) - (m.m12 * m.m24 * m.m43) - (m.m13 * m.m22 * m.m44) + (m.m12 * m.m23 * m.m44),
		(m.m14 * m.m23 * m.m32) - (m.m13 * m.m24 * m.m32) - (m.m14 * m.m22 * m.m33) + (m.m12 * m.m24 * m.m33) + (m.m13 * m.m22 * m.m34) - (m.m12 * m.m23 * m.m34),
		(m.m24 * m.m33 * m.m41) - (m.m23 * m.m34 * m.m41) - (m.m24 * m.m31 * m.m43) + (m.m21 * m.m34 * m.m43) + (m.m23 * m.m31 * m.m44) - (m.m21 * m.m33 * m.m44),
		(m.m13 * m.m34 * m.m41) - (m.m14 * m.m33 * m.m41) + (m.m14 * m.m31 * m.m43) - (m.m11 * m.m34 * m.m43) - (m.m13 * m.m31 * m.m44) + (m.m11 * m.m33 * m.m44),
		(m.m14 * m.m23 * m.m41) - (m.m13 * m.m24 * m.m41) - (m.m14 * m.m21 * m.m43) + (m.m11 * m.m24 * m.m43) + (m.m13 * m.m21 * m.m44) - (m.m11 * m.m23 * m.m44),
		(m.m13 * m.m24 * m.m31) - (m.m14 * m.m23 * m.m31) + (m.m14 * m.m21 * m.m33) - (m.m11 * m.m24 * m.m33) - (m.m13 * m.m21 * m.m34) + (m.m11 * m.m23 * m.m34),
		(m.m22 * m.m34 * m.m41) - (m.m24 * m.m32 * m.m41) + (m.m24 * m.m31 * m.m42) - (m.m21 * m.m34 * m.m42) - (m.m22 * m.m31 * m.m44) + (m.m21 * m.m32 * m.m44),
		(m.m14 * m.m32 * m.m41) - (m.m12 * m.m34 * m.m41) - (m.m14 * m.m31 * m.m42) + (m.m11 * m.m34 * m.m42) + (m.m12 * m.m31 * m.m44) - (m.m11 * m.m32 * m.m44),
		(m.m12 * m.m24 * m.m41) - (m.m14 * m.m22 * m.m41) + (m.m14 * m.m21 * m.m42) - (m.m11 * m.m24 * m.m42) - (m.m12 * m.m21 * m.m44) + (m.m11 * m.m22 * m.m44),
		(m.m14 * m.m22 * m.m31) - (m.m12 * m.m24 * m.m31) - (m.m14 * m.m21 * m.m32) + (m.m11 * m.m24 * m.m32) + (m.m12 * m.m21 * m.m34) - (m.m11 * m.m22 * m.m34),
		(m.m23 * m.m32 * m.m41) - (m.m22 * m.m33 * m.m41) - (m.m23 * m.m31 * m.m42) + (m.m21 * m.m33 * m.m42) + (m.m22 * m.m31 * m.m43) - (m.m21 * m.m32 * m.m43),
		(m.m12 * m.m33 * m.m41) - (m.m13 * m.m32 * m.m41) + (m.m13 * m.m31 * m.m42) - (m.m11 * m.m33 * m.m42) - (m.m12 * m.m31 * m.m43) + (m.m11 * m.m32 * m.m43),
		(m.m13 * m.m22 * m.m41) - (m.m12 * m.m23 * m.m41) - (m.m13 * m.m21 * m.m42) + (m.m11 * m.m23 * m.m42) + (m.m12 * m.m21 * m.m43) - (m.m11 * m.m22 * m.m43),
		(m.m12 * m.m23 * m.m31) - (m.m13 * m.m22 * m.m31) + (m.m13 * m.m21 * m.m32) - (m.m11 * m.m23 * m.m32) - (m.m12 * m.m21 * m.m33) + (m.m11 * m.m22 * m.m33),
	}
}

// MultiplyMatrices multiplies two 4x4 matrices together, and returns a pointer
// to the result.
func MultiplyMatrices(a Matrix44, b Matrix44) *Matrix44 {
	return &Matrix44{
		(a.m11 * b.m11) + (a.m12 * b.m21) + (a.m13 * b.m31) + (a.m14 * b.m41),
		(a.m11 * b.m12) + (a.m12 * b.m22) + (a.m13 * b.m32) + (a.m14 * b.m42),
		(a.m11 * b.m13) + (a.m12 * b.m23) + (a.m13 * b.m33) + (a.m14 * b.m43),
		(a.m11 * b.m14) + (a.m12 * b.m24) + (a.m13 * b.m34) + (a.m14 * b.m44),
		(a.m21 * b.m11) + (a.m22 * b.m21) + (a.m23 * b.m31) + (a.m24 * b.m41),
		(a.m21 * b.m12) + (a.m22 * b.m22) + (a.m23 * b.m32) + (a.m24 * b.m42),
		(a.m21 * b.m13) + (a.m22 * b.m23) + (a.m23 * b.m33) + (a.m24 * b.m43),
		(a.m21 * b.m14) + (a.m22 * b.m24) + (a.m23 * b.m34) + (a.m24 * b.m44),
		(a.m31 * b.m11) + (a.m32 * b.m21) + (a.m33 * b.m31) + (a.m34 * b.m41),
		(a.m31 * b.m12) + (a.m32 * b.m22) + (a.m33 * b.m32) + (a.m34 * b.m42),
		(a.m31 * b.m13) + (a.m32 * b.m23) + (a.m33 * b.m33) + (a.m34 * b.m43),
		(a.m31 * b.m14) + (a.m32 * b.m24) + (a.m33 * b.m34) + (a.m34 * b.m44),
		(a.m41 * b.m11) + (a.m42 * b.m21) + (a.m43 * b.m31) + (a.m44 * b.m41),
		(a.m41 * b.m12) + (a.m42 * b.m22) + (a.m43 * b.m32) + (a.m44 * b.m42),
		(a.m41 * b.m13) + (a.m42 * b.m23) + (a.m43 * b.m33) + (a.m44 * b.m43),
		(a.m41 * b.m14) + (a.m42 * b.m24) + (a.m43 * b.m34) + (a.m44 * b.m44),
	}
}

// SetRotation sets the rotation of a matrix to that of the given Euler Angle.
// TODO: Should this return a new matrix instead?
// TODO: This is actually kind of a constructor?
func (m *Matrix44) SetRotation(ea EulerAngles) {

	// precompute
	cy := math.Cos(ea.Heading)
	sy := math.Sin(ea.Heading)
	cx := math.Cos(ea.Pitch)
	sx := math.Sin(ea.Pitch)
	cz := math.Cos(ea.Bank)
	sz := math.Sin(ea.Bank)

	// perform intense snafucation
	m.m11 = cy * cz
	m.m21 = -cy * sz
	m.m31 = sy
	m.m14 = 0
	m.m12 = (cx * sz) + ((sx * cz) * sy)
	m.m22 = (cx * cz) - ((sx * sz) * sy)
	m.m32 = -sx * cy
	m.m24 = 0
	m.m13 = (sx * sz) - ((cx * cz) * sy)
	m.m23 = (sx * cz) + ((cx * sz) * sy)
	m.m33 = cx * cy
	m.m34 = 0
	m.m41 = 0
	m.m42 = 0
	m.m43 = 0
	m.m44 = 1
}

// SetTranslation sets the translation of a matrix by overwriting the fourth
// row. Other cells are left alone.
func (m *Matrix44) SetTranslation(v Vector3) {
	m.m41 = v.X
	m.m42 = v.Y
	m.m43 = v.Z
}
