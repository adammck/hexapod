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
			recv: Pose{Vector3{0, 0, 0}, 0},
			arg:  Pose{Vector3{0, 0, 0}, 0},
			out:  Pose{Vector3{0, 0, 0}, 0},
		},
		{
			recv: Pose{Vector3{0, 0, 0}, 90},
			arg:  Pose{Vector3{1, 0, 0}, 0},
			out:  Pose{Vector3{0, 0, -1}, 90},
		},
		{
			recv: Pose{Vector3{0, 0, 0}, 180},
			arg:  Pose{Vector3{1, 0, 0}, 0},
			out:  Pose{Vector3{-1, 0, 0}, 180},
		},
		{
			recv: Pose{Vector3{0, 0, 0}, 270},
			arg:  Pose{Vector3{1, 0, 0}, 0},
			out:  Pose{Vector3{0, 0, 1}, 270},
		},
	}

	for i, x := range examples {
		act := x.recv.Add(x.arg)
		assert.InDelta(t, act.Position.X, x.out.Position.X, 0.01, "example %d/X failed", i+1)
		assert.InDelta(t, act.Position.Y, x.out.Position.Y, 0.01, "example %d/Y failed", i+1)
		assert.InDelta(t, act.Position.Z, x.out.Position.Z, 0.01, "example %d/Z failed", i+1)
		assert.InDelta(t, act.Heading, x.out.Heading, 0.01, "example %d/H failed", i+1)
	}
}
