package math3d

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMagnitude(t *testing.T) {
	type eg struct {
		input Vector3
		exp   float64
	}

	examples := []eg{
		{Vector3{X: 0, Y: 0, Z: 0}, 0},
		{Vector3{X: 1, Y: 1, Z: 1}, 1.732050808},
		{Vector3{X: 1, Y: 2, Z: 3}, 3.741657387},
		{Vector3{X: 4, Y: 5, Z: 6}, 8.774964387},
	}

	for _, x := range examples {
		assert.InDelta(t, x.exp, x.input.Magnitude(), 0.01)
	}
}

func TestDistance(t *testing.T) {
	type eg struct {
		recv Vector3
		arg  Vector3
		out  float64
	}

	examples := []eg{
		{Vector3{X: 1, Y: 1, Z: 1}, Vector3{X: 1, Y: 1, Z: 1}, 0},
		{Vector3{X: 1, Y: 1, Z: 1}, Vector3{X: 2, Y: 2, Z: 2}, 1.732050808},
	}

	for _, x := range examples {
		assert.InDelta(t, x.out, x.recv.Distance(x.arg), 0.01)
	}
}

func TestUnit(t *testing.T) {
	type eg struct {
		in  Vector3
		out Vector3
	}

	examples := []eg{
		{Vector3{X: 0, Y: 0, Z: 0}, ZeroVector3},
		{Vector3{X: 1, Y: 1, Z: 1}, Vector3{X: 0.5773502691896258, Y: 0.5773502691896258, Z: 0.5773502691896258}},
		{Vector3{X: 2, Y: 2, Z: 2}, Vector3{X: 0.5773502691896258, Y: 0.5773502691896258, Z: 0.5773502691896258}},
	}

	for _, x := range examples {
		assert.Equal(t, x.out, x.in.Unit())
	}
}

func TestSubtract(t *testing.T) {
	v1 := Vector3{X: 1, Y: 2, Z: 3}
	v2 := Vector3{X: 4, Y: 5, Z: 6}

	vAct := v2.Subtract(v1)
	vExp := Vector3{X: 3, Y: 3, Z: 3}
	assert.Equal(t, vExp, vAct)
}

func TestMultiplyByScalar(t *testing.T) {
	v := Vector3{X: 1, Y: 2, Z: 3}

	vAct := v.MultiplyByScalar(0.5)
	vExp := Vector3{X: 0.5, Y: 1, Z: 1.5}
	assert.Equal(t, vExp, vAct)

	vAct = v.MultiplyByScalar(2)
	vExp = Vector3{X: 2, Y: 4, Z: 6}
	assert.Equal(t, vExp, vAct)
}
