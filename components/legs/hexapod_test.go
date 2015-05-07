package legs

import (
	"github.com/adammck/hexapod/math3d"
	"testing"
)

type Vector3 math3d.Vector3

type eg struct {
	pos Vector3 // position
	rot float64 // rotation (heading)
	vec Vector3 // input
	exp Vector3 // expected result
}

func TestWorld(t *testing.T) {
	data := []eg{
		eg{Vector3{00, 00, 00}, 0.0, Vector3{0, 0, 0}, Vector3{0, 0, 00}},
		eg{Vector3{00, 00, 10}, 0.0, Vector3{0, 0, 0}, Vector3{0, 0, 10}},
		eg{Vector3{00, 00, 20}, 0.0, Vector3{0, 0, 0}, Vector3{0, 0, 20}},
		eg{Vector3{00, 00, 30}, 0.0, Vector3{0, 0, 0}, Vector3{0, 0, 30}},
	}

	for i, eg := range data {
		h := Hexapod{
			Position: eg.pos,
			Rotation: eg.rot,
		}

		actual := eg.vec.MultiplyByMatrix44(h.World())
		if actual.Distance(eg.exp) > 0.000001 {
			t.Errorf("Example #%d: got %s, expected: %s", i+1, actual, eg.exp)
		}
	}
}

func TestLocal(t *testing.T) {

	data := []eg{
		eg{Vector3{00, 00, 00}, 0.0, Vector3{10, 20, 30}, Vector3{10, 20, 30}},
		eg{Vector3{00, 00, 10}, 0.0, Vector3{10, 20, 30}, Vector3{10, 20, 20}},
		eg{Vector3{00, 00, 20}, 0.0, Vector3{10, 20, 30}, Vector3{10, 20, 10}},
		eg{Vector3{00, 00, 30}, 0.0, Vector3{10, 20, 30}, Vector3{10, 20, 00}},
	}

	for i, eg := range data {
		h := Hexapod{
			Position: eg.pos,
			Rotation: eg.rot,
		}

		actual := eg.vec.MultiplyByMatrix44(h.Local())
		if actual.Distance(eg.exp) > 0.000001 {
			t.Errorf("Example #%d: got %s, expected: %s", i+1, actual, eg.exp)
		}
	}
}
