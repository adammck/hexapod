package legs

import (
	"testing"
	"github.com/adammck/hexapod/math3d"
)

func TestLegMatrix(t *testing.T) {

	type example struct {
		legOrigin  Vector3
		legHeading float64
		vec        Vector3
		exp        Vector3
	}

	// TODO (adammck): Moar
	data := []example{
		example{ZeroVector3, 0, Vector3{1, 1, 1}, Vector3{1, 1, 1}},
		example{ZeroVector3, 180.0, Vector3{10, 20, 30}, Vector3{-10, 20, -30}},
		example{Vector3{10, 20, 30}, 90.0, Vector3{1, 2, 3}, Vector3{13, 22, 29}},
	}

	for i, eg := range data {
		leg := Leg{
			Origin: &eg.legOrigin,
			Angle:  eg.legHeading,
			Name:   "whatever",
		}

		// Check that the result was very close to what was expected. We're
		// dealing with floats, so can't be exact.
		actual := eg.vec.MultiplyByMatrix44(leg.Matrix())
		if actual.Distance(eg.exp) > 0.000001 {
			t.Errorf("Example #%d: got %s, expected: %s", i+1, actual, eg.exp)
		}
	}
}
