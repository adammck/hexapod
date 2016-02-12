package math3d

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAdd(t *testing.T) {
	type eg struct {
		recv Pose
		arg  Pose
		out  Pose
	}

	examples := []eg{
		{
			recv: Pose{Vector3{+0, +0, +0}, 0},
			arg:  Pose{Vector3{+0, +0, +0}, 0},
			out:  Pose{Vector3{+0, +0, +0}, 0},
		},
		{
			recv: Pose{Vector3{+0, +0, +0}, 90},
			arg:  Pose{Vector3{+1, +0, +0}, 0},
			out:  Pose{Vector3{+0, +0, -1}, 90},
		},
		{
			recv: Pose{Vector3{+0, +0, +0}, 180},
			arg:  Pose{Vector3{+1, +0, +0}, 0},
			out:  Pose{Vector3{-1, +0, +0}, 180},
		},
		{
			recv: Pose{Vector3{+0, +0, +0}, 270},
			arg:  Pose{Vector3{+1, +0, +0}, 0},
			out:  Pose{Vector3{+0, +0, +1}, 270},
		},
		{
			recv: Pose{Vector3{+9, +1, +9}, 90},
			arg:  Pose{Vector3{+1, +0, +0}, 90},
			out:  Pose{Vector3{+9, +1, +8}, 180},
		},
	}

	for i, x := range examples {
		act := x.recv.Add(x.arg)
		assert.InDelta(t, act.Position.X, x.out.Position.X, 0.01, "expected example %d:X to be %0.2f, but was %0.2f", i+1, x.out.Position.X, act.Position.X)
		assert.InDelta(t, act.Position.Y, x.out.Position.Y, 0.01, "expected example %d:Y to be %0.2f, but was %0.2f", i+1, x.out.Position.Y, act.Position.Y)
		assert.InDelta(t, act.Position.Z, x.out.Position.Z, 0.01, "expected example %d:Z to be %0.2f, but was %0.2f", i+1, x.out.Position.Z, act.Position.Z)
		assert.InDelta(t, act.Heading, x.out.Heading, 0.01, "expected example %d:H to be %0.2f, but was %0.2f", i+1, x.out.Heading, act.Heading)
	}
}

func TestOut(t *testing.T) {
	type eg struct {
		recv Pose
		arg  Pose
		out  Pose
	}

	examples := []eg{
		{
			recv: Pose{Vector3{+0, +0, -8}, 90},
			arg:  Pose{Vector3{+0, +0, -9}, 0},
			out:  Pose{Vector3{+1, +0, +0}, -90},
		},
	}

	for i, x := range examples {
		act := x.recv.Out(x.arg)
		assert.InDelta(t, act.Position.X, x.out.Position.X, 0.01, "expected example %d:X to be %0.2f, but was %0.2f", i+1, x.out.Position.X, act.Position.X)
		assert.InDelta(t, act.Position.Y, x.out.Position.Y, 0.01, "expected example %d:Y to be %0.2f, but was %0.2f", i+1, x.out.Position.Y, act.Position.Y)
		assert.InDelta(t, act.Position.Z, x.out.Position.Z, 0.01, "expected example %d:Z to be %0.2f, but was %0.2f", i+1, x.out.Position.Z, act.Position.Z)
		assert.InDelta(t, act.Heading, x.out.Heading, 0.01, "expected example %d:H to be %0.2f, but was %0.2f", i+1, x.out.Heading, act.Heading)
	}
}
